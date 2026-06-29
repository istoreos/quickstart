package disklifecycle

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
)

var waitRefresh = func(d time.Duration) {
	time.Sleep(d)
}

type SnapshotReader interface {
	ReadAll(ctx context.Context) ([]DiskSnapshot, error)
	ReadDisk(ctx context.Context, name string) (*DiskSnapshot, error)
	ReadDiskIncludeFree(ctx context.Context, name string) (*DiskSnapshot, error)
}

type CommandStore interface {
	Mount(devicePath string, mountPoint string) error
	UnMount(devicePath string) error
	Unmount(mountPoint string) error
	Erase(devicePath string) error
	MakePart(devicePath string) error
	FixGPTTable(devicePath string) error
	MakePartRange(devicePath string, typeOrName string, alignedStart uint64, alignedEnd uint64) error
	Ext4Partition(devicePath string) error
	AddFstab(uuid string, path string, skipExisted bool) (string, error)
	CommitFstab() error
	CommitFstabAndBlockMount() error
}

type MountPointGenerator interface {
	Generate(name string) string
}

type Service struct {
	snapshotReader      SnapshotReader
	commandStore        CommandStore
	mountPointGenerator MountPointGenerator
}

func NewService(snapshotReader SnapshotReader, commandStore CommandStore, mountPointGenerator MountPointGenerator) *Service {
	return &Service{
		snapshotReader:      snapshotReader,
		commandStore:        commandStore,
		mountPointGenerator: mountPointGenerator,
	}
}

func (svc *Service) MountPartition(ctx context.Context, input PartitionMountInput) (*models.PartitionInfo, error) {
	if len(input.MountPoint) < 2 || !strings.HasPrefix(input.MountPoint, "/") {
		return nil, errors.New("挂载点必须以/开头且至少一级目录，例如/mnt/mydata")
	}

	disks, err := svc.snapshotReader.ReadAll(ctx)
	if err != nil {
		return nil, err
	}
	_, target := findPartitionByIdentity(disks, input.UUID, input.Path)
	if target == nil {
		return nil, errors.New("partition not found")
	}
	if checkMountPoint(target.MountPoint) {
		if err := svc.commandStore.Unmount(target.MountPoint); err != nil {
			return nil, err
		}
	}

	disks, err = svc.snapshotReader.ReadAll(ctx)
	if err != nil {
		return nil, err
	}
	_, target = findPartitionByIdentity(disks, input.UUID, input.Path)
	if target == nil || checkMountPoint(target.MountPoint) {
		return nil, errors.New("partition format error ")
	}

	if err := svc.commandStore.Mount(target.Path, input.MountPoint); err != nil {
		return nil, err
	}
	if _, err := svc.commandStore.AddFstab(target.UUID, input.MountPoint, false); err != nil {
		return nil, err
	}
	if err := svc.commandStore.CommitFstabAndBlockMount(); err != nil {
		return nil, err
	}

	model := partitionModel(*target)
	model.MountPoint = input.MountPoint
	return model, nil
}

func (svc *Service) GenerateMountPoint(ctx context.Context, path string) (string, error) {
	return svc.buildGeneratedMountPoint(path)
}

func (svc *Service) FormatByDevicePath(ctx context.Context, input FormatByDevicePathInput) (*models.PartitionInfo, error) {
	disks, err := svc.snapshotReader.ReadAll(ctx)
	if err != nil {
		return nil, err
	}

	isWholeDisk := isWholeDiskTarget(disks, input.DevicePath)
	var target *PartitionSnapshot
	for diskIdx := range disks {
		for partitionIdx := range disks[diskIdx].Partitions {
			partition := &disks[diskIdx].Partitions[partitionIdx]
			if partition.Path == input.DevicePath {
				target = partition
			}
		}
	}

	if target == nil {
		if !isWholeDisk {
			return nil, errors.New("partition not found: " + input.DevicePath)
		}
	} else if checkMountPoint(target.MountPoint) {
		if err := svc.commandStore.Unmount(target.MountPoint); err != nil {
			return nil, err
		}
	}

	if err := svc.commandStore.Ext4Partition(input.DevicePath); err != nil {
		return nil, err
	}
	waitRefresh(3 * time.Second)

	disks, err = svc.snapshotReader.ReadAll(ctx)
	if err != nil {
		return nil, err
	}
	for diskIdx := range disks {
		for partitionIdx := range disks[diskIdx].Partitions {
			partition := &disks[diskIdx].Partitions[partitionIdx]
			if !checkMountPoint(partition.MountPoint) && partition.Path == input.DevicePath {
				mountPoint, err := svc.buildGeneratedMountPoint(partition.Path)
				if err != nil {
					return nil, err
				}
				if err := svc.commandStore.Mount(partition.Path, mountPoint); err != nil {
					return nil, err
				}
				if _, err := svc.commandStore.AddFstab(partition.UUID, mountPoint, false); err != nil {
					return nil, err
				}
				if err := svc.commandStore.CommitFstabAndBlockMount(); err != nil {
					return nil, err
				}
				model := partitionModel(*partition)
				model.MountPoint = mountPoint
				return model, nil
			}
		}
	}
	return nil, errors.New("partition format error")
}

func (svc *Service) InitDisk(ctx context.Context, input InitInput) (*models.NasDiskInfo, error) {
	if len(input.Name) == 0 || len(input.Path) == 0 {
		return nil, errors.New("param missing")
	}

	disk, err := svc.snapshotReader.ReadDisk(ctx, input.Name)
	if err != nil || disk == nil || disk.Name != input.Name {
		return nil, errors.New("disk not found")
	}
	for _, partition := range disk.Partitions {
		if checkMountPoint(partition.MountPoint) {
			if err := svc.commandStore.Unmount(partition.MountPoint); err != nil {
				return nil, err
			}
		}
	}

	if err := svc.commandStore.Erase(input.Path); err != nil {
		return nil, err
	}
	if usesDirectDeviceFormat(input.Name) {
		if err := svc.commandStore.Ext4Partition(input.Path); err != nil {
			return nil, err
		}
	} else {
		if err := svc.commandStore.MakePart(input.Path); err != nil {
			return nil, err
		}
		waitRefresh(3 * time.Millisecond)
		disk, err = svc.snapshotReader.ReadDisk(ctx, input.Name)
		if err != nil || disk == nil {
			return nil, errors.New("disk not found")
		}
		for _, partition := range disk.Partitions {
			_ = svc.commandStore.UnMount(partition.Path)
			if err := svc.commandStore.Ext4Partition(partition.Path); err != nil {
				return nil, err
			}
		}
	}

	waitRefresh(3 * time.Millisecond)
	disk, err = svc.snapshotReader.ReadDisk(ctx, input.Name)
	if err != nil || disk == nil {
		return nil, errors.New("disk not found")
	}
	result := diskModel(*disk)
	for _, partition := range result.Childrens {
		if !checkMountPoint(partition.MountPoint) {
			mountPoint, err := svc.buildGeneratedMountPoint(partition.Path)
			if err != nil {
				return nil, err
			}
			if err := svc.commandStore.Mount(partition.Path, mountPoint); err != nil {
				return nil, err
			}
			if _, err := svc.commandStore.AddFstab(partition.UUID, mountPoint, false); err != nil {
				return nil, err
			}
			partition.MountPoint = mountPoint
		}
		partition.SecStart = 0
		partition.SecEnd = 0
	}
	if err := svc.commandStore.CommitFstabAndBlockMount(); err != nil {
		return nil, err
	}
	return result, nil
}

func (svc *Service) InitDiskRest(ctx context.Context, input InitRestInput) (*models.NasDiskInfo, error) {
	if len(input.Name) == 0 || len(input.Path) == 0 {
		return nil, errors.New("init rest disk, param missing")
	}

	disk, err := svc.snapshotReader.ReadDiskIncludeFree(ctx, input.Name)
	if err != nil || disk == nil || disk.Name != input.Name {
		return nil, errors.New("disk not found")
	}
	last := findLastPartition(*disk)
	if last == nil {
		return nil, errors.New("partition table is corrupted")
	}
	if last.Filesystem != "Free Space" {
		return nil, errors.New("磁盘末尾没有足够的空闲空间")
	}

	typeOrName := "primary"
	alignedStart := (last.SecStart + 2047) / 2048 * 2048
	alignedEnd := (last.SecEnd - 512) / 2048 * 2048
	if disk.PartLabelType == "GPT" {
		if err := svc.commandStore.FixGPTTable(disk.Path); err != nil {
			return nil, err
		}
		typeOrName = "UserData"
	}
	if err := svc.commandStore.MakePartRange(disk.Path, typeOrName, alignedStart, alignedEnd); err != nil {
		return nil, err
	}
	waitRefresh(2 * time.Millisecond)

	disk, err = svc.snapshotReader.ReadDisk(ctx, input.Name)
	if err != nil || disk == nil {
		return nil, errors.New("disk not found")
	}
	partIdx := len(disk.Partitions) - 1
	if partIdx < 0 {
		return nil, errors.New("disk not found")
	}
	last = &disk.Partitions[partIdx]
	fsExisted := true
	if last.Filesystem == "No FileSystem" {
		if err := svc.commandStore.Ext4Partition(last.Path); err != nil {
			return nil, err
		}
		waitRefresh(2 * time.Millisecond)
		disk, err = svc.snapshotReader.ReadDisk(ctx, input.Name)
		if err != nil || disk == nil {
			return nil, errors.New("ex4格式化后未能获取到磁盘信息")
		}
		partIdx = len(disk.Partitions) - 1
		if partIdx < 0 {
			return nil, errors.New("ex4格式化后未能获取到磁盘信息")
		}
		last = &disk.Partitions[partIdx]
		if last.Path == "" {
			return nil, errors.New("ex4格式化后未能获取到磁盘信息")
		}
		fsExisted = false
	}

	result := diskModel(*disk)
	lastModel := result.Childrens[partIdx]
	if len(lastModel.Path) > 0 {
		mountPoint, err := svc.buildGeneratedMountPoint(lastModel.Path)
		if err != nil {
			return nil, err
		}
		_ = svc.commandStore.UnMount(lastModel.Path)
		mountPoint, err = svc.commandStore.AddFstab(lastModel.UUID, mountPoint, fsExisted)
		if err != nil {
			return nil, err
		}
		if err := svc.commandStore.Mount(lastModel.Path, mountPoint); err != nil {
			_ = svc.commandStore.CommitFstab()
			return nil, err
		}
		lastModel.MountPoint = mountPoint
		if err := svc.commandStore.CommitFstabAndBlockMount(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (svc *Service) buildGeneratedMountPoint(devicePath string) (string, error) {
	paths := strings.Split(devicePath, "/")
	if len(paths) == 0 {
		return "", errors.New("mountPoint生成失败")
	}
	name := svc.mountPointGenerator.Generate(paths[len(paths)-1])
	if name == "" {
		return "", errors.New("mountPoint生成失败")
	}
	return "/mnt/" + name, nil
}

func partitionModel(snapshot PartitionSnapshot) *models.PartitionInfo {
	return &models.PartitionInfo{
		Filesystem: snapshot.Filesystem,
		MountPoint: snapshot.MountPoint,
		Name:       snapshot.Name,
		Path:       snapshot.Path,
		SecEnd:     snapshot.SecEnd,
		SecStart:   snapshot.SecStart,
		UUID:       snapshot.UUID,
	}
}

func diskModel(snapshot DiskSnapshot) *models.NasDiskInfo {
	model := &models.NasDiskInfo{
		Name:          snapshot.Name,
		PartLabelType: snapshot.PartLabelType,
		Path:          snapshot.Path,
		Childrens:     make([]*models.PartitionInfo, 0, len(snapshot.Partitions)),
	}
	for _, partition := range snapshot.Partitions {
		model.Childrens = append(model.Childrens, partitionModel(partition))
	}
	return model
}

func checkMountPoint(mountPoint string) bool {
	if mountPoint == "" || mountPoint == "-" {
		return false
	}
	return true
}

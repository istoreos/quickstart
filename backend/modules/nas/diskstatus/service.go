package diskstatus

import (
	"context"
	"strconv"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/nas/diskinventory"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type InventoryReader interface {
	List(ctx context.Context) ([]*diskinventory.DiskInfo, error)
}

type PartitionMarker interface {
	Mark(ctx context.Context, disk *models.NasDiskInfo, partition *models.PartitionInfo)
}

type RAIDMemberReader interface {
	RAIDMember(ctx context.Context, diskName string) string
}

type SMARTReader interface {
	Config(ctx context.Context) (*models.SmartConfigResponseResult, error)
	Health(ctx context.Context, diskName string) (string, error)
}

type Service struct {
	inventoryReader InventoryReader
	partitionMarker PartitionMarker
	raidReader      RAIDMemberReader
	smartReader     SMARTReader
}

func NewService(
	inventoryReader InventoryReader,
	partitionMarker PartitionMarker,
	raidReader RAIDMemberReader,
	smartReader SMARTReader,
) *Service {
	return &Service{
		inventoryReader: inventoryReader,
		partitionMarker: partitionMarker,
		raidReader:      raidReader,
		smartReader:     smartReader,
	}
}

func (svc *Service) List(ctx context.Context) ([]*models.NasDiskInfo, error) {
	blockdevices, err := svc.inventoryReader.List(ctx)
	if err != nil {
		return nil, err
	}

	disks := make([]*models.NasDiskInfo, 0)
	for _, blockdevice := range blockdevices {
		diskInfo, ok := BuildDiskInfo(blockdevice)
		if !ok {
			continue
		}

		partitions := make([]*models.PartitionInfo, 0)
		used := uint64(0)
		total := uint64(0)
		for _, rawPartition := range blockdevice.Children {
			partition, usage := BuildPartitionInfo(rawPartition)
			used += usage.Used
			total += usage.Total

			if len(partition.MountPoint) > 0 && partition.Filesystem != "swap" && svc.partitionMarker != nil {
				svc.partitionMarker.Mark(ctx, diskInfo, partition)
			}

			if !ShouldIncludePartition(rawPartition.Label, partition, usage.Total) {
				continue
			}
			partitions = append(partitions, partition)
		}

		diskSizeByte, _ := strconv.ParseUint(diskInfo.SizeInt, 10, 64)
		raidMember := ""
		if svc.raidReader != nil {
			raidMember = svc.raidReader.RAIDMember(ctx, diskInfo.Name)
		}
		if !ShouldIncludeDisk(diskInfo, len(partitions), diskSizeByte, raidMember) {
			continue
		}

		ApplyDiskUsage(diskInfo, used, total)
		if diskInfo.IsSystemRoot {
			if partition := BuildFreeSpacePartition(diskSizeByte, total); partition != nil {
				partitions = append(partitions, partition)
			}
		}

		diskInfo.Childrens = partitions
		if svc.smartReader != nil {
			config, _ := svc.smartReader.Config(ctx)
			if ShouldCheckSMART(diskInfo.Path, config) {
				health, _ := svc.smartReader.Health(ctx, diskInfo.Name)
				ApplySMARTHealth(diskInfo, health)
			}
		}
		disks = append(disks, diskInfo)
	}

	return disks, nil
}

type UsageSummary struct {
	Used  uint64
	Total uint64
}

func BuildDiskInfo(disk *diskinventory.DiskInfo) (*models.NasDiskInfo, bool) {
	if disk.Root.Type != "disk" {
		return nil, false
	}
	diskInfo := &models.NasDiskInfo{
		Name:          disk.Root.Name,
		Path:          disk.Root.Path,
		Size:          disk.Root.SizeStr,
		SizeInt:       disk.Root.SizeIntStr,
		VenderModel:   disk.Root.DisplayName,
		PartLabelType: disk.Root.PType,
		TranName:      disk.Root.TranName,
	}
	if len(disk.Root.TranName) > 0 {
		diskInfo.IsExternalDisk = true
	}
	return diskInfo, true
}

func BuildPartitionInfo(partInfo *diskinventory.DiskInfoChildren) (*models.PartitionInfo, UsageSummary) {
	sizeByte := partInfo.SizeInt
	usedSizeByte := partInfo.Fsused
	usageInt := uint32(0)
	if sizeByte > 0 {
		usageInt = uint32(usedSizeByte * 100 / sizeByte)
	}

	point := &models.PartitionInfo{
		Filesystem: partInfo.FSType,
		MountPoint: partInfo.Mountpoint,
		Name:       partInfo.Name,
		Path:       partInfo.Path,
		UUID:       partInfo.UUID,
		Total:      utils.ByteCountBinary(sizeByte),
		SizeInt:    strconv.FormatUint(sizeByte, 10),
		Used:       utils.ByteCountBinary(usedSizeByte),
		Usage:      usageInt,
	}
	if point.Filesystem == "" {
		if point.MountPoint == "" {
			point.Filesystem = "No FileSystem"
		} else {
			point.Filesystem = "unknown"
		}
	}

	return point, UsageSummary{Used: usedSizeByte, Total: sizeByte}
}

func ShouldIncludePartition(label string, point *models.PartitionInfo, sizeByte uint64) bool {
	if label == "kernel" || point.MountPoint == "/rom" {
		return false
	}
	if !point.IsSystemRoot && sizeByte < 64*1024*1024 {
		return false
	}
	return true
}

func ApplyDiskUsage(diskInfo *models.NasDiskInfo, used uint64, total uint64) {
	diskInfo.Used = utils.ByteCountBinary(used)
	diskInfo.UsedInt = strconv.FormatInt(int64(used), 10)
	diskInfo.Total = utils.ByteCountBinary(total)
	if total > 0 {
		diskInfo.Usage = uint32(used * 100 / total)
	}
}

func BuildFreeSpacePartition(diskSizeByte uint64, total uint64) *models.PartitionInfo {
	if diskSizeByte <= total || diskSizeByte-total <= 1024*1024*1024 {
		return nil
	}
	freeBytes := diskSizeByte - total
	return &models.PartitionInfo{
		Name:       "Free Space",
		Filesystem: "Free Space",
		Total:      utils.ByteCountBinary(freeBytes),
		SizeInt:    strconv.FormatUint(freeBytes, 10),
	}
}

func MarkSystemAndDocker(disk *models.NasDiskInfo, partition *models.PartitionInfo, rootPaths []string, dockerDevicePath string) {
	for _, rootPath := range rootPaths {
		if partition.Path == rootPath {
			partition.IsSystemRoot = true
			disk.IsSystemRoot = true
			break
		}
	}

	if partition.Path == dockerDevicePath {
		partition.IsDockerRoot = true
		disk.IsDockerRoot = true
	} else if partition.MountPoint == "/rom" && dockerDevicePath == "/dev/loop0" {
		disk.IsDockerRoot = true
	}
}

func ShouldIncludeDisk(diskInfo *models.NasDiskInfo, visiblePartitionCount int, diskSizeByte uint64, raidMember string) bool {
	if diskInfo.IsSystemRoot && diskInfo.PartLabelType == "LOOP" && visiblePartitionCount < 1 {
		return false
	}
	if !diskInfo.IsSystemRoot && (diskSizeByte < 1000*1000*1000 || raidMember != "") {
		return false
	}
	return true
}

func ShouldCheckSMART(diskPath string, config *models.SmartConfigResponseResult) bool {
	if config == nil || config.Global == nil || !config.Global.Enable {
		return false
	}
	for _, device := range config.Devices {
		if device != nil && device.DevicePath == diskPath {
			return true
		}
	}
	return false
}

func ApplySMARTHealth(diskInfo *models.NasDiskInfo, health string) {
	if health != "PASSED" {
		diskInfo.SmartWarning = true
	}
}

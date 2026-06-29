package writeflow

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/linkease/quick-start/istore-backend/modules/raid/inventory"
	"github.com/linkease/quick-start/istore-backend/modules/raid/writecommands"
)

type Partition struct {
	MountPoint string
	Path       string
	IsRaidOn   bool
}

type Disk struct {
	Path       string
	Partitions []Partition
}

type CreateInput struct {
	Level       string
	DevicePaths []string
}

type DeleteInput struct {
	Path      string
	MountPath string
	Members   []string
}

type MemberInput struct {
	Path       string
	MemberPath string
}

type RecoverInput struct {
	Path               string
	MemberPath         string
	CheckRaidPartition bool
}

type Store interface {
	ReadDisk(ctx context.Context, name string) (*Disk, error)
	Unmount(ctx context.Context, mountPoint string) error
	Erase(ctx context.Context, path string) error
	MakeRaidPart(ctx context.Context, path string) error
	WaitAfterPartition(ctx context.Context)
	WaitAfterCreate(ctx context.Context)
	FindFreeMd(min int) int
	RunCreate(ctx context.Context, command string) error
	CleanupCreatedDevice(ctx context.Context, path string)
	GenerateMdadmConfig(ctx context.Context) error
	UnmountMountPath(ctx context.Context, mountPath string) error
	RunDelete(ctx context.Context, commands []string) error
	RemoveDeletedDevice(ctx context.Context, path string)
	ActiveDeviceCount(ctx context.Context, path string) uint64
	RunAdd(ctx context.Context, command string) (string, error)
	Grow(ctx context.Context, command string)
	RunRemove(ctx context.Context, commands []string) error
	RunRecover(ctx context.Context, commands []string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Create(ctx context.Context, input CreateInput) (string, error) {
	level := writecommands.NormalizeLevel(input.Level)
	if err := writecommands.ValidateMemberCount(level, len(input.DevicePaths)); err != nil {
		return "", err
	}

	diskPartPaths := make([]string, 0, len(input.DevicePaths))
	for _, devicePath := range input.DevicePaths {
		name := deviceNameFromPath(devicePath)
		disk, err := svc.store.ReadDisk(ctx, name)
		if err != nil {
			return "", err
		}
		for _, part := range disk.Partitions {
			if strings.HasPrefix(part.MountPoint, "Raid Member:") {
				return "", errors.New(part.MountPoint + " already found")
			}
			if len(part.MountPoint) > 0 && part.MountPoint != "" {
				if err := svc.store.Unmount(ctx, part.MountPoint); err != nil {
					return "", err
				}
			}
		}
		if err := svc.store.Erase(ctx, disk.Path); err != nil {
			return "", err
		}
		if err := svc.store.MakeRaidPart(ctx, disk.Path); err != nil {
			return "", err
		}
		diskPartPaths = append(diskPartPaths, inventory.DiskToPart(disk.Path, "1"))
	}
	svc.store.WaitAfterPartition(ctx)

	idx := svc.store.FindFreeMd(0)
	if idx == -1 {
		return "", errors.New("生成raid路径失败")
	}
	path := fmt.Sprintf("/dev/md%v", idx)
	command := writecommands.BuildCreateCommand(path, level, diskPartPaths)
	if err := svc.store.RunCreate(ctx, command); err != nil {
		svc.store.CleanupCreatedDevice(ctx, path)
		return "", errors.New("raid创建失败,请重试" + " " + command)
	}
	if err := svc.store.GenerateMdadmConfig(ctx); err != nil {
		return "", err
	}

	svc.store.WaitAfterCreate(ctx)
	return path, nil
}

func (svc *Service) Delete(ctx context.Context, input DeleteInput) error {
	if input.MountPath != "" {
		if err := svc.store.UnmountMountPath(ctx, input.MountPath); err != nil {
			return errors.New("卸载磁盘失败" + err.Error())
		}
	}

	commands := writecommands.BuildDeleteCommands(input.Path, input.Members)
	if err := svc.store.RunDelete(ctx, commands); err != nil {
		return errors.New("raid删除失败" + err.Error())
	}

	svc.store.RemoveDeletedDevice(ctx, input.Path)
	return svc.store.GenerateMdadmConfig(ctx)
}

func (svc *Service) Add(ctx context.Context, input MemberInput) error {
	if err := svc.store.Erase(ctx, input.MemberPath); err != nil {
		return err
	}
	if err := svc.store.MakeRaidPart(ctx, input.MemberPath); err != nil {
		return err
	}
	svc.store.WaitAfterPartition(ctx)

	count := svc.store.ActiveDeviceCount(ctx, input.Path)
	count += 1
	command := writecommands.BuildAddCommand(input.Path, inventory.DiskToPart(input.MemberPath, "1"))
	stderr, err := svc.store.RunAdd(ctx, command)
	if err != nil {
		return errors.New("扩充成员失败 " + stderr)
	}
	svc.store.Grow(ctx, writecommands.BuildGrowCommand(input.Path, count))
	return nil
}

func (svc *Service) Remove(ctx context.Context, input MemberInput) error {
	commands := writecommands.BuildRemoveCommands(input.Path, input.MemberPath)
	if err := svc.store.RunRemove(ctx, commands); err != nil {
		return errors.New("删除成员失败")
	}
	return nil
}

func (svc *Service) Recover(ctx context.Context, input RecoverInput) error {
	memberPath := input.MemberPath
	checkRaidPart := false
	if input.CheckRaidPartition {
		disk, _ := svc.store.ReadDisk(ctx, input.MemberPath)
		if disk != nil && len(disk.Partitions) == 1 {
			part := disk.Partitions[0]
			if part.IsRaidOn {
				memberPath = part.Path
				checkRaidPart = true
			}
		}
	}

	if !checkRaidPart {
		if err := svc.store.Erase(ctx, input.MemberPath); err != nil {
			return err
		}
		if err := svc.store.MakeRaidPart(ctx, input.MemberPath); err != nil {
			return err
		}
		memberPath = inventory.DiskToPart(input.MemberPath, "1")
	}

	svc.store.WaitAfterPartition(ctx)

	commands := writecommands.BuildRecoverCommands(input.Path, memberPath)
	if err := svc.store.RunRecover(ctx, commands); err != nil {
		return errors.New("恢复失败 " + commands[0])
	}
	return nil
}

func deviceNameFromPath(path string) string {
	match := regexp.MustCompile(`/dev/(\w+)`).FindStringSubmatch(path)
	if match == nil {
		return ""
	}
	return match[1]
}

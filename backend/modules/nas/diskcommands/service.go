package diskcommands

import (
	"context"
	"errors"
	"fmt"
)

type Runner interface {
	Run(ctx context.Context, commands []string) error
	OutErr(ctx context.Context, commands []string) (string, string, error)
	Output(ctx context.Context, command string) ([]byte, error)
}

type FstabMount struct {
	Name   string
	UUID   string
	Target string
}

type Service struct {
	runner Runner
}

func NewService(runner Runner) *Service {
	return &Service{runner: runner}
}

func (svc *Service) Mount(ctx context.Context, devicePath string, mountPoint string) error {
	if len(devicePath) == 0 || len(mountPoint) == 0 {
		return errors.New("mount param missing," + "path:" + devicePath + " mountPoint:" + mountPoint)
	}

	commands := []string{
		fmt.Sprintf("test -d '%v' || mkdir -p '%v'", mountPoint, mountPoint),
		fmt.Sprintf("busybox mount '%v' '%v'", devicePath, mountPoint),
	}
	_, errout, err := svc.runner.OutErr(ctx, commands)
	if err != nil {
		return fmt.Errorf("mount '%v' '%v' failed: %v", devicePath, mountPoint, errout)
	}
	return nil
}

func (svc *Service) UnMount(ctx context.Context, devicePath string) error {
	if len(devicePath) == 0 {
		return errors.New("umount param missing," + "path:" + devicePath)
	}

	_, errout, err := svc.runner.OutErr(ctx, []string{fmt.Sprintf("umount '%v'", devicePath)})
	if err != nil {
		return errors.New(fmt.Sprintf("umount '%v' failed: %v", devicePath, errout))
	}
	return nil
}

func (svc *Service) Unmount(ctx context.Context, mountPoint string) error {
	if mountPoint == "-" {
		return nil
	}
	if len(mountPoint) == 0 {
		return errors.New("unmount param missing")
	}
	if mountPoint == "/overlay" {
		return errors.New("cannot unmount overlay")
	}

	_, errout, err := svc.runner.OutErr(ctx, []string{fmt.Sprintf("umount '%v'", mountPoint)})
	if err != nil {
		return fmt.Errorf("umount '%v' failed: %v", mountPoint, errout)
	}
	return nil
}

func (svc *Service) Erase(ctx context.Context, device string) error {
	if device == "" {
		return errors.New("param missing")
	}

	commands := []string{
		fmt.Sprintf("dd if=/dev/zero of=%v bs=4096 count=8192", device),
		fmt.Sprintf("parted -a optimal -s %v mklabel gpt", device),
	}
	if err := svc.runner.Run(ctx, commands); err != nil {
		return errors.New("erase disk failed " + device)
	}
	return nil
}

func (svc *Service) MakePart(ctx context.Context, device string) error {
	if device == "" {
		return errors.New("param missing")
	}

	command := fmt.Sprintf("parted -a opt %v mkpart primary ext4 16M 100%%", device)
	if _, err := svc.runner.Output(ctx, command); err != nil {
		return errors.New("make partition failed " + command)
	}
	return nil
}

func (svc *Service) Ext4Partition(ctx context.Context, path string) error {
	command := fmt.Sprintf("mkfs.ext4 -F %v", path)
	if _, err := svc.runner.Output(ctx, command); err != nil {
		return errors.New("格式化ext4失败" + path)
	}
	return nil
}

func (svc *Service) AddFstab(ctx context.Context, uuid string, path string, skipExisted bool, mounts []FstabMount) (string, error) {
	if len(uuid) == 0 || len(path) == 0 {
		return "", errors.New("addFstab param missing")
	}

	mount := ""
	for _, entry := range mounts {
		if entry.UUID != uuid {
			continue
		}
		mount = entry.Name
		if skipExisted && len(entry.Target) > 0 {
			return entry.Target, nil
		}
		break
	}

	commands := buildAddFstabCommands(mount, uuid, path)
	if err := svc.runner.Run(ctx, commands); err != nil {
		return "", errors.New("add fstab failed")
	}
	return path, nil
}

func (svc *Service) CommitFstab(ctx context.Context) error {
	if err := svc.runner.Run(ctx, []string{"uci commit fstab"}); err != nil {
		return errors.New("commitFstab failed")
	}
	return nil
}

func (svc *Service) CommitFstabAndBlockMount(ctx context.Context) error {
	if err := svc.runner.Run(ctx, []string{"uci commit fstab", "block mount"}); err != nil {
		return errors.New("block mount failed")
	}
	return nil
}

func buildAddFstabCommands(mount string, uuid string, path string) []string {
	if mount == "" {
		return []string{
			"uci add fstab mount",
			fmt.Sprintf("uci set fstab.@mount[-1].uuid=%v", uuid),
			fmt.Sprintf("uci set 'fstab.@mount[-1].target=%v'", path),
			"uci set fstab.@mount[-1].enabled=1",
		}
	}
	return []string{
		fmt.Sprintf("uci set fstab.%v.uuid=%v", mount, uuid),
		fmt.Sprintf("uci set 'fstab.%v.target=%v'", mount, path),
		fmt.Sprintf("uci set fstab.%v.enabled=1", mount),
	}
}

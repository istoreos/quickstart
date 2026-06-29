package service

import (
	"context"
	"errors"
	"os/exec"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	sandboxmodule "github.com/linkease/quick-start/istore-backend/modules/nas/sandbox"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type nasSandboxFacade interface {
	ListDisks(ctx context.Context) ([]*models.NasDiskInfo, error)
	Status(ctx context.Context) (string, error)
	FormatPartition(ctx context.Context, path string) error
	Commit(ctx context.Context) error
	Reset(ctx context.Context) error
	Exit(ctx context.Context) error
}

var newNasSandboxService = func() nasSandboxFacade {
	return NewDefaultNasSandboxService()
}

type nasSandboxServiceFacade struct {
	service *sandboxmodule.Service
}

func NewDefaultNasSandboxService() nasSandboxFacade {
	return &nasSandboxServiceFacade{
		service: sandboxmodule.NewService(
			defaultNasSandboxDiskReader{},
			defaultNasSandboxRuntimeStore{},
			defaultNasSandboxPartitionStore{},
		),
	}
}

func (svc *nasSandboxServiceFacade) ListDisks(ctx context.Context) ([]*models.NasDiskInfo, error) {
	return svc.service.ListDisks(ctx)
}

func (svc *nasSandboxServiceFacade) Status(ctx context.Context) (string, error) {
	status, err := svc.service.Status(ctx)
	return string(status), err
}

func (svc *nasSandboxServiceFacade) FormatPartition(ctx context.Context, path string) error {
	return svc.service.FormatPartition(ctx, path)
}

func (svc *nasSandboxServiceFacade) Commit(ctx context.Context) error {
	return svc.service.Commit(ctx)
}

func (svc *nasSandboxServiceFacade) Reset(ctx context.Context) error {
	return svc.service.Reset(ctx)
}

func (svc *nasSandboxServiceFacade) Exit(ctx context.Context) error {
	return svc.service.Exit(ctx)
}

type defaultNasSandboxDiskReader struct{}

func (reader defaultNasSandboxDiskReader) ReadAll(ctx context.Context) ([]*models.NasDiskInfo, error) {
	return getAllDisks(ctx)
}

type defaultNasSandboxRuntimeStore struct{}

func (store defaultNasSandboxRuntimeStore) HasSandboxBinary() bool {
	return canAccessPath("/usr/sbin/sandbox")
}

func (store defaultNasSandboxRuntimeStore) Status(ctx context.Context) (sandboxmodule.Status, error) {
	_, _, err := utils.BatchOutErr(ctx, []string{"/usr/sbin/sandbox status"}, 0)
	if err != nil {
		if ex, ok := err.(*exec.ExitError); ok && ex.ExitCode() == 1 {
			return sandboxmodule.StatusStopped, nil
		}
		return "", err
	}
	return sandboxmodule.StatusRunning, nil
}

func (store defaultNasSandboxRuntimeStore) RunAction(ctx context.Context, action sandboxmodule.Action) error {
	cmd := ""
	switch action {
	case sandboxmodule.ActionCommit:
		cmd = "/usr/sbin/sandbox commit && reboot"
	case sandboxmodule.ActionReset:
		cmd = "/usr/sbin/sandbox reset && reboot"
	case sandboxmodule.ActionExit:
		cmd = "/usr/sbin/sandbox exit && reboot"
	default:
		return errors.New("unsupported sandbox action")
	}
	_, _, err := utils.BatchOutErr(ctx, []string{cmd}, 0)
	return err
}

type defaultNasSandboxPartitionStore struct{}

func (store defaultNasSandboxPartitionStore) Unmount(mountPoint string) error {
	return Unmount(mountPoint)
}

func (store defaultNasSandboxPartitionStore) Ext4Partition(path string) error {
	return Ext4Partition(path)
}

func (store defaultNasSandboxPartitionStore) ClearOverlayMounts(ctx context.Context) {
	uci.LoadConfig("fstab", true)
	sections, _ := uci.GetSections("fstab", "mount")
	for _, sec := range sections {
		targetStr, result := uci.GetLast("fstab", sec, "target")
		if result && targetStr == "/overlay" {
			cmdStr := []string{
				"uci del fstab." + sec,
				"uci commit fstab",
			}
			utils.BatchRun(ctx, cmdStr, 0)
		}
	}
}

func (store defaultNasSandboxPartitionStore) AddOverlayFstab(uuid string) error {
	_, err := AddFstab(uuid, "/overlay", false)
	return err
}

func (store defaultNasSandboxPartitionStore) CommitFstab() error {
	return commitFstab()
}

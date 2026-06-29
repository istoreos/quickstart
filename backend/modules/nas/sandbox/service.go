package sandbox

import (
	"context"
	"errors"
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Status string

const (
	StatusUnsupported Status = "unsupport"
	StatusRunning     Status = "running"
	StatusStopped     Status = "stopped"
)

type Action string

const (
	ActionCommit Action = "commit"
	ActionReset  Action = "reset"
	ActionExit   Action = "exit"
)

var waitRefresh = func(d time.Duration) {
	time.Sleep(d)
}

type DiskReader interface {
	ReadAll(ctx context.Context) ([]*models.NasDiskInfo, error)
}

type RuntimeStore interface {
	HasSandboxBinary() bool
	Status(ctx context.Context) (Status, error)
	RunAction(ctx context.Context, action Action) error
}

type PartitionStore interface {
	Unmount(mountPoint string) error
	Ext4Partition(path string) error
	ClearOverlayMounts(ctx context.Context)
	AddOverlayFstab(uuid string) error
	CommitFstab() error
}

type Service struct {
	diskReader     DiskReader
	runtimeStore   RuntimeStore
	partitionStore PartitionStore
}

func NewService(diskReader DiskReader, runtimeStore RuntimeStore, partitionStore PartitionStore) *Service {
	return &Service{
		diskReader:     diskReader,
		runtimeStore:   runtimeStore,
		partitionStore: partitionStore,
	}
}

func (svc *Service) ListDisks(ctx context.Context) ([]*models.NasDiskInfo, error) {
	disks, err := svc.diskReader.ReadAll(ctx)
	if err != nil {
		return nil, err
	}

	externalDisks := make([]*models.NasDiskInfo, 0)
	for _, disk := range disks {
		if disk != nil && disk.IsExternalDisk {
			externalDisks = append(externalDisks, disk)
		}
	}
	return externalDisks, nil
}

func (svc *Service) Status(ctx context.Context) (Status, error) {
	if !svc.runtimeStore.HasSandboxBinary() {
		return StatusUnsupported, nil
	}

	status, err := svc.runtimeStore.Status(ctx)
	if err != nil {
		return StatusUnsupported, nil
	}
	if status == "" {
		return StatusUnsupported, nil
	}
	return status, nil
}

func (svc *Service) Commit(ctx context.Context) error {
	return svc.runAction(ctx, ActionCommit, "提交失败")
}

func (svc *Service) Reset(ctx context.Context) error {
	return svc.runAction(ctx, ActionReset, "重置失败")
}

func (svc *Service) Exit(ctx context.Context) error {
	return svc.runAction(ctx, ActionExit, "退出失败")
}

func (svc *Service) FormatPartition(ctx context.Context, path string) error {
	disks, err := svc.diskReader.ReadAll(ctx)
	if err != nil {
		return err
	}

	target := findPartition(disks, path)
	if target == nil {
		return errors.New("partition not found" + path)
	}

	if checkMountPoint(target.MountPoint) {
		if err := svc.partitionStore.Unmount(target.MountPoint); err != nil {
			return err
		}
	}

	if err := svc.partitionStore.Ext4Partition(target.Path); err != nil {
		return err
	}

	waitRefresh(3 * time.Second)

	disks, err = svc.diskReader.ReadAll(ctx)
	if err != nil {
		return err
	}
	for _, disk := range disks {
		if disk == nil {
			continue
		}
		for _, partition := range disk.Childrens {
			if partition == nil {
				continue
			}
			if !checkMountPoint(partition.MountPoint) && partition.Path == path {
				svc.partitionStore.ClearOverlayMounts(ctx)
				if err := svc.partitionStore.AddOverlayFstab(partition.UUID); err != nil {
					return err
				}
				if err := svc.partitionStore.CommitFstab(); err != nil {
					return err
				}
				return nil
			}
		}
	}
	return errors.New("sanbox format error ")
}

func (svc *Service) runAction(ctx context.Context, action Action, legacyPrefix string) error {
	if err := svc.runtimeStore.RunAction(ctx, action); err != nil {
		return errors.New(legacyPrefix + err.Error())
	}
	return nil
}

func findPartition(disks []*models.NasDiskInfo, path string) *models.PartitionInfo {
	for _, disk := range disks {
		if disk == nil {
			continue
		}
		for _, partition := range disk.Childrens {
			if partition != nil && partition.Path == path {
				return partition
			}
		}
	}
	return nil
}

func checkMountPoint(mountPoint string) bool {
	if mountPoint == "" || mountPoint == "-" {
		return false
	}
	return true
}

package automount

import (
	"context"
	"path/filepath"
)

type Disk struct {
	Partitions []Partition
}

type Partition struct {
	Name       string
	Path       string
	MountPoint string
	UUID       string
}

type Store interface {
	AutoMountEnabled(ctx context.Context) bool
	ListDisks(ctx context.Context) ([]Disk, error)
	HasFstabMount(uuid string) bool
	MountPointInUse(ctx context.Context, mountPoint string) bool
	GenerateMountName(name string) string
	AddFstab(uuid string, mountPoint string) error
	CommitFstab() error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Reload(ctx context.Context) error {
	if !svc.store.AutoMountEnabled(ctx) {
		return nil
	}
	disks, err := svc.store.ListDisks(ctx)
	if err != nil {
		return err
	}
	for _, disk := range disks {
		for _, partition := range disk.Partitions {
			if partition.MountPoint != "" || partition.UUID == "" || svc.store.HasFstabMount(partition.UUID) {
				continue
			}
			mountPath := "/mnt/" + svc.store.GenerateMountName(filepath.Base(partition.Path))
			if svc.store.MountPointInUse(ctx, mountPath) {
				continue
			}
			if err := svc.store.AddFstab(partition.UUID, mountPath); err != nil {
				continue
			}
			_ = svc.store.CommitFstab()
		}
	}
	return nil
}

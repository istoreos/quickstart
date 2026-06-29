package service

import (
	"context"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/modules/nas/automount"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type nasAutoMountFacade interface {
	Reload(ctx context.Context) error
}

var newNasAutoMountService = func() nasAutoMountFacade {
	return automount.NewService(defaultNasAutoMountStore{})
}

type defaultNasAutoMountStore struct{}

func (store defaultNasAutoMountStore) AutoMountEnabled(ctx context.Context) bool {
	stdout, _, _ := utils.BatchOutErr(ctx, []string{"uci get fstab.@global[0].anon_mount"}, 0)
	return stdout == "1"
}

func (store defaultNasAutoMountStore) ListDisks(ctx context.Context) ([]automount.Disk, error) {
	status, err := getDisksStatus(ctx)
	if err != nil {
		return nil, err
	}
	disks := make([]automount.Disk, 0, len(status.Disks))
	for _, disk := range status.Disks {
		mapped := automount.Disk{Partitions: make([]automount.Partition, 0, len(disk.Childrens))}
		for _, partition := range disk.Childrens {
			mapped.Partitions = append(mapped.Partitions, automount.Partition{
				Name:       partition.Name,
				Path:       partition.Path,
				MountPoint: partition.MountPoint,
				UUID:       partition.UUID,
			})
		}
		disks = append(disks, mapped)
	}
	return disks, nil
}

func (store defaultNasAutoMountStore) HasFstabMount(uuid string) bool {
	uci.LoadConfig("fstab", true)
	sections, _ := uci.GetSections("fstab", "mount")
	for _, section := range sections {
		uuidStr, _ := uci.GetLast("fstab", section, "uuid")
		if uuidStr == uuid {
			return true
		}
	}
	return false
}

func (store defaultNasAutoMountStore) MountPointInUse(ctx context.Context, mountPoint string) bool {
	stdout, _, _ := utils.BatchOutErr(ctx, []string{"mountpoint -qd " + mountPoint}, 0)
	return len(stdout) > 0
}

func (store defaultNasAutoMountStore) GenerateMountName(name string) string {
	return genMountPoint(name)
}

func (store defaultNasAutoMountStore) AddFstab(uuid string, mountPoint string) error {
	_, err := AddFstab(uuid, mountPoint, false)
	return err
}

func (store defaultNasAutoMountStore) CommitFstab() error {
	return commitFstab()
}

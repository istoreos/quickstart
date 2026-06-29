package service

import (
	"context"
	"fmt"
	"os"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/raid/inventory"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type raidInventoryFacade interface {
	List(ctx context.Context) ([]*models.NasDiskInfo, error)
	Detail(ctx context.Context, path string) (string, error)
	CreateList(ctx context.Context) ([]*models.RaidMemberInfo, error)
}

var newRaidInventoryService = func() raidInventoryFacade {
	return inventory.NewService(defaultRaidInventoryStore{})
}

type defaultRaidInventoryStore struct{}

func (store defaultRaidInventoryStore) ReadMDStat(ctx context.Context) (string, error) {
	content, err := os.ReadFile("/proc/mdstat")
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (store defaultRaidInventoryStore) ReadDisk(ctx context.Context, name string) (*models.NasDiskInfo, error) {
	return get_disk_info(name)
}

func (store defaultRaidInventoryStore) ReadMDDetail(ctx context.Context, path string) (map[string]string, error) {
	return mddetail(path), nil
}

func (store defaultRaidInventoryStore) ReadDetailText(ctx context.Context, path string) (string, error) {
	out, err := utils.BatchOutputCmd(ctx, fmt.Sprintf("mdadm -D %v", path), 0)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (store defaultRaidInventoryStore) ReadAllDisks(ctx context.Context) ([]*models.NasDiskInfo, error) {
	diskStatus, err := getDisksStatus(ctx)
	if err != nil {
		return nil, err
	}
	return diskStatus.Disks, nil
}

func (store defaultRaidInventoryStore) ReadRaidMember(path string) string {
	return is_raid_member(path)
}

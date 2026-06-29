package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/linkease/quick-start/istore-backend/modules/raid/writeflow"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type raidWriteFlowStore struct{}

func newRaidWriteFlowService() *writeflow.Service {
	return writeflow.NewService(raidWriteFlowStore{})
}

func (store raidWriteFlowStore) ReadDisk(ctx context.Context, name string) (*writeflow.Disk, error) {
	disk, err := get_disk_info(name)
	if err != nil {
		return nil, err
	}
	partitions := make([]writeflow.Partition, 0, len(disk.Childrens))
	for _, part := range disk.Childrens {
		partitions = append(partitions, writeflow.Partition{
			MountPoint: part.MountPoint,
			Path:       part.Path,
			IsRaidOn:   part.IsRaidOn,
		})
	}
	return &writeflow.Disk{
		Path:       disk.Path,
		Partitions: partitions,
	}, nil
}

func (store raidWriteFlowStore) Unmount(ctx context.Context, mountPoint string) error {
	return Unmount(mountPoint)
}

func (store raidWriteFlowStore) Erase(ctx context.Context, path string) error {
	return Erase(path)
}

func (store raidWriteFlowStore) MakeRaidPart(ctx context.Context, path string) error {
	return makeRaidPart(path)
}

func (store raidWriteFlowStore) WaitAfterPartition(ctx context.Context) {
	time.Sleep(time.Second)
}

func (store raidWriteFlowStore) WaitAfterCreate(ctx context.Context) {
	time.Sleep(time.Second)
}

func (store raidWriteFlowStore) FindFreeMd(min int) int {
	return findFreeMd(min)
}

func (store raidWriteFlowStore) RunCreate(ctx context.Context, command string) error {
	return utils.BatchRun(ctx, []string{command}, 0)
}

func (store raidWriteFlowStore) CleanupCreatedDevice(ctx context.Context, path string) {
	utils.BatchOutputCmd(ctx, fmt.Sprintf("rm -f %v", path), 0)
}

func (store raidWriteFlowStore) GenerateMdadmConfig(ctx context.Context) error {
	return gen_mdadm_config()
}

func (store raidWriteFlowStore) UnmountMountPath(ctx context.Context, mountPath string) error {
	return utils.BatchRun(ctx, []string{fmt.Sprintf("umount '%v'", mountPath)}, 0)
}

func (store raidWriteFlowStore) RunDelete(ctx context.Context, commands []string) error {
	return utils.BatchRun(ctx, commands, 0)
}

func (store raidWriteFlowStore) RemoveDeletedDevice(ctx context.Context, path string) {
	utils.BatchOutputCmd(ctx, fmt.Sprintf("rm %v", path), 0)
}

func (store raidWriteFlowStore) ActiveDeviceCount(ctx context.Context, path string) uint64 {
	detail := mddetail(path)
	count, _ := strconv.ParseUint(detail["Active Devices"], 10, 8)
	return count
}

func (store raidWriteFlowStore) RunAdd(ctx context.Context, command string) (string, error) {
	_, stderr, err := utils.BatchOutErr(ctx, []string{command}, 0)
	return stderr, err
}

func (store raidWriteFlowStore) Grow(ctx context.Context, command string) {
	utils.BatchOutputCmd(ctx, command, 0)
}

func (store raidWriteFlowStore) RunRemove(ctx context.Context, commands []string) error {
	return utils.BatchRun(ctx, commands, 10)
}

func (store raidWriteFlowStore) RunRecover(ctx context.Context, commands []string) error {
	return utils.BatchRun(ctx, commands, 0)
}

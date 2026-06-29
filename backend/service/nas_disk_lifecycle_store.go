package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/linkease/quick-start/istore-backend/models"
	disklifecycle "github.com/linkease/quick-start/istore-backend/modules/nas/disklifecycle"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type NasDiskSnapshotReader = disklifecycle.SnapshotReader
type NasDiskCommandStore = disklifecycle.CommandStore
type NasDiskMountPointGenerator = disklifecycle.MountPointGenerator

var readNasAllDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
	return getAllDisks(ctx)
}

var readNasDisk = func(name string) (*models.NasDiskInfo, error) {
	return get_disk_info(name)
}

var readNasDiskIncludeFree = func(name string) (*models.NasDiskInfo, error) {
	return get_disk_info_include_free(name)
}

type defaultNasDiskSnapshotReader struct{}

func newDefaultNasDiskSnapshotReader() NasDiskSnapshotReader {
	return &defaultNasDiskSnapshotReader{}
}

func (reader *defaultNasDiskSnapshotReader) ReadAll(ctx context.Context) ([]NasDiskLifecycleDiskSnapshot, error) {
	disks, err := readNasAllDisks(ctx)
	if err != nil {
		return nil, err
	}
	return buildNasDiskLifecycleDiskSnapshots(disks), nil
}

func (reader *defaultNasDiskSnapshotReader) ReadDisk(ctx context.Context, name string) (*NasDiskLifecycleDiskSnapshot, error) {
	disk, err := readNasDisk(name)
	if err != nil {
		return nil, err
	}
	snapshots := buildNasDiskLifecycleDiskSnapshots([]*models.NasDiskInfo{disk})
	if len(snapshots) == 0 {
		return nil, nil
	}
	return &snapshots[0], nil
}

func (reader *defaultNasDiskSnapshotReader) ReadDiskIncludeFree(ctx context.Context, name string) (*NasDiskLifecycleDiskSnapshot, error) {
	disk, err := readNasDiskIncludeFree(name)
	if err != nil {
		return nil, err
	}
	snapshots := buildNasDiskLifecycleDiskSnapshots([]*models.NasDiskInfo{disk})
	if len(snapshots) == 0 {
		return nil, nil
	}
	return &snapshots[0], nil
}

type defaultNasDiskCommandStore struct{}

func newDefaultNasDiskCommandStore() NasDiskCommandStore {
	return &defaultNasDiskCommandStore{}
}

func (store *defaultNasDiskCommandStore) Mount(devicePath string, mountPoint string) error {
	return Mount(devicePath, mountPoint)
}

func (store *defaultNasDiskCommandStore) UnMount(devicePath string) error {
	return UnMount(devicePath)
}

func (store *defaultNasDiskCommandStore) Unmount(mountPoint string) error {
	return Unmount(mountPoint)
}

func (store *defaultNasDiskCommandStore) Erase(devicePath string) error {
	return Erase(devicePath)
}

func (store *defaultNasDiskCommandStore) MakePart(devicePath string) error {
	return MakePart(devicePath)
}

func (store *defaultNasDiskCommandStore) FixGPTTable(devicePath string) error {
	cmdStr := fmt.Sprintf(`printf "ok\nfix\n" | parted ---pretend-input-tty %v print`, devicePath)
	err := utils.BatchRun(context.Background(), []string{cmdStr}, 0)
	if err != nil {
		return errors.New("fix gpt table 报错" + cmdStr)
	}
	return nil
}

func (store *defaultNasDiskCommandStore) MakePartRange(devicePath string, typeOrName string, alignedStart uint64, alignedEnd uint64) error {
	cmdStr := fmt.Sprintf("parted -a optimal %v mkpart \"%v\" ext4 %vs %vs", devicePath, typeOrName, alignedStart, alignedEnd)
	stdout, stderr, err := utils.BatchOutErr(context.Background(), []string{cmdStr}, 0)
	if err != nil {
		return errors.New(stderr + stdout + "创建分区失败 " + cmdStr)
	}
	return nil
}

func (store *defaultNasDiskCommandStore) Ext4Partition(devicePath string) error {
	return Ext4Partition(devicePath)
}

func (store *defaultNasDiskCommandStore) AddFstab(uuid string, path string, skipExisted bool) (string, error) {
	return AddFstab(uuid, path, skipExisted)
}

func (store *defaultNasDiskCommandStore) CommitFstab() error {
	return commitFstab()
}

func (store *defaultNasDiskCommandStore) CommitFstabAndBlockMount() error {
	return commitFstabAndBlockMount()
}

type defaultNasDiskMountPointGenerator struct{}

func newDefaultNasDiskMountPointGenerator() NasDiskMountPointGenerator {
	return &defaultNasDiskMountPointGenerator{}
}

func (generator *defaultNasDiskMountPointGenerator) Generate(name string) string {
	return genMountPoint(name)
}

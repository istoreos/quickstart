package disklifecycle

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeSnapshotReader struct {
	readAllResults             [][]DiskSnapshot
	readAllErr                 error
	readAllCalls               int
	readDiskResults            []*DiskSnapshot
	readDiskErr                error
	readDiskCalls              int
	readDiskIncludeFreeResults []*DiskSnapshot
	readDiskIncludeFreeErr     error
	readDiskIncludeFreeCalls   int
}

func (reader *fakeSnapshotReader) ReadAll(ctx context.Context) ([]DiskSnapshot, error) {
	if reader.readAllErr != nil {
		return nil, reader.readAllErr
	}
	idx := reader.readAllCalls
	reader.readAllCalls++
	if idx >= len(reader.readAllResults) {
		return nil, nil
	}
	return reader.readAllResults[idx], nil
}

func (reader *fakeSnapshotReader) ReadDisk(ctx context.Context, name string) (*DiskSnapshot, error) {
	if reader.readDiskErr != nil {
		return nil, reader.readDiskErr
	}
	idx := reader.readDiskCalls
	reader.readDiskCalls++
	if idx >= len(reader.readDiskResults) {
		return nil, nil
	}
	return reader.readDiskResults[idx], nil
}

func (reader *fakeSnapshotReader) ReadDiskIncludeFree(ctx context.Context, name string) (*DiskSnapshot, error) {
	if reader.readDiskIncludeFreeErr != nil {
		return nil, reader.readDiskIncludeFreeErr
	}
	idx := reader.readDiskIncludeFreeCalls
	reader.readDiskIncludeFreeCalls++
	if idx >= len(reader.readDiskIncludeFreeResults) {
		return nil, nil
	}
	return reader.readDiskIncludeFreeResults[idx], nil
}

type fakeCommandStore struct {
	mountCalls                    [][2]string
	unMountCalls                  []string
	unmountCalls                  []string
	eraseCalls                    []string
	makePartCalls                 []string
	fixGPTTableCalls              []string
	makePartRangeCalls            []string
	ext4Calls                     []string
	addFstabCalls                 [][3]string
	addFstabSkipExisted           []bool
	addFstabResults               []string
	addFstabErr                   error
	commitFstabCalls              int
	commitFstabErr                error
	commitFstabAndBlockMountCalls int
	commitFstabAndBlockMountErr   error
	mountErr                      error
	unMountErr                    error
	unmountErr                    error
	eraseErr                      error
	makePartErr                   error
	fixGPTTableErr                error
	makePartRangeErr              error
	ext4Err                       error
}

func (store *fakeCommandStore) Mount(devicePath string, mountPoint string) error {
	store.mountCalls = append(store.mountCalls, [2]string{devicePath, mountPoint})
	return store.mountErr
}

func (store *fakeCommandStore) UnMount(devicePath string) error {
	store.unMountCalls = append(store.unMountCalls, devicePath)
	return store.unMountErr
}

func (store *fakeCommandStore) Unmount(mountPoint string) error {
	store.unmountCalls = append(store.unmountCalls, mountPoint)
	return store.unmountErr
}

func (store *fakeCommandStore) Erase(devicePath string) error {
	store.eraseCalls = append(store.eraseCalls, devicePath)
	return store.eraseErr
}

func (store *fakeCommandStore) MakePart(devicePath string) error {
	store.makePartCalls = append(store.makePartCalls, devicePath)
	return store.makePartErr
}

func (store *fakeCommandStore) FixGPTTable(devicePath string) error {
	store.fixGPTTableCalls = append(store.fixGPTTableCalls, devicePath)
	return store.fixGPTTableErr
}

func (store *fakeCommandStore) MakePartRange(devicePath string, typeOrName string, alignedStart uint64, alignedEnd uint64) error {
	store.makePartRangeCalls = append(store.makePartRangeCalls, devicePath+"|"+typeOrName)
	return store.makePartRangeErr
}

func (store *fakeCommandStore) Ext4Partition(devicePath string) error {
	store.ext4Calls = append(store.ext4Calls, devicePath)
	return store.ext4Err
}

func (store *fakeCommandStore) AddFstab(uuid string, path string, skipExisted bool) (string, error) {
	store.addFstabCalls = append(store.addFstabCalls, [3]string{uuid, path, ""})
	store.addFstabSkipExisted = append(store.addFstabSkipExisted, skipExisted)
	if store.addFstabErr != nil {
		return "", store.addFstabErr
	}
	if len(store.addFstabResults) == 0 {
		return path, nil
	}
	result := store.addFstabResults[0]
	store.addFstabResults = store.addFstabResults[1:]
	return result, nil
}

func (store *fakeCommandStore) CommitFstab() error {
	store.commitFstabCalls++
	return store.commitFstabErr
}

func (store *fakeCommandStore) CommitFstabAndBlockMount() error {
	store.commitFstabAndBlockMountCalls++
	return store.commitFstabAndBlockMountErr
}

type fakeMountPointGenerator struct {
	name string
}

func (generator *fakeMountPointGenerator) Generate(name string) string {
	if generator.name != "" {
		return generator.name
	}
	return name
}

func skipLifecycleWaits(t *testing.T) {
	t.Helper()
	originalWait := waitRefresh
	waitRefresh = func(d time.Duration) {}
	t.Cleanup(func() { waitRefresh = originalWait })
}

func TestServiceMountPartitionMountsAndWritesFstab(t *testing.T) {
	t.Parallel()

	reader := &fakeSnapshotReader{
		readAllResults: [][]DiskSnapshot{
			{{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1"}}}},
			{{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1"}}}},
		},
	}
	store := &fakeCommandStore{}
	svc := NewService(reader, store, &fakeMountPointGenerator{})

	partition, err := svc.MountPartition(context.Background(), PartitionMountInput{
		UUID:       "uuid-1",
		Path:       "/dev/sda1",
		MountPoint: "/mnt/data_sda1",
	})
	if err != nil {
		t.Fatalf("unexpected mount partition error: %v", err)
	}
	if partition == nil || partition.MountPoint != "/mnt/data_sda1" {
		t.Fatalf("unexpected partition result: %#v", partition)
	}
	if len(store.mountCalls) != 1 || store.mountCalls[0] != [2]string{"/dev/sda1", "/mnt/data_sda1"} {
		t.Fatalf("unexpected mount calls: %#v", store.mountCalls)
	}
	if len(store.addFstabCalls) != 1 || store.addFstabCalls[0][0] != "uuid-1" || store.addFstabCalls[0][1] != "/mnt/data_sda1" || store.addFstabSkipExisted[0] {
		t.Fatalf("unexpected add fstab calls: %#v skip=%#v", store.addFstabCalls, store.addFstabSkipExisted)
	}
	if store.commitFstabAndBlockMountCalls != 1 {
		t.Fatalf("expected final apply, got %d", store.commitFstabAndBlockMountCalls)
	}
}

func TestServiceGenerateMountPointUsesPathBaseName(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeSnapshotReader{}, &fakeCommandStore{}, &fakeMountPointGenerator{name: "data_sda1"})

	mountPoint, err := svc.GenerateMountPoint(context.Background(), "/dev/sda1")
	if err != nil {
		t.Fatalf("GenerateMountPoint returned error: %v", err)
	}
	if mountPoint != "/mnt/data_sda1" {
		t.Fatalf("mountPoint = %q, want /mnt/data_sda1", mountPoint)
	}
}

func TestServiceGenerateMountPointReturnsErrorForEmptyGeneratedName(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeSnapshotReader{}, &fakeCommandStore{}, &fakeMountPointGenerator{name: ""})

	_, err := svc.GenerateMountPoint(context.Background(), "")
	if err == nil || err.Error() != "mountPoint生成失败" {
		t.Fatalf("GenerateMountPoint error = %v, want mountPoint生成失败", err)
	}
}

func TestServiceFormatByDevicePathUnmountsThenRemounts(t *testing.T) {
	skipLifecycleWaits(t)

	reader := &fakeSnapshotReader{
		readAllResults: [][]DiskSnapshot{
			{{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1", MountPoint: "/mnt/old"}}}},
			{{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1"}}}},
		},
	}
	store := &fakeCommandStore{}
	svc := NewService(reader, store, &fakeMountPointGenerator{name: "data_sda1"})

	partition, err := svc.FormatByDevicePath(context.Background(), FormatByDevicePathInput{DevicePath: "/dev/sda1"})
	if err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}
	if len(store.unmountCalls) != 1 || store.unmountCalls[0] != "/mnt/old" {
		t.Fatalf("expected mounted partition to be unmounted, got %#v", store.unmountCalls)
	}
	if len(store.ext4Calls) != 1 || store.ext4Calls[0] != "/dev/sda1" {
		t.Fatalf("unexpected ext4 calls: %#v", store.ext4Calls)
	}
	if partition == nil || partition.MountPoint != "/mnt/data_sda1" {
		t.Fatalf("unexpected formatted partition: %#v", partition)
	}
}

func TestServiceInitDiskHandlesMDAndRegularDisks(t *testing.T) {
	skipLifecycleWaits(t)

	mdReader := &fakeSnapshotReader{
		readDiskResults: []*DiskSnapshot{
			{Name: "md0", Path: "/dev/md0", Partitions: []PartitionSnapshot{{Name: "md0", Path: "/dev/md0", UUID: "uuid-md0"}}},
			{Name: "md0", Path: "/dev/md0", Partitions: []PartitionSnapshot{{Name: "md0", Path: "/dev/md0", UUID: "uuid-md0"}}},
		},
	}
	mdStore := &fakeCommandStore{}
	mdSvc := NewService(mdReader, mdStore, &fakeMountPointGenerator{name: "data_md0"})
	if _, err := mdSvc.InitDisk(context.Background(), InitInput{Name: "md0", Path: "/dev/md0"}); err != nil {
		t.Fatalf("unexpected md init error: %v", err)
	}
	if len(mdStore.ext4Calls) != 1 || mdStore.ext4Calls[0] != "/dev/md0" {
		t.Fatalf("expected direct ext4 on md device, got %#v", mdStore.ext4Calls)
	}
	if len(mdStore.makePartCalls) != 0 {
		t.Fatalf("did not expect MakePart for md device, got %#v", mdStore.makePartCalls)
	}

	regularReader := &fakeSnapshotReader{
		readDiskResults: []*DiskSnapshot{
			{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", MountPoint: "/mnt/old"}}},
			{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-sda1"}}},
			{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-sda1"}}},
		},
	}
	regularStore := &fakeCommandStore{}
	regularSvc := NewService(regularReader, regularStore, &fakeMountPointGenerator{name: "data_sda1"})
	disk, err := regularSvc.InitDisk(context.Background(), InitInput{Name: "sda", Path: "/dev/sda"})
	if err != nil {
		t.Fatalf("unexpected regular init error: %v", err)
	}
	if len(regularStore.unmountCalls) != 1 || regularStore.unmountCalls[0] != "/mnt/old" {
		t.Fatalf("expected mounted child to be unmounted first, got %#v", regularStore.unmountCalls)
	}
	if len(regularStore.eraseCalls) != 1 || regularStore.eraseCalls[0] != "/dev/sda" {
		t.Fatalf("unexpected erase calls: %#v", regularStore.eraseCalls)
	}
	if len(regularStore.makePartCalls) != 1 || regularStore.makePartCalls[0] != "/dev/sda" {
		t.Fatalf("unexpected makepart calls: %#v", regularStore.makePartCalls)
	}
	if len(regularStore.unMountCalls) != 1 || regularStore.unMountCalls[0] != "/dev/sda1" {
		t.Fatalf("expected device unmount before child format, got %#v", regularStore.unMountCalls)
	}
	if disk == nil || len(disk.Childrens) != 1 || disk.Childrens[0].MountPoint != "/mnt/data_sda1" {
		t.Fatalf("unexpected initialized disk result: %#v", disk)
	}
}

func TestServiceInitDiskRestHandlesFreeSpaceAndGPT(t *testing.T) {
	skipLifecycleWaits(t)

	reader := &fakeSnapshotReader{
		readDiskIncludeFreeResults: []*DiskSnapshot{
			{
				Name:          "sda",
				Path:          "/dev/sda",
				PartLabelType: "GPT",
				Partitions: []PartitionSnapshot{
					{Name: "sda1", Path: "/dev/sda1", Filesystem: "ext4"},
					{Name: "free", Path: "", Filesystem: "Free Space", SecStart: 4097, SecEnd: 8192},
				},
			},
		},
		readDiskResults: []*DiskSnapshot{
			{
				Name: "sda",
				Path: "/dev/sda",
				Partitions: []PartitionSnapshot{
					{Name: "sda1", Path: "/dev/sda1", Filesystem: "ext4"},
					{Name: "sda2", Path: "/dev/sda2", Filesystem: "No FileSystem"},
				},
			},
			{
				Name: "sda",
				Path: "/dev/sda",
				Partitions: []PartitionSnapshot{
					{Name: "sda1", Path: "/dev/sda1", Filesystem: "ext4"},
					{Name: "sda2", Path: "/dev/sda2", Filesystem: "ext4", UUID: "uuid-sda2"},
				},
			},
		},
	}
	store := &fakeCommandStore{addFstabResults: []string{"/mnt/existing"}}
	svc := NewService(reader, store, &fakeMountPointGenerator{name: "data_sda2"})

	disk, err := svc.InitDiskRest(context.Background(), InitRestInput{Name: "sda", Path: "/dev/sda"})
	if err != nil {
		t.Fatalf("unexpected init-rest error: %v", err)
	}
	if len(store.fixGPTTableCalls) != 1 || store.fixGPTTableCalls[0] != "/dev/sda" {
		t.Fatalf("expected GPT fix call, got %#v", store.fixGPTTableCalls)
	}
	if len(store.makePartRangeCalls) != 1 || store.makePartRangeCalls[0] != "/dev/sda|UserData" {
		t.Fatalf("unexpected makepart-range calls: %#v", store.makePartRangeCalls)
	}
	if len(store.ext4Calls) != 1 || store.ext4Calls[0] != "/dev/sda2" {
		t.Fatalf("expected ext4 on new child partition, got %#v", store.ext4Calls)
	}
	if len(store.unMountCalls) != 1 || store.unMountCalls[0] != "/dev/sda2" {
		t.Fatalf("expected direct unmount before remounting child, got %#v", store.unMountCalls)
	}
	if disk == nil || len(disk.Childrens) != 2 || disk.Childrens[1].MountPoint != "/mnt/existing" {
		t.Fatalf("unexpected init-rest result: %#v", disk)
	}
}

func TestServicePropagatesReaderStoreAndApplyErrors(t *testing.T) {
	t.Parallel()

	readerErr := errors.New("snapshot failed")
	svc := NewService(
		&fakeSnapshotReader{readAllErr: readerErr},
		&fakeCommandStore{},
		&fakeMountPointGenerator{},
	)
	if _, err := svc.MountPartition(context.Background(), PartitionMountInput{UUID: "u", Path: "/dev/sda1", MountPoint: "/mnt/data"}); !errors.Is(err, readerErr) {
		t.Fatalf("expected snapshot error, got %v", err)
	}

	storeErr := errors.New("erase failed")
	svc = NewService(
		&fakeSnapshotReader{readDiskResults: []*DiskSnapshot{{Name: "sda", Path: "/dev/sda"}}},
		&fakeCommandStore{eraseErr: storeErr},
		&fakeMountPointGenerator{},
	)
	if _, err := svc.InitDisk(context.Background(), InitInput{Name: "sda", Path: "/dev/sda"}); !errors.Is(err, storeErr) {
		t.Fatalf("expected store error, got %v", err)
	}

	svc = NewService(
		&fakeSnapshotReader{
			readAllResults: [][]DiskSnapshot{
				{{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1"}}}},
				{{Name: "sda", Path: "/dev/sda", Partitions: []PartitionSnapshot{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1"}}}},
			},
		},
		&fakeCommandStore{commitFstabAndBlockMountErr: errors.New("apply failed")},
		&fakeMountPointGenerator{},
	)
	if _, err := svc.MountPartition(context.Background(), PartitionMountInput{UUID: "uuid-1", Path: "/dev/sda1", MountPoint: "/mnt/data"}); err == nil || err.Error() != "apply failed" {
		t.Fatalf("expected final apply error, got %v", err)
	}
}

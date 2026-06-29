package writeflow

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type fakeStore struct {
	disks map[string]*Disk

	freeMd int

	readErr       error
	unmountErr    error
	eraseErr      error
	makePartErr   error
	runCreateErr  error
	runDeleteErr  error
	runAddErr     error
	runRemoveErr  error
	runRecoverErr error
	genConfigErr  error
	rawUnmountErr error
	addStderr     string
	activeDevices uint64

	events []string
}

func (store *fakeStore) ReadDisk(ctx context.Context, name string) (*Disk, error) {
	store.events = append(store.events, "read:"+name)
	if store.readErr != nil {
		return nil, store.readErr
	}
	return store.disks[name], nil
}

func (store *fakeStore) Unmount(ctx context.Context, mountPoint string) error {
	store.events = append(store.events, "unmount:"+mountPoint)
	return store.unmountErr
}

func (store *fakeStore) Erase(ctx context.Context, path string) error {
	store.events = append(store.events, "erase:"+path)
	return store.eraseErr
}

func (store *fakeStore) MakeRaidPart(ctx context.Context, path string) error {
	store.events = append(store.events, "make-part:"+path)
	return store.makePartErr
}

func (store *fakeStore) WaitAfterPartition(ctx context.Context) {
	store.events = append(store.events, "wait-partition")
}

func (store *fakeStore) WaitAfterCreate(ctx context.Context) {
	store.events = append(store.events, "wait-create")
}

func (store *fakeStore) FindFreeMd(min int) int {
	store.events = append(store.events, "find-md")
	return store.freeMd
}

func (store *fakeStore) RunCreate(ctx context.Context, command string) error {
	store.events = append(store.events, "create:"+command)
	return store.runCreateErr
}

func (store *fakeStore) CleanupCreatedDevice(ctx context.Context, path string) {
	store.events = append(store.events, "cleanup:"+path)
}

func (store *fakeStore) GenerateMdadmConfig(ctx context.Context) error {
	store.events = append(store.events, "gen-config")
	return store.genConfigErr
}

func (store *fakeStore) UnmountMountPath(ctx context.Context, mountPath string) error {
	store.events = append(store.events, "raw-unmount:"+mountPath)
	return store.rawUnmountErr
}

func (store *fakeStore) RunDelete(ctx context.Context, commands []string) error {
	store.events = append(store.events, "delete:"+fmt.Sprint(commands))
	return store.runDeleteErr
}

func (store *fakeStore) RemoveDeletedDevice(ctx context.Context, path string) {
	store.events = append(store.events, "remove:"+path)
}

func (store *fakeStore) ActiveDeviceCount(ctx context.Context, path string) uint64 {
	store.events = append(store.events, "active-count:"+path)
	return store.activeDevices
}

func (store *fakeStore) RunAdd(ctx context.Context, command string) (string, error) {
	store.events = append(store.events, "add:"+command)
	return store.addStderr, store.runAddErr
}

func (store *fakeStore) Grow(ctx context.Context, command string) {
	store.events = append(store.events, "grow:"+command)
}

func (store *fakeStore) RunRemove(ctx context.Context, commands []string) error {
	store.events = append(store.events, "remove-member:"+fmt.Sprint(commands))
	return store.runRemoveErr
}

func (store *fakeStore) RunRecover(ctx context.Context, commands []string) error {
	store.events = append(store.events, "recover:"+fmt.Sprint(commands))
	return store.runRecoverErr
}

func TestCreatePreparesDisksCreatesArrayAndGeneratesConfig(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		freeMd: 1,
		disks: map[string]*Disk{
			"sda": {Path: "/dev/sda", Partitions: []Partition{{MountPoint: "/mnt/a"}}},
			"sdb": {Path: "/dev/sdb"},
		},
	}

	got, err := NewService(store).Create(context.Background(), CreateInput{
		Level:       "raid1",
		DevicePaths: []string{"/dev/sda", "/dev/sdb"},
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if got != "/dev/md1" {
		t.Fatalf("unexpected created device: %q", got)
	}

	want := []string{
		"read:sda",
		"unmount:/mnt/a",
		"erase:/dev/sda",
		"make-part:/dev/sda",
		"read:sdb",
		"erase:/dev/sdb",
		"make-part:/dev/sdb",
		"wait-partition",
		"find-md",
		"create:mdadm -C /dev/md1 --run --quiet --assume-clean --homehost=any -n 2 -l 1 /dev/sda1 /dev/sdb1",
		"gen-config",
		"wait-create",
	}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestCreateRejectsInsufficientMembers(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	_, err := NewService(store).Create(context.Background(), CreateInput{
		Level:       "raid5",
		DevicePaths: []string{"/dev/sda", "/dev/sdb"},
	})
	if err == nil || err.Error() != "没有足够的成员设备" {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.events) != 0 {
		t.Fatalf("did not expect store calls, got %#v", store.events)
	}
}

func TestCreateRejectsExistingRaidMember(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		disks: map[string]*Disk{
			"sda": {Path: "/dev/sda", Partitions: []Partition{{MountPoint: "Raid Member: md1"}}},
			"sdb": {Path: "/dev/sdb"},
		},
	}

	_, err := NewService(store).Create(context.Background(), CreateInput{
		Level:       "raid1",
		DevicePaths: []string{"/dev/sda", "/dev/sdb"},
	})
	if err == nil || err.Error() != "Raid Member: md1 already found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCleansUpDeviceWhenMdadmCreateFails(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		freeMd:       2,
		runCreateErr: errors.New("failed"),
		disks: map[string]*Disk{
			"sda": {Path: "/dev/sda"},
			"sdb": {Path: "/dev/sdb"},
		},
	}

	_, err := NewService(store).Create(context.Background(), CreateInput{
		Level:       "raid1",
		DevicePaths: []string{"/dev/sda", "/dev/sdb"},
	})
	wantErr := "raid创建失败,请重试 mdadm -C /dev/md2 --run --quiet --assume-clean --homehost=any -n 2 -l 1 /dev/sda1 /dev/sdb1"
	if err == nil || err.Error() != wantErr {
		t.Fatalf("unexpected error:\nwant=%q\ngot=%v", wantErr, err)
	}
	if last := store.events[len(store.events)-1]; last != "cleanup:/dev/md2" {
		t.Fatalf("expected cleanup as final event, got %#v", store.events)
	}
}

func TestDeleteUnmountsStopsRemovesAndRegeneratesConfig(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}

	if err := NewService(store).Delete(context.Background(), DeleteInput{
		Path:      "/dev/md1",
		MountPath: "/mnt/raid",
		Members:   []string{"/dev/sda1", "/dev/sdb1"},
	}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	want := []string{
		"raw-unmount:/mnt/raid",
		"delete:[mdadm --stop /dev/md1 mdadm --remove /dev/md1 mdadm --zero-superblock /dev/sda1 /dev/sdb1]",
		"remove:/dev/md1",
		"gen-config",
	}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected delete events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestDeleteWrapsUnmountAndDeleteErrors(t *testing.T) {
	t.Parallel()

	unmountStore := &fakeStore{rawUnmountErr: errors.New("busy")}
	err := NewService(unmountStore).Delete(context.Background(), DeleteInput{MountPath: "/mnt/raid"})
	if err == nil || err.Error() != "卸载磁盘失败busy" {
		t.Fatalf("unexpected unmount error: %v", err)
	}

	deleteStore := &fakeStore{runDeleteErr: errors.New("failed")}
	err = NewService(deleteStore).Delete(context.Background(), DeleteInput{Path: "/dev/md1"})
	if err == nil || err.Error() != "raid删除失败failed" {
		t.Fatalf("unexpected delete error: %v", err)
	}
}

func TestAddPreparesMemberAddsAndGrowsArray(t *testing.T) {
	t.Parallel()

	store := &fakeStore{activeDevices: 2}

	if err := NewService(store).Add(context.Background(), MemberInput{
		Path:       "/dev/md1",
		MemberPath: "/dev/sdc",
	}); err != nil {
		t.Fatalf("Add returned error: %v", err)
	}

	want := []string{
		"erase:/dev/sdc",
		"make-part:/dev/sdc",
		"wait-partition",
		"active-count:/dev/md1",
		"add:mdadm -a /dev/md1 /dev/sdc1",
		"grow:mdadm -G /dev/md1 -n 3",
	}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected add events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestAddWrapsAddErrorWithStderr(t *testing.T) {
	t.Parallel()

	store := &fakeStore{runAddErr: errors.New("failed"), addStderr: "stderr"}
	err := NewService(store).Add(context.Background(), MemberInput{Path: "/dev/md1", MemberPath: "/dev/sdc"})
	if err == nil || err.Error() != "扩充成员失败 stderr" {
		t.Fatalf("unexpected add error: %v", err)
	}
}

func TestRemoveRunsFailAndRemoveCommands(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}

	if err := NewService(store).Remove(context.Background(), MemberInput{
		Path:       "/dev/md1",
		MemberPath: "/dev/sdc1",
	}); err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}

	want := []string{"remove-member:[mdadm --manage /dev/md1 --fail /dev/sdc1 mdadm --manage /dev/md1 --remove /dev/sdc1]"}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected remove events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestRemoveWrapsError(t *testing.T) {
	t.Parallel()

	store := &fakeStore{runRemoveErr: errors.New("failed")}
	err := NewService(store).Remove(context.Background(), MemberInput{Path: "/dev/md1", MemberPath: "/dev/sdc1"})
	if err == nil || err.Error() != "删除成员失败" {
		t.Fatalf("unexpected remove error: %v", err)
	}
}

func TestRecoverUsesExistingRaidPartitionWhenRequested(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		disks: map[string]*Disk{
			"sdc": {Path: "/dev/sdc", Partitions: []Partition{{Path: "/dev/sdc1", IsRaidOn: true}}},
		},
	}

	if err := NewService(store).Recover(context.Background(), RecoverInput{
		Path:               "/dev/md1",
		MemberPath:         "sdc",
		CheckRaidPartition: true,
	}); err != nil {
		t.Fatalf("Recover returned error: %v", err)
	}

	want := []string{
		"read:sdc",
		"wait-partition",
		"recover:[mdadm -a /dev/md1 /dev/sdc1]",
	}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected recover events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestRecoverPreparesMemberWhenExistingRaidPartitionIsNotUsed(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}

	if err := NewService(store).Recover(context.Background(), RecoverInput{
		Path:       "/dev/md1",
		MemberPath: "/dev/sdc",
	}); err != nil {
		t.Fatalf("Recover returned error: %v", err)
	}

	want := []string{
		"erase:/dev/sdc",
		"make-part:/dev/sdc",
		"wait-partition",
		"recover:[mdadm -a /dev/md1 /dev/sdc1]",
	}
	if !reflect.DeepEqual(store.events, want) {
		t.Fatalf("unexpected recover events:\nwant=%#v\ngot=%#v", want, store.events)
	}
}

func TestRecoverWrapsErrorWithCommand(t *testing.T) {
	t.Parallel()

	store := &fakeStore{runRecoverErr: errors.New("failed")}
	err := NewService(store).Recover(context.Background(), RecoverInput{Path: "/dev/md1", MemberPath: "/dev/sdc"})
	want := "恢复失败 mdadm -a /dev/md1 /dev/sdc1"
	if err == nil || err.Error() != want {
		t.Fatalf("unexpected recover error:\nwant=%q\ngot=%v", want, err)
	}
}

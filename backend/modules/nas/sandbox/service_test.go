package sandbox

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeDiskReader struct {
	disksResults [][]*models.NasDiskInfo
	err          error
	calls        int
}

func (reader *fakeDiskReader) ReadAll(ctx context.Context) ([]*models.NasDiskInfo, error) {
	if reader.err != nil {
		return nil, reader.err
	}
	idx := reader.calls
	reader.calls++
	if idx >= len(reader.disksResults) {
		return nil, nil
	}
	return reader.disksResults[idx], nil
}

type fakeRuntimeStore struct {
	available bool
	status    Status
	err       error
	actions   []Action
	actionErr error
}

func (store *fakeRuntimeStore) HasSandboxBinary() bool {
	return store.available
}

func (store *fakeRuntimeStore) Status(ctx context.Context) (Status, error) {
	return store.status, store.err
}

func (store *fakeRuntimeStore) RunAction(ctx context.Context, action Action) error {
	store.actions = append(store.actions, action)
	return store.actionErr
}

type fakePartitionStore struct {
	unmountCalls      []string
	formatCalls       []string
	clearOverlayCalls int
	addOverlayCalls   []string
	commitCalls       int
	unmountErr        error
	formatErr         error
	addOverlayErr     error
	commitErr         error
}

func (store *fakePartitionStore) Unmount(mountPoint string) error {
	store.unmountCalls = append(store.unmountCalls, mountPoint)
	return store.unmountErr
}

func (store *fakePartitionStore) Ext4Partition(path string) error {
	store.formatCalls = append(store.formatCalls, path)
	return store.formatErr
}

func (store *fakePartitionStore) ClearOverlayMounts(ctx context.Context) {
	store.clearOverlayCalls++
}

func (store *fakePartitionStore) AddOverlayFstab(uuid string) error {
	store.addOverlayCalls = append(store.addOverlayCalls, uuid)
	return store.addOverlayErr
}

func (store *fakePartitionStore) CommitFstab() error {
	store.commitCalls++
	return store.commitErr
}

func skipRefresh(t *testing.T) {
	t.Helper()
	original := waitRefresh
	waitRefresh = func(d time.Duration) {}
	t.Cleanup(func() { waitRefresh = original })
}

func TestServiceListDisksReturnsOnlyExternalDisks(t *testing.T) {
	t.Parallel()

	reader := &fakeDiskReader{disksResults: [][]*models.NasDiskInfo{{
		{Name: "sda", IsExternalDisk: false},
		{Name: "usb0", IsExternalDisk: true},
		{Name: "usb1", IsExternalDisk: true},
	}}}
	svc := NewService(reader, &fakeRuntimeStore{}, &fakePartitionStore{})

	disks, err := svc.ListDisks(context.Background())
	if err != nil {
		t.Fatalf("unexpected list disks error: %v", err)
	}
	if len(disks) != 2 || disks[0].Name != "usb0" || disks[1].Name != "usb1" {
		t.Fatalf("expected external disks only, got %#v", disks)
	}
}

func TestServiceStatusMapsSupportAndRuntimeState(t *testing.T) {
	t.Parallel()

	unsupported := NewService(&fakeDiskReader{}, &fakeRuntimeStore{available: false}, &fakePartitionStore{})
	if status, err := unsupported.Status(context.Background()); err != nil || status != StatusUnsupported {
		t.Fatalf("expected unsupported status, got status=%q err=%v", status, err)
	}

	running := NewService(&fakeDiskReader{}, &fakeRuntimeStore{available: true, status: StatusRunning}, &fakePartitionStore{})
	if status, err := running.Status(context.Background()); err != nil || status != StatusRunning {
		t.Fatalf("expected running status, got status=%q err=%v", status, err)
	}

	failed := NewService(&fakeDiskReader{}, &fakeRuntimeStore{available: true, err: errors.New("status failed")}, &fakePartitionStore{})
	if status, err := failed.Status(context.Background()); err != nil || status != StatusUnsupported {
		t.Fatalf("expected unsupported status on unknown runtime error, got status=%q err=%v", status, err)
	}
}

func TestServiceActionsRunExpectedSandboxCommand(t *testing.T) {
	t.Parallel()

	runtime := &fakeRuntimeStore{}
	svc := NewService(&fakeDiskReader{}, runtime, &fakePartitionStore{})

	if err := svc.Commit(context.Background()); err != nil {
		t.Fatalf("unexpected commit error: %v", err)
	}
	if err := svc.Reset(context.Background()); err != nil {
		t.Fatalf("unexpected reset error: %v", err)
	}
	if err := svc.Exit(context.Background()); err != nil {
		t.Fatalf("unexpected exit error: %v", err)
	}
	expected := []Action{ActionCommit, ActionReset, ActionExit}
	if len(runtime.actions) != len(expected) {
		t.Fatalf("unexpected action calls: %#v", runtime.actions)
	}
	for i := range expected {
		if runtime.actions[i] != expected[i] {
			t.Fatalf("unexpected action calls: %#v", runtime.actions)
		}
	}
}

func TestServiceActionsPreserveLegacyErrorPrefixes(t *testing.T) {
	t.Parallel()

	actionErr := errors.New("runtime failed")
	svc := NewService(&fakeDiskReader{}, &fakeRuntimeStore{actionErr: actionErr}, &fakePartitionStore{})

	if err := svc.Commit(context.Background()); err == nil || err.Error() != "提交失败runtime failed" {
		t.Fatalf("expected commit prefix, got %v", err)
	}
	if err := svc.Reset(context.Background()); err == nil || err.Error() != "重置失败runtime failed" {
		t.Fatalf("expected reset prefix, got %v", err)
	}
	if err := svc.Exit(context.Background()); err == nil || err.Error() != "退出失败runtime failed" {
		t.Fatalf("expected exit prefix, got %v", err)
	}
}

func TestServiceFormatPartitionFormatsUnmountedPartitionAndWritesOverlayFstab(t *testing.T) {
	skipRefresh(t)

	reader := &fakeDiskReader{disksResults: [][]*models.NasDiskInfo{
		{
			{Name: "usb", Childrens: []*models.PartitionInfo{
				{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1", MountPoint: "/mnt/old"},
			}},
		},
		{
			{Name: "usb", Childrens: []*models.PartitionInfo{
				{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1"},
			}},
		},
	}}
	partitionStore := &fakePartitionStore{}
	svc := NewService(reader, &fakeRuntimeStore{}, partitionStore)

	if err := svc.FormatPartition(context.Background(), "/dev/sda1"); err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}
	if len(partitionStore.unmountCalls) != 1 || partitionStore.unmountCalls[0] != "/mnt/old" {
		t.Fatalf("expected old mount to be unmounted, got %#v", partitionStore.unmountCalls)
	}
	if len(partitionStore.formatCalls) != 1 || partitionStore.formatCalls[0] != "/dev/sda1" {
		t.Fatalf("expected target partition to be formatted, got %#v", partitionStore.formatCalls)
	}
	if partitionStore.clearOverlayCalls != 1 {
		t.Fatalf("expected overlay fstab cleanup, got %d", partitionStore.clearOverlayCalls)
	}
	if len(partitionStore.addOverlayCalls) != 1 || partitionStore.addOverlayCalls[0] != "uuid-1" {
		t.Fatalf("expected overlay fstab for partition UUID, got %#v", partitionStore.addOverlayCalls)
	}
	if partitionStore.commitCalls != 1 {
		t.Fatalf("expected fstab commit, got %d", partitionStore.commitCalls)
	}
}

func TestServiceFormatPartitionPropagatesLookupAndStoreErrors(t *testing.T) {
	t.Parallel()

	readerErr := errors.New("read failed")
	svc := NewService(&fakeDiskReader{err: readerErr}, &fakeRuntimeStore{}, &fakePartitionStore{})
	if err := svc.FormatPartition(context.Background(), "/dev/sda1"); !errors.Is(err, readerErr) {
		t.Fatalf("expected reader error, got %v", err)
	}

	svc = NewService(&fakeDiskReader{disksResults: [][]*models.NasDiskInfo{{}}}, &fakeRuntimeStore{}, &fakePartitionStore{})
	if err := svc.FormatPartition(context.Background(), "/dev/missing"); err == nil || err.Error() != "partition not found/dev/missing" {
		t.Fatalf("expected partition-not-found error, got %v", err)
	}

	formatErr := errors.New("format failed")
	svc = NewService(
		&fakeDiskReader{disksResults: [][]*models.NasDiskInfo{{{Childrens: []*models.PartitionInfo{{Path: "/dev/sda1"}}}}}},
		&fakeRuntimeStore{},
		&fakePartitionStore{formatErr: formatErr},
	)
	if err := svc.FormatPartition(context.Background(), "/dev/sda1"); !errors.Is(err, formatErr) {
		t.Fatalf("expected format error, got %v", err)
	}
}

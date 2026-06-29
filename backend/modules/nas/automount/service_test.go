package automount

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	enabled        bool
	listCalled     bool
	disks          []Disk
	listErr        error
	existingUUIDs  map[string]bool
	usedMounts     map[string]bool
	generatedNames map[string]string
	addCalls       [][2]string
	addErr         error
	commitCalls    int
	commitErr      error
}

func (store *fakeStore) AutoMountEnabled(ctx context.Context) bool {
	return store.enabled
}

func (store *fakeStore) ListDisks(ctx context.Context) ([]Disk, error) {
	store.listCalled = true
	return store.disks, store.listErr
}

func (store *fakeStore) HasFstabMount(uuid string) bool {
	return store.existingUUIDs[uuid]
}

func (store *fakeStore) MountPointInUse(ctx context.Context, mountPoint string) bool {
	return store.usedMounts[mountPoint]
}

func (store *fakeStore) GenerateMountName(name string) string {
	if generated := store.generatedNames[name]; generated != "" {
		return generated
	}
	return "data_" + name
}

func (store *fakeStore) AddFstab(uuid string, mountPoint string) error {
	store.addCalls = append(store.addCalls, [2]string{uuid, mountPoint})
	return store.addErr
}

func (store *fakeStore) CommitFstab() error {
	store.commitCalls++
	return store.commitErr
}

func TestReloadSkipsWhenAutoMountDisabled(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}
	if store.listCalled {
		t.Fatal("ListDisks was called while disabled")
	}
}

func TestReloadAddsFstabForEligiblePartitions(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		enabled: true,
		disks: []Disk{{
			Partitions: []Partition{{
				Name: "sda1",
				Path: "/dev/sda1",
				UUID: "uuid-1",
			}},
		}},
		generatedNames: map[string]string{"sda1": "data_sda1"},
	}
	svc := NewService(store)

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}
	if len(store.addCalls) != 1 || store.addCalls[0] != [2]string{"uuid-1", "/mnt/data_sda1"} {
		t.Fatalf("addCalls = %#v", store.addCalls)
	}
	if store.commitCalls != 1 {
		t.Fatalf("commitCalls = %d, want 1", store.commitCalls)
	}
}

func TestReloadSkipsIneligiblePartitions(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		enabled:       true,
		existingUUIDs: map[string]bool{"uuid-existing": true},
		usedMounts:    map[string]bool{"/mnt/data_sdd1": true},
		disks: []Disk{{
			Partitions: []Partition{
				{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-mounted", MountPoint: "/mnt/old"},
				{Name: "sdb1", Path: "/dev/sdb1"},
				{Name: "sdc1", Path: "/dev/sdc1", UUID: "uuid-existing"},
				{Name: "sdd1", Path: "/dev/sdd1", UUID: "uuid-used"},
			},
		}},
	}
	svc := NewService(store)

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}
	if len(store.addCalls) != 0 {
		t.Fatalf("addCalls = %#v, want none", store.addCalls)
	}
	if store.commitCalls != 0 {
		t.Fatalf("commitCalls = %d, want 0", store.commitCalls)
	}
}

func TestReloadContinuesWhenAddOrCommitFails(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		enabled: true,
		disks: []Disk{{
			Partitions: []Partition{
				{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-1"},
				{Name: "sdb1", Path: "/dev/sdb1", UUID: "uuid-2"},
			},
		}},
		addErr:    errors.New("add failed"),
		commitErr: errors.New("commit failed"),
	}
	svc := NewService(store)

	if err := svc.Reload(context.Background()); err != nil {
		t.Fatalf("Reload returned error: %v", err)
	}
	if len(store.addCalls) != 2 {
		t.Fatalf("addCalls = %#v, want two attempts", store.addCalls)
	}
	if store.commitCalls != 0 {
		t.Fatalf("commitCalls = %d, want 0 when add fails", store.commitCalls)
	}
}

func TestReloadPropagatesDiskListError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("list failed")
	svc := NewService(&fakeStore{enabled: true, listErr: expectedErr})

	if err := svc.Reload(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("Reload error = %v, want expectedErr", err)
	}
}

package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	mdstat        string
	mdstatErr     error
	diskByName    map[string]*models.NasDiskInfo
	diskErr       error
	detailByPath  map[string]map[string]string
	detailText    string
	detailTextErr error
	allDisks      []*models.NasDiskInfo
	allDisksErr   error
	raidMembers   map[string]string
}

func (store *fakeStore) ReadMDStat(ctx context.Context) (string, error) {
	if store.mdstatErr != nil {
		return "", store.mdstatErr
	}
	return store.mdstat, nil
}

func (store *fakeStore) ReadDisk(ctx context.Context, name string) (*models.NasDiskInfo, error) {
	if store.diskErr != nil {
		return nil, store.diskErr
	}
	return store.diskByName[name], nil
}

func (store *fakeStore) ReadMDDetail(ctx context.Context, path string) (map[string]string, error) {
	if store.detailByPath == nil {
		return nil, nil
	}
	return store.detailByPath[path], nil
}

func (store *fakeStore) ReadDetailText(ctx context.Context, path string) (string, error) {
	if store.detailTextErr != nil {
		return "", store.detailTextErr
	}
	return store.detailText, nil
}

func (store *fakeStore) ReadAllDisks(ctx context.Context) ([]*models.NasDiskInfo, error) {
	if store.allDisksErr != nil {
		return nil, store.allDisksErr
	}
	return store.allDisks, nil
}

func (store *fakeStore) ReadRaidMember(path string) string {
	return store.raidMembers[path]
}

func TestServiceListBuildsRaidDevicesFromMDStat(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		mdstat: "Personalities : [raid1]\n" +
			"md0 : active raid1 sda1[0] sdb1[1]\n" +
			"      1046528 blocks super 1.2 [2/2] [UU]\n" +
			"      [====>................]  recovery = 23.4%\n",
		diskByName: map[string]*models.NasDiskInfo{
			"md0": {
				Childrens: []*models.PartitionInfo{
					{Name: "md0", Path: "/dev/md0", SecStart: 2048, SecEnd: 4096},
				},
			},
		},
		detailByPath: map[string]map[string]string{
			"/dev/md0": {"State": "clean"},
		},
	}
	svc := NewService(store)

	disks, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(disks) != 1 {
		t.Fatalf("expected one raid disk, got %#v", disks)
	}
	disk := disks[0]
	if disk.Name != "md0" || disk.Path != "/dev/md0" || disk.Active != "active" || disk.Level != "raid1" {
		t.Fatalf("unexpected raid identity: %#v", disk)
	}
	if len(disk.Members) != 2 || disk.Members[0] != "/dev/sda1" || disk.Members[1] != "/dev/sdb1" {
		t.Fatalf("unexpected raid members: %#v", disk.Members)
	}
	if disk.Status != "clean" {
		t.Fatalf("expected mdadm state to populate status, got %q", disk.Status)
	}
	if disk.RebuildStatus != "recovery = 23.4%" {
		t.Fatalf("unexpected rebuild status: %q", disk.RebuildStatus)
	}
	if len(disk.Childrens) != 1 || disk.Childrens[0].SecStart != 0 || disk.Childrens[0].SecEnd != 0 {
		t.Fatalf("expected child sector range to be hidden, got %#v", disk.Childrens)
	}
}

func TestServiceListDoesNotMutateStoreDiskSnapshots(t *testing.T) {
	t.Parallel()

	storeDisk := &models.NasDiskInfo{
		Name: "original",
		Path: "/dev/original",
		Childrens: []*models.PartitionInfo{
			nil,
			{Name: "md0", Path: "/dev/md0", SecStart: 2048, SecEnd: 4096},
		},
	}
	store := &fakeStore{
		mdstat: "Personalities : [raid1]\n" +
			"md0 : active raid1 sda1[0] sdb1[1]\n",
		diskByName: map[string]*models.NasDiskInfo{"md0": storeDisk},
	}
	svc := NewService(store)

	disks, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(disks) != 1 {
		t.Fatalf("expected one raid disk, got %#v", disks)
	}
	if disks[0] == storeDisk {
		t.Fatalf("expected returned disk to be cloned, got same pointer")
	}
	if storeDisk.Name != "original" || storeDisk.Path != "/dev/original" {
		t.Fatalf("expected store disk identity to remain unchanged, got %#v", storeDisk)
	}
	if storeDisk.Childrens[1].SecStart != 2048 || storeDisk.Childrens[1].SecEnd != 4096 {
		t.Fatalf("expected store child sectors to remain unchanged, got %#v", storeDisk.Childrens[1])
	}
	if len(disks[0].Childrens) != 2 || disks[0].Childrens[0] != nil || disks[0].Childrens[1].SecStart != 0 || disks[0].Childrens[1].SecEnd != 0 {
		t.Fatalf("expected returned child sectors to be hidden without nil panic, got %#v", disks[0].Childrens)
	}
}

func TestServiceListPreservesLegacyMDStatReadError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{mdstatErr: errors.New("missing mdstat")})
	if _, err := svc.List(context.Background()); err == nil || err.Error() != "读取raid配置文件失败" {
		t.Fatalf("expected legacy mdstat error, got %v", err)
	}
}

func TestServiceCreateListFiltersEligibleDisks(t *testing.T) {
	t.Parallel()

	store := &fakeStore{
		allDisks: []*models.NasDiskInfo{
			{Name: "root", Path: "/dev/root", IsSystemRoot: true},
			{Name: "empty", Path: "/dev/sda", Size: "1 TiB", VenderModel: "empty-model"},
			{Name: "mounted", Path: "/dev/sdb", Childrens: []*models.PartitionInfo{{Path: "/dev/sdb1", MountPoint: "/mnt/data"}}},
			{Name: "raid", Path: "/dev/sdc", Childrens: []*models.PartitionInfo{{Path: "/dev/sdc1"}}},
			{Name: "candidate", Path: "/dev/sdd", Size: "4 TiB", VenderModel: "candidate-model", Childrens: []*models.PartitionInfo{{Path: "/dev/sdd1"}}},
		},
		raidMembers: map[string]string{
			"/dev/sdc1": "Raid Member: md0",
		},
	}
	svc := NewService(store)

	members, err := svc.CreateList(context.Background())
	if err != nil {
		t.Fatalf("unexpected create-list error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected empty and candidate disks, got %#v", members)
	}
	if members[0].Name != "empty" || members[0].Path != "/dev/sda" || members[0].Model != "empty-model" || members[0].SizeStr != "1 TiB" {
		t.Fatalf("unexpected empty disk member: %#v", members[0])
	}
	if members[1].Name != "candidate" || members[1].Path != "/dev/sdd" || members[1].Model != "candidate-model" || members[1].SizeStr != "4 TiB" {
		t.Fatalf("unexpected candidate disk member: %#v", members[1])
	}
}

func TestServiceCreateListPropagatesDiskReadError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("disk status failed")
	svc := NewService(&fakeStore{allDisksErr: expectedErr})
	if _, err := svc.CreateList(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected disk status error, got %v", err)
	}
}

func TestServiceDetailReadsRawMDADMDetail(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{detailText: "mdadm detail"})
	detail, err := svc.Detail(context.Background(), "/dev/md0")
	if err != nil {
		t.Fatalf("unexpected detail error: %v", err)
	}
	if detail != "mdadm detail" {
		t.Fatalf("unexpected detail text: %q", detail)
	}
}

func TestServiceDetailPreservesLegacyError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{detailTextErr: errors.New("mdadm failed")})
	if _, err := svc.Detail(context.Background(), "/dev/md0"); err == nil || err.Error() != "获取raid详情失败" {
		t.Fatalf("expected legacy detail error, got %v", err)
	}
}

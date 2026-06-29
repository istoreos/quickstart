package inventory

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	deviceNames []string
	scanOutput  string
	scanErr     error
	infoByName  map[string]*models.SmartInfo
	infoErr     error

	infoCalls []string
}

func (store *fakeStore) DeviceNames(ctx context.Context) []string {
	return append([]string(nil), store.deviceNames...)
}

func (store *fakeStore) Scan(ctx context.Context) (string, error) {
	return store.scanOutput, store.scanErr
}

func (store *fakeStore) Info(ctx context.Context, device string) (*models.SmartInfo, error) {
	store.infoCalls = append(store.infoCalls, device)
	if store.infoErr != nil {
		return nil, store.infoErr
	}
	return store.infoByName[device], nil
}

func TestFilterCandidateDeviceNamesPreservesLegacyPatterns(t *testing.T) {
	got := FilterCandidateDeviceNames([]string{
		"sda", "sdb1", "mmcblk0", "mmcblk0p1", "sataa", "nvme0n1", "nvme0", "loop0",
	})
	want := []string{"sda", "mmcblk0", "sataa", "nvme0n1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected candidates\nwant: %#v\n got: %#v", want, got)
	}
}

func TestServiceListsScannedSmartDevicesAndAlwaysIncludesNVMeNames(t *testing.T) {
	store := &fakeStore{
		deviceNames: []string{"sda", "sdb", "nvme0n1", "loop0"},
		scanOutput:  "/dev/sda -d sat # /dev/sda\n",
		infoByName: map[string]*models.SmartInfo{
			"sda":     {Name: "sda", Path: "/dev/sda"},
			"nvme0n1": {Name: "nvme0n1", Path: "/dev/nvme0n1"},
		},
	}
	service := NewService(store)

	resp, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list smart devices: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected result")
	}
	wantDisks := []*models.SmartInfo{
		{Name: "sda", Path: "/dev/sda"},
		{Name: "nvme0n1", Path: "/dev/nvme0n1"},
	}
	if !reflect.DeepEqual(resp.Result.Disks, wantDisks) {
		t.Fatalf("unexpected disks\nwant: %#v\n got: %#v", wantDisks, resp.Result.Disks)
	}
	if !reflect.DeepEqual(store.infoCalls, []string{"sda", "nvme0n1"}) {
		t.Fatalf("unexpected info calls: %#v", store.infoCalls)
	}
}

func TestServicePreservesLegacyErrors(t *testing.T) {
	service := NewService(&fakeStore{scanErr: errors.New("scan failed")})
	if _, err := service.List(context.Background()); err == nil || err.Error() != "获取smart设备列表失败" {
		t.Fatalf("expected scan error, got %v", err)
	}

	infoErr := errors.New("info failed")
	service = NewService(&fakeStore{
		deviceNames: []string{"sda"},
		scanOutput:  "sda",
		infoErr:     infoErr,
	})
	if _, err := service.List(context.Background()); !errors.Is(err, infoErr) {
		t.Fatalf("expected info error, got %v", err)
	}
}

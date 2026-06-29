package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeLanSpeedLimitedDeviceRuleStore struct {
	readRuleListsFn func(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error)
}

func (store *fakeLanSpeedLimitedDeviceRuleStore) ReadRuleLists(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error) {
	return store.readRuleListsFn(ctx)
}

type fakeLanSpeedLimitedDeviceInventoryReader struct {
	readInventoryFn func(ctx context.Context) (LanSpeedLimitedDeviceInventorySnapshot, error)
}

func (reader *fakeLanSpeedLimitedDeviceInventoryReader) ReadInventory(ctx context.Context) (LanSpeedLimitedDeviceInventorySnapshot, error) {
	return reader.readInventoryFn(ctx)
}

type fakeLanSpeedLimitedDeviceHostnameReader struct {
	readHostnamesFn func(ctx context.Context) (map[string]string, error)
}

func (reader *fakeLanSpeedLimitedDeviceHostnameReader) ReadHostnames(ctx context.Context) (map[string]string, error) {
	return reader.readHostnamesFn(ctx)
}

type fakeLanSpeedLimitedDeviceListGetter struct {
	getListSpeedLimitedDevicesFn func(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error)
}

func (getter *fakeLanSpeedLimitedDeviceListGetter) GetListSpeedLimitedDevices(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error) {
	return getter.getListSpeedLimitedDevicesFn(ctx)
}

func TestServiceBackendGetLanListSpeedLimitedDevicesDelegatesToLanSpeedLimitedDeviceListService(t *testing.T) {
	t.Parallel()

	original := newLanSpeedLimitedDeviceListService
	t.Cleanup(func() {
		newLanSpeedLimitedDeviceListService = original
	})

	called := false
	expected := &models.LANCtrlSpeedLimitResponse{
		Result: []*models.LANCtrlSpeedLimitItem{
			{
				IP:            "192.168.100.8",
				Mac:           "AA:BB:CC:DD:EE:FF",
				Hostname:      "tablet",
				UploadSpeed:   1024,
				DownloadSpeed: 2048,
				NetworkAccess: false,
				Enabled:       true,
			},
		},
	}

	newLanSpeedLimitedDeviceListService = func() lanSpeedLimitedDeviceListFacade {
		return &fakeLanSpeedLimitedDeviceListGetter{
			getListSpeedLimitedDevicesFn: func(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error) {
				called = true
				_ = ctx
				return expected, nil
			},
		}
	}

	resp, err := (&ServiceBackend{}).GetLanListSpeedLimitedDevices(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected GetLanListSpeedLimitedDevices to delegate to LanSpeedLimitedDeviceListService")
	}
	if resp != expected {
		t.Fatalf("resp = %#v, want %#v", resp, expected)
	}
}

func TestLanSpeedLimitedDeviceListServiceReturnsErrorWhenRuleStoreReadFails(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("eqos unavailable")
	svc := &LanSpeedLimitedDeviceListService{
		RuleStore: &fakeLanSpeedLimitedDeviceRuleStore{
			readRuleListsFn: func(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error) {
				_ = ctx
				return nil, nil, expectedErr
			},
		},
		InventoryReader: &fakeLanSpeedLimitedDeviceInventoryReader{
			readInventoryFn: func(ctx context.Context) (LanSpeedLimitedDeviceInventorySnapshot, error) {
				t.Fatal("inventory reader should not be called after hard failure")
				return LanSpeedLimitedDeviceInventorySnapshot{}, nil
			},
		},
		HostnameReader: &fakeLanSpeedLimitedDeviceHostnameReader{
			readHostnamesFn: func(ctx context.Context) (map[string]string, error) {
				t.Fatal("hostname reader should not be called after hard failure")
				return nil, nil
			},
		},
	}

	resp, err := svc.GetListSpeedLimitedDevices(context.Background())
	if !errors.Is(err, expectedErr) {
		t.Fatalf("err = %v, want %v", err, expectedErr)
	}
	if resp != nil {
		t.Fatalf("resp = %#v, want nil", resp)
	}
}

func TestLanSpeedLimitedDeviceListServiceKeepsMergeOrderingAndVisibility(t *testing.T) {
	t.Parallel()

	svc := &LanSpeedLimitedDeviceListService{
		RuleStore: &fakeLanSpeedLimitedDeviceRuleStore{
			readRuleListsFn: func(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error) {
				_ = ctx
				return []*models.LANCtrlSpeedLimitItem{
						{Mac: "AA:BB:CC:DD:EE:FF", Enabled: true, NetworkAccess: false},
						{Mac: "11:22:33:44:55:66", Enabled: true, NetworkAccess: false},
						{Mac: "22:33:44:55:66:77", Enabled: true, NetworkAccess: false},
					}, []*models.LANCtrlSpeedLimitItem{
						{IP: "192.168.100.2", UploadSpeed: 2048, DownloadSpeed: 4096, Comment: "kid tablet", NetworkAccess: true},
					}, nil
			},
		},
		InventoryReader: &fakeLanSpeedLimitedDeviceInventoryReader{
			readInventoryFn: func(ctx context.Context) (LanSpeedLimitedDeviceInventorySnapshot, error) {
				_ = ctx
				return LanSpeedLimitedDeviceInventorySnapshot{
					DeviceByMAC: map[string]*models.DeviceInfo{
						"AA:BB:CC:DD:EE:FF": {Mac: "AA:BB:CC:DD:EE:FF", Ipv4addr: "192.168.100.2"},
						"11:22:33:44:55:66": {Mac: "11:22:33:44:55:66", Ipv4addr: "192.168.100.3"},
					},
					DeviceByIP: map[string]*models.DeviceInfo{
						"192.168.100.2": {Mac: "AA:BB:CC:DD:EE:FF", Ipv4addr: "192.168.100.2"},
					},
				}, nil
			},
		},
		HostnameReader: &fakeLanSpeedLimitedDeviceHostnameReader{
			readHostnamesFn: func(ctx context.Context) (map[string]string, error) {
				_ = ctx
				return map[string]string{
					"AA:BB:CC:DD:EE:FF": "tablet",
					"11:22:33:44:55:66": "console",
					"22:33:44:55:66:77": "offline",
				}, nil
			},
		},
	}

	resp, err := svc.GetListSpeedLimitedDevices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Result); got != 2 {
		t.Fatalf("len(resp.Result) = %d, want 2", got)
	}
	if got := resp.Result[0].IP; got != "192.168.100.2" {
		t.Fatalf("first item IP = %q, want %q", got, "192.168.100.2")
	}
	if got := resp.Result[0].Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("first item Mac = %q, want %q", got, "AA:BB:CC:DD:EE:FF")
	}
	if resp.Result[0].NetworkAccess {
		t.Fatal("expected combined speedlimit+block item to disable network access")
	}
	if got := resp.Result[0].Hostname; got != "tablet" {
		t.Fatalf("first item Hostname = %q, want %q", got, "tablet")
	}
	if got := resp.Result[1].IP; got != "192.168.100.3" {
		t.Fatalf("second item IP = %q, want %q", got, "192.168.100.3")
	}
	if got := resp.Result[1].Mac; got != "11:22:33:44:55:66" {
		t.Fatalf("second item Mac = %q, want %q", got, "11:22:33:44:55:66")
	}
	if resp.Result[1].NetworkAccess {
		t.Fatal("expected block-only item to disable network access")
	}
	if got := resp.Result[1].Hostname; got != "console" {
		t.Fatalf("second item Hostname = %q, want %q", got, "console")
	}
}

func TestLanSpeedLimitedDeviceListServiceTreatsInventoryAndHostnameAsBestEffort(t *testing.T) {
	t.Parallel()

	svc := &LanSpeedLimitedDeviceListService{
		RuleStore: &fakeLanSpeedLimitedDeviceRuleStore{
			readRuleListsFn: func(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error) {
				_ = ctx
				return []*models.LANCtrlSpeedLimitItem{
						{Mac: "AA:BB:CC:DD:EE:FF", Enabled: true, NetworkAccess: false},
					}, []*models.LANCtrlSpeedLimitItem{
						{IP: "192.168.100.2", UploadSpeed: 1024, DownloadSpeed: 2048, Comment: "desk", NetworkAccess: true},
					}, nil
			},
		},
		InventoryReader: &fakeLanSpeedLimitedDeviceInventoryReader{
			readInventoryFn: func(ctx context.Context) (LanSpeedLimitedDeviceInventorySnapshot, error) {
				_ = ctx
				return LanSpeedLimitedDeviceInventorySnapshot{}, errors.New("device list unavailable")
			},
		},
		HostnameReader: &fakeLanSpeedLimitedDeviceHostnameReader{
			readHostnamesFn: func(ctx context.Context) (map[string]string, error) {
				_ = ctx
				return nil, errors.New("dhcp unavailable")
			},
		},
	}

	resp, err := svc.GetListSpeedLimitedDevices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Result); got != 1 {
		t.Fatalf("len(resp.Result) = %d, want 1", got)
	}
	if got := resp.Result[0].IP; got != "192.168.100.2" {
		t.Fatalf("IP = %q, want %q", got, "192.168.100.2")
	}
	if got := resp.Result[0].Mac; got != "" {
		t.Fatalf("Mac = %q, want empty string when inventory is unavailable", got)
	}
	if got := resp.Result[0].Hostname; got != "" {
		t.Fatalf("Hostname = %q, want empty string when hostname enrich fails", got)
	}
}

func TestMergeLanSpeedLimitedDeviceListsPreservesSpeedLimitFieldsWhenBlocked(t *testing.T) {
	t.Parallel()

	result := mergeLanSpeedLimitedDeviceLists(
		[]*models.LANCtrlSpeedLimitItem{
			{Mac: " aa:bb:cc:dd:ee:ff ", Enabled: true, NetworkAccess: false},
		},
		[]*models.LANCtrlSpeedLimitItem{
			{
				IP:            "192.168.100.2",
				UploadSpeed:   2048,
				DownloadSpeed: 4096,
				Comment:       "kid tablet",
				NetworkAccess: true,
			},
		},
		LanSpeedLimitedDeviceInventorySnapshot{
			DeviceByMAC: map[string]*models.DeviceInfo{
				"AA:BB:CC:DD:EE:FF": {Mac: "AA:BB:CC:DD:EE:FF", Ipv4addr: "192.168.100.2"},
			},
			DeviceByIP: map[string]*models.DeviceInfo{
				"192.168.100.2": {Mac: "AA:BB:CC:DD:EE:FF", Ipv4addr: "192.168.100.2"},
			},
		},
	)

	if got := len(result); got != 1 {
		t.Fatalf("len(result) = %d, want 1", got)
	}
	if got := result[0].IP; got != "192.168.100.2" {
		t.Fatalf("IP = %q, want %q", got, "192.168.100.2")
	}
	if got := result[0].Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("Mac = %q, want %q", got, "AA:BB:CC:DD:EE:FF")
	}
	if got := result[0].UploadSpeed; got != 2048 {
		t.Fatalf("UploadSpeed = %d, want %d", got, 2048)
	}
	if got := result[0].DownloadSpeed; got != 4096 {
		t.Fatalf("DownloadSpeed = %d, want %d", got, 4096)
	}
	if got := result[0].Comment; got != "kid tablet" {
		t.Fatalf("Comment = %q, want %q", got, "kid tablet")
	}
	if result[0].NetworkAccess {
		t.Fatal("expected merged speedlimit+block item to disable network access")
	}
}

func TestMergeLanSpeedLimitedDeviceListsSkipsBlockOnlyWithoutInventoryHit(t *testing.T) {
	t.Parallel()

	result := mergeLanSpeedLimitedDeviceLists(
		[]*models.LANCtrlSpeedLimitItem{
			{Mac: "AA:BB:CC:DD:EE:FF", Enabled: true, NetworkAccess: false},
			{Mac: "11:22:33:44:55:66", Enabled: true, NetworkAccess: false},
		},
		[]*models.LANCtrlSpeedLimitItem{
			{IP: "192.168.100.8", UploadSpeed: 1024, DownloadSpeed: 2048, NetworkAccess: true},
		},
		LanSpeedLimitedDeviceInventorySnapshot{
			DeviceByMAC: map[string]*models.DeviceInfo{
				"AA:BB:CC:DD:EE:FF": {Mac: "AA:BB:CC:DD:EE:FF", Ipv4addr: "192.168.100.2"},
			},
			DeviceByIP: map[string]*models.DeviceInfo{
				"192.168.100.8": {Mac: "77:88:99:AA:BB:CC", Ipv4addr: "192.168.100.8"},
			},
		},
	)

	if got := len(result); got != 2 {
		t.Fatalf("len(result) = %d, want 2", got)
	}
	if got := result[0].IP; got != "192.168.100.8" {
		t.Fatalf("first item IP = %q, want %q", got, "192.168.100.8")
	}
	if got := result[0].Mac; got != "77:88:99:AA:BB:CC" {
		t.Fatalf("first item Mac = %q, want %q", got, "77:88:99:AA:BB:CC")
	}
	if got := result[1].Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("second item Mac = %q, want %q", got, "AA:BB:CC:DD:EE:FF")
	}
	if got := result[1].IP; got != "192.168.100.2" {
		t.Fatalf("second item IP = %q, want %q", got, "192.168.100.2")
	}
}

func TestAttachLanSpeedLimitedHostnamesUsesNormalizedMAC(t *testing.T) {
	t.Parallel()

	items := []*models.LANCtrlSpeedLimitItem{
		{Mac: " aa:bb:cc:dd:ee:ff "},
		{Mac: "11:22:33:44:55:66"},
		{Mac: ""},
		nil,
	}

	attachLanSpeedLimitedHostnames(items, map[string]string{
		"AA:BB:CC:DD:EE:FF": "tablet",
		"11:22:33:44:55:66": "",
	})

	if got := items[0].Hostname; got != "tablet" {
		t.Fatalf("first Hostname = %q, want %q", got, "tablet")
	}
	if got := items[1].Hostname; got != "" {
		t.Fatalf("second Hostname = %q, want empty string", got)
	}
	if got := items[2].Hostname; got != "" {
		t.Fatalf("third Hostname = %q, want empty string", got)
	}
}

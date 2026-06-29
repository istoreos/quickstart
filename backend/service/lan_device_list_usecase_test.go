package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeLanDeviceListLanStatusReader struct {
	readLanStatusFn func(ctx context.Context) (LanStatusSnapshot, error)
}

func (reader *fakeLanDeviceListLanStatusReader) ReadLanStatus(ctx context.Context) (LanStatusSnapshot, error) {
	if reader.readLanStatusFn != nil {
		return reader.readLanStatusFn(ctx)
	}
	return LanStatusSnapshot{}, nil
}

type fakeLanDeviceListInventoryReader struct {
	readInventoryFn func(ctx context.Context) (models.LANDevices, error)
}

func (reader *fakeLanDeviceListInventoryReader) ReadInventory(ctx context.Context) (models.LANDevices, error) {
	if reader.readInventoryFn != nil {
		return reader.readInventoryFn(ctx)
	}
	return models.LANDevices{}, nil
}

type fakeLanDeviceListDhcpTagReader struct {
	readDhcpTagsFn func(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error)
}

func (reader *fakeLanDeviceListDhcpTagReader) ReadDhcpTags(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
	if reader.readDhcpTagsFn != nil {
		return reader.readDhcpTagsFn(ctx, lanStatus)
	}
	return []*models.LANCtrlDhcpTagInfo{}, nil
}

type fakeLanDeviceListHostHintReader struct {
	readHostHintsFn func(ctx context.Context) (map[string]HostHintSnapshot, error)
}

func (reader *fakeLanDeviceListHostHintReader) ReadHostHints(ctx context.Context) (map[string]HostHintSnapshot, error) {
	if reader.readHostHintsFn != nil {
		return reader.readHostHintsFn(ctx)
	}
	return map[string]HostHintSnapshot{}, nil
}

type fakeLanDeviceListWifiAssocReader struct {
	readWifiAssocFn func(ctx context.Context) (map[string]struct{}, error)
}

func (reader *fakeLanDeviceListWifiAssocReader) ReadWifiAssoc(ctx context.Context) (map[string]struct{}, error) {
	if reader.readWifiAssocFn != nil {
		return reader.readWifiAssocFn(ctx)
	}
	return map[string]struct{}{}, nil
}

type fakeLanDeviceListTrafficStatReader struct {
	readTrafficStatsFn func(ctx context.Context, lstats *LanStats, devices models.LANDevices) (map[string]TrafficStatSnapshot, error)
}

func (reader *fakeLanDeviceListTrafficStatReader) ReadTrafficStats(ctx context.Context, lstats *LanStats, devices models.LANDevices) (map[string]TrafficStatSnapshot, error) {
	if reader.readTrafficStatsFn != nil {
		return reader.readTrafficStatsFn(ctx, lstats, devices)
	}
	return map[string]TrafficStatSnapshot{}, nil
}

type fakeLanDeviceListStaticAssignmentReader struct {
	readStaticAssignmentsFn func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) (map[string]*models.LANStaticAssigned, error)
}

func (reader *fakeLanDeviceListStaticAssignmentReader) ReadStaticAssignments(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) (map[string]*models.LANStaticAssigned, error) {
	if reader.readStaticAssignmentsFn != nil {
		return reader.readStaticAssignmentsFn(ctx, tagList)
	}
	return map[string]*models.LANStaticAssigned{}, nil
}

type fakeLanDeviceListSpeedLimitReader struct {
	readSpeedLimitsFn func(ctx context.Context) (map[string]*models.LANCtrlSpeedLimitItem, map[string]*models.LANCtrlSpeedLimitItem, error)
}

func (reader *fakeLanDeviceListSpeedLimitReader) ReadSpeedLimits(ctx context.Context) (map[string]*models.LANCtrlSpeedLimitItem, map[string]*models.LANCtrlSpeedLimitItem, error) {
	if reader.readSpeedLimitsFn != nil {
		return reader.readSpeedLimitsFn(ctx)
	}
	return map[string]*models.LANCtrlSpeedLimitItem{}, map[string]*models.LANCtrlSpeedLimitItem{}, nil
}

type fakeLanDeviceListFacade struct {
	resp          *models.LANDeviceResponse
	err           error
	seenBackend   *ServiceBackend
	seenCtxCalled bool
}

func (svc *fakeLanDeviceListFacade) GetListDevices(ctx context.Context, backend *ServiceBackend) (*models.LANDeviceResponse, error) {
	_ = ctx
	svc.seenCtxCalled = true
	svc.seenBackend = backend
	return svc.resp, svc.err
}

func TestServiceBackendGetLanListDevicesDelegatesToLanDeviceListService(t *testing.T) {
	original := newLanDeviceListService
	defer func() {
		newLanDeviceListService = original
	}()

	expected := &models.LANDeviceResponse{
		Result: &models.LANDeviceResponseResult{
			Devices: models.LANDevices{{IP: "192.168.100.20", Mac: "AA:BB:CC:DD:EE:20"}},
		},
	}
	fake := &fakeLanDeviceListFacade{resp: expected}
	newLanDeviceListService = func() lanDeviceListFacade {
		return fake
	}

	backend := &ServiceBackend{}
	resp, err := backend.GetLanListDevices(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp != expected {
		t.Fatalf("resp = %#v, want %#v", resp, expected)
	}
	if !fake.seenCtxCalled || fake.seenBackend != backend {
		t.Fatalf("delegate saw called=%v backend=%p, want backend=%p", fake.seenCtxCalled, fake.seenBackend, backend)
	}
}

func TestServiceBackendGetLanListDevicesPropagatesError(t *testing.T) {
	original := newLanDeviceListService
	defer func() {
		newLanDeviceListService = original
	}()

	wantErr := errors.New("device list failed")
	newLanDeviceListService = func() lanDeviceListFacade {
		return &fakeLanDeviceListFacade{err: wantErr}
	}

	resp, err := (&ServiceBackend{}).GetLanListDevices(context.Background())
	if !errors.Is(err, wantErr) || resp != nil {
		t.Fatalf("resp=%#v err=%v, want %v", resp, err, wantErr)
	}
}

func TestLanDeviceListServiceReturnsBaseDevicesWhenSideReadersFail(t *testing.T) {
	t.Parallel()

	svc := &LanDeviceListService{
		LanStatusReader: &fakeLanDeviceListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr:          "192.168.100.1",
					Nexthop:          "192.168.100.254",
					IsDefaultGateway: true,
				}, nil
			},
		},
		InventoryReader: &fakeLanDeviceListInventoryReader{
			readInventoryFn: func(ctx context.Context) (models.LANDevices, error) {
				_ = ctx
				return models.LANDevices{
					{
						IP:             "192.168.100.2",
						Mac:            "AA:BB:CC:DD:EE:FF",
						Intr:           "lan",
						StaticAssigned: &models.LANStaticAssigned{AssignedIP: "192.168.100.2", AssignedMac: "AA:BB:CC:DD:EE:FF"},
						SpeedLimit:     &models.LANCtrlSpeedLimitItem{IP: "192.168.100.2", Mac: "AA:BB:CC:DD:EE:FF", NetworkAccess: true},
					},
				}, nil
			},
		},
		DhcpTagReader: &fakeLanDeviceListDhcpTagReader{
			readDhcpTagsFn: func(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
				_ = ctx
				_ = lanStatus
				return nil, errors.New("boom")
			},
		},
		HostHintReader: &fakeLanDeviceListHostHintReader{
			readHostHintsFn: func(ctx context.Context) (map[string]HostHintSnapshot, error) {
				_ = ctx
				return nil, errors.New("boom")
			},
		},
		WifiAssocReader: &fakeLanDeviceListWifiAssocReader{
			readWifiAssocFn: func(ctx context.Context) (map[string]struct{}, error) {
				_ = ctx
				return nil, errors.New("boom")
			},
		},
		TrafficStatReader: &fakeLanDeviceListTrafficStatReader{
			readTrafficStatsFn: func(ctx context.Context, lstats *LanStats, devices models.LANDevices) (map[string]TrafficStatSnapshot, error) {
				_ = ctx
				_ = lstats
				_ = devices
				return nil, errors.New("boom")
			},
		},
		StaticAssignmentReader: &fakeLanDeviceListStaticAssignmentReader{
			readStaticAssignmentsFn: func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) (map[string]*models.LANStaticAssigned, error) {
				_ = ctx
				_ = tagList
				return nil, errors.New("boom")
			},
		},
		SpeedLimitReader: &fakeLanDeviceListSpeedLimitReader{
			readSpeedLimitsFn: func(ctx context.Context) (map[string]*models.LANCtrlSpeedLimitItem, map[string]*models.LANCtrlSpeedLimitItem, error) {
				_ = ctx
				return nil, nil, errors.New("boom")
			},
		},
	}

	resp, err := svc.GetListDevices(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil {
		t.Fatal("expected response result")
	}
	if got := len(resp.Result.Devices); got != 1 {
		t.Fatalf("expected 1 device, got %d", got)
	}
	device := resp.Result.Devices[0]
	if device == nil {
		t.Fatal("expected base device")
	}
	if got := device.Intr; got != "lan" {
		t.Fatalf("expected intr %q, got %q", "lan", got)
	}
	if device.StaticAssigned == nil {
		t.Fatal("expected StaticAssigned to remain initialized")
	}
	if got := device.StaticAssigned.AssignedIP; got != "192.168.100.2" {
		t.Fatalf("expected assigned ip %q, got %q", "192.168.100.2", got)
	}
	if got := device.StaticAssigned.AssignedMac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected assigned mac %q, got %q", "AA:BB:CC:DD:EE:FF", got)
	}
	if device.SpeedLimit == nil {
		t.Fatal("expected SpeedLimit to remain initialized")
	}
	if got := device.SpeedLimit.IP; got != "192.168.100.2" {
		t.Fatalf("expected speed limit ip %q to remain unchanged, got %q", "192.168.100.2", got)
	}
	if got := device.SpeedLimit.Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected speed limit mac %q to remain unchanged, got %q", "AA:BB:CC:DD:EE:FF", got)
	}
	if got := device.SpeedLimit.NetworkAccess; !got {
		t.Fatal("expected speed limit network access to remain enabled")
	}
	if got := len(resp.Result.DhcpTags); got != 0 {
		t.Fatalf("expected no dhcp tags, got %d", got)
	}
}

func TestLanDeviceListServiceReturnsErrorWhenLanStatusReaderFails(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("lan status unavailable")
	svc := &LanDeviceListService{
		LanStatusReader: &fakeLanDeviceListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{}, expectedErr
			},
		},
		InventoryReader: &fakeLanDeviceListInventoryReader{
			readInventoryFn: func(ctx context.Context) (models.LANDevices, error) {
				_ = ctx
				t.Fatal("inventory reader should not be called when lan status reader fails")
				return nil, nil
			},
		},
	}

	resp, err := svc.GetListDevices(context.Background(), nil)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected lan status error %v, got %v", expectedErr, err)
	}
	if resp != nil {
		t.Fatalf("expected nil response when lan status reader fails, got %#v", resp)
	}
}

func TestLanDeviceListServiceReturnsErrorWhenInventoryReaderFails(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("inventory unavailable")
	svc := &LanDeviceListService{
		LanStatusReader: &fakeLanDeviceListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr:          "192.168.100.1",
					Nexthop:          "192.168.100.254",
					IsDefaultGateway: true,
				}, nil
			},
		},
		InventoryReader: &fakeLanDeviceListInventoryReader{
			readInventoryFn: func(ctx context.Context) (models.LANDevices, error) {
				_ = ctx
				return nil, expectedErr
			},
		},
	}

	resp, err := svc.GetListDevices(context.Background(), nil)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected inventory error %v, got %v", expectedErr, err)
	}
	if resp != nil {
		t.Fatalf("expected nil response when inventory reader fails, got %#v", resp)
	}
}

func TestLanDeviceListServiceMergesHostnameWifiAndSpeedLimit(t *testing.T) {
	t.Parallel()

	svc := &LanDeviceListService{
		LanStatusReader: &fakeLanDeviceListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr:          "192.168.100.1",
					Nexthop:          "192.168.100.254",
					IsDefaultGateway: true,
				}, nil
			},
		},
		InventoryReader: &fakeLanDeviceListInventoryReader{
			readInventoryFn: func(ctx context.Context) (models.LANDevices, error) {
				_ = ctx
				return models.LANDevices{
					{
						IP:             "192.168.100.2",
						Mac:            "AA:BB:CC:DD:EE:FF",
						Intr:           "lan",
						StaticAssigned: &models.LANStaticAssigned{AssignedIP: "192.168.100.2", AssignedMac: "AA:BB:CC:DD:EE:FF"},
						SpeedLimit:     &models.LANCtrlSpeedLimitItem{IP: "192.168.100.2", Mac: "AA:BB:CC:DD:EE:FF", NetworkAccess: true},
					},
				}, nil
			},
		},
		DhcpTagReader: &fakeLanDeviceListDhcpTagReader{
			readDhcpTagsFn: func(ctx context.Context, lanStatus LanStatusSnapshot) ([]*models.LANCtrlDhcpTagInfo, error) {
				_ = ctx
				_ = lanStatus
				return []*models.LANCtrlDhcpTagInfo{
					{TagName: "office", TagTitle: "Office"},
				}, nil
			},
		},
		HostHintReader: &fakeLanDeviceListHostHintReader{
			readHostHintsFn: func(ctx context.Context) (map[string]HostHintSnapshot, error) {
				_ = ctx
				return map[string]HostHintSnapshot{
					"AA:BB:CC:DD:EE:FF": {Hostname: "office-pc"},
				}, nil
			},
		},
		WifiAssocReader: &fakeLanDeviceListWifiAssocReader{
			readWifiAssocFn: func(ctx context.Context) (map[string]struct{}, error) {
				_ = ctx
				return map[string]struct{}{
					"AA:BB:CC:DD:EE:FF": {},
				}, nil
			},
		},
		TrafficStatReader: &fakeLanDeviceListTrafficStatReader{
			readTrafficStatsFn: func(ctx context.Context, lstats *LanStats, devices models.LANDevices) (map[string]TrafficStatSnapshot, error) {
				_ = ctx
				_ = lstats
				_ = devices
				return map[string]TrafficStatSnapshot{
					"192.168.100.2": {
						UploadSpeed:   1536,
						DownloadSpeed: 3072,
					},
				}, nil
			},
		},
		StaticAssignmentReader: &fakeLanDeviceListStaticAssignmentReader{
			readStaticAssignmentsFn: func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) (map[string]*models.LANStaticAssigned, error) {
				_ = ctx
				if got := len(tagList); got != 1 {
					t.Fatalf("expected 1 dhcp tag, got %d", got)
				}
				return map[string]*models.LANStaticAssigned{
					"AA:BB:CC:DD:EE:FF": {
						AssignedMac: "AA:BB:CC:DD:EE:FF",
						AssignedIP:  "10.0.0.10",
						Hostname:    "office-override",
						TagName:     "office",
						TagTitle:    "Office",
					},
				}, nil
			},
		},
		SpeedLimitReader: &fakeLanDeviceListSpeedLimitReader{
			readSpeedLimitsFn: func(ctx context.Context) (map[string]*models.LANCtrlSpeedLimitItem, map[string]*models.LANCtrlSpeedLimitItem, error) {
				_ = ctx
				return map[string]*models.LANCtrlSpeedLimitItem{
						"AA:BB:CC:DD:EE:FF": {Mac: "AA:BB:CC:DD:EE:FF"},
					}, map[string]*models.LANCtrlSpeedLimitItem{
						"192.168.100.2": {
							IP:            "192.168.100.88",
							UploadSpeed:   2048,
							DownloadSpeed: 4096,
							Comment:       "limited",
						},
					}, nil
			},
		},
	}

	resp, err := svc.GetListDevices(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil {
		t.Fatal("expected response result")
	}
	if got := len(resp.Result.Devices); got != 1 {
		t.Fatalf("expected 1 device, got %d", got)
	}
	if got := len(resp.Result.DhcpTags); got != 1 {
		t.Fatalf("expected 1 dhcp tag, got %d", got)
	}
	device := resp.Result.Devices[0]
	if device == nil {
		t.Fatal("expected device")
	}
	if got := device.Hostname; got != "office-override" {
		t.Fatalf("expected hostname %q, got %q", "office-override", got)
	}
	if got := device.Intr; got != "wifi" {
		t.Fatalf("expected intr %q, got %q", "wifi", got)
	}
	if device.StaticAssigned == nil {
		t.Fatal("expected static assignment")
	}
	if got := device.StaticAssigned.AssignedIP; got != "192.168.100.2" {
		t.Fatalf("expected assigned ip to preserve default %q, got %q", "192.168.100.2", got)
	}
	if got := device.StaticAssigned.TagName; got != "office" {
		t.Fatalf("expected tag name %q, got %q", "office", got)
	}
	if got := device.UploadSpeed; got != 1536 {
		t.Fatalf("expected upload speed %d, got %d", 1536, got)
	}
	if got := device.DownloadSpeed; got != 3072 {
		t.Fatalf("expected download speed %d, got %d", 3072, got)
	}
	if got := device.UploadSpeedStr; got == "" {
		t.Fatal("expected upload speed string")
	}
	if got := device.DownloadSpeedStr; got == "" {
		t.Fatal("expected download speed string")
	}
	if device.SpeedLimit == nil {
		t.Fatal("expected speed limit")
	}
	if got := device.SpeedLimit.IP; got != "192.168.100.88" {
		t.Fatalf("expected speed limit ip %q, got %q", "192.168.100.88", got)
	}
	if got := device.SpeedLimit.Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected speed limit mac %q, got %q", "AA:BB:CC:DD:EE:FF", got)
	}
	if got := device.SpeedLimit.UploadSpeed; got != 2048 {
		t.Fatalf("expected speed limit upload %d, got %d", 2048, got)
	}
	if got := device.SpeedLimit.DownloadSpeed; got != 4096 {
		t.Fatalf("expected speed limit download %d, got %d", 4096, got)
	}
	if got := device.SpeedLimit.Comment; got != "limited" {
		t.Fatalf("expected speed limit comment %q, got %q", "limited", got)
	}
	if got := device.SpeedLimit.NetworkAccess; got {
		t.Fatalf("expected speed limit network access false")
	}
	if got := device.SpeedLimit.Enabled; !got {
		t.Fatalf("expected speed limit enabled true")
	}
}

func TestLanDeviceListServicePreservesLegacyBlockAndSpeedLimitPriority(t *testing.T) {
	t.Parallel()

	svc := &LanDeviceListService{
		LanStatusReader: &fakeLanDeviceListLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr:          "192.168.100.1",
					Nexthop:          "192.168.100.254",
					IsDefaultGateway: true,
				}, nil
			},
		},
		InventoryReader: &fakeLanDeviceListInventoryReader{
			readInventoryFn: func(ctx context.Context) (models.LANDevices, error) {
				_ = ctx
				return models.LANDevices{
					{
						IP:         "192.168.100.2",
						Mac:        "AA:BB:CC:DD:EE:FF",
						Intr:       "lan",
						SpeedLimit: &models.LANCtrlSpeedLimitItem{IP: "192.168.100.2", Mac: "AA:BB:CC:DD:EE:FF", NetworkAccess: true},
					},
				}, nil
			},
		},
		DhcpTagReader:          &fakeLanDeviceListDhcpTagReader{},
		HostHintReader:         &fakeLanDeviceListHostHintReader{},
		WifiAssocReader:        &fakeLanDeviceListWifiAssocReader{},
		TrafficStatReader:      &fakeLanDeviceListTrafficStatReader{},
		StaticAssignmentReader: &fakeLanDeviceListStaticAssignmentReader{},
		SpeedLimitReader: &fakeLanDeviceListSpeedLimitReader{
			readSpeedLimitsFn: func(ctx context.Context) (map[string]*models.LANCtrlSpeedLimitItem, map[string]*models.LANCtrlSpeedLimitItem, error) {
				_ = ctx
				return map[string]*models.LANCtrlSpeedLimitItem{
						"AA:BB:CC:DD:EE:FF": {
							Mac:           "AA:BB:CC:DD:EE:00",
							NetworkAccess: false,
						},
					}, map[string]*models.LANCtrlSpeedLimitItem{
						"192.168.100.2": {
							IP:            "192.168.100.88",
							UploadSpeed:   2048,
							DownloadSpeed: 4096,
							Comment:       "limited by profile",
						},
					}, nil
			},
		},
	}

	resp, err := svc.GetListDevices(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp == nil || resp.Result == nil {
		t.Fatal("expected response result")
	}
	if got := len(resp.Result.Devices); got != 1 {
		t.Fatalf("expected 1 device, got %d", got)
	}
	device := resp.Result.Devices[0]
	if device == nil {
		t.Fatal("expected device")
	}
	if device.SpeedLimit == nil {
		t.Fatal("expected speed limit")
	}
	if got := device.SpeedLimit.NetworkAccess; got {
		t.Fatal("expected block item to keep network access disabled")
	}
	if got := device.SpeedLimit.UploadSpeed; got != 2048 {
		t.Fatalf("expected upload speed %d from speed-limit item, got %d", 2048, got)
	}
	if got := device.SpeedLimit.DownloadSpeed; got != 4096 {
		t.Fatalf("expected download speed %d from speed-limit item, got %d", 4096, got)
	}
	if got := device.SpeedLimit.Enabled; !got {
		t.Fatal("expected combined block+speed limit to stay enabled")
	}
	if got := device.SpeedLimit.Mac; got != "AA:BB:CC:DD:EE:00" {
		t.Fatalf("expected mac %q from block item, got %q", "AA:BB:CC:DD:EE:00", got)
	}
	if got := device.SpeedLimit.IP; got != "192.168.100.88" {
		t.Fatalf("expected ip %q from speed-limit item, got %q", "192.168.100.88", got)
	}
	if got := device.SpeedLimit.Comment; got != "limited by profile" {
		t.Fatalf("expected comment %q from speed-limit item, got %q", "limited by profile", got)
	}
}

func TestMergeLanDeviceStaticAssignmentPreservesHostHintAndDoesNotAliasSharedPointer(t *testing.T) {
	t.Parallel()

	device := &models.LANDevice{
		IP:             "192.168.100.2",
		Mac:            "AA:BB:CC:DD:EE:FF",
		Hostname:       "office-pc",
		StaticAssigned: &models.LANStaticAssigned{AssignedIP: "192.168.100.2", AssignedMac: "AA:BB:CC:DD:EE:FF"},
	}
	assignment := &models.LANStaticAssigned{
		AssignedIP:  "10.0.0.10",
		AssignedMac: "AA:BB:CC:DD:EE:FF",
		Hostname:    "",
		TagName:     "office",
	}

	mergeLanDeviceStaticAssignment(device, map[string]*models.LANStaticAssigned{
		"AA:BB:CC:DD:EE:FF": assignment,
	})

	if got := device.Hostname; got != "office-pc" {
		t.Fatalf("expected existing hostname %q to be preserved, got %q", "office-pc", got)
	}
	if device.StaticAssigned == assignment {
		t.Fatal("expected static assignment to be copied before attach")
	}
	if got := assignment.AssignedIP; got != "10.0.0.10" {
		t.Fatalf("expected source assignment ip to stay %q, got %q", "10.0.0.10", got)
	}
	if got := device.StaticAssigned.AssignedIP; got != "192.168.100.2" {
		t.Fatalf("expected attached assignment ip %q, got %q", "192.168.100.2", got)
	}
}

func TestMergeLanDeviceStaticAssignmentInitializesNilPointerAndIgnoresNilMapValue(t *testing.T) {
	t.Parallel()

	device := &models.LANDevice{
		IP:  "192.168.100.2",
		Mac: "AA:BB:CC:DD:EE:FF",
	}

	mergeLanDeviceStaticAssignment(device, map[string]*models.LANStaticAssigned{
		"AA:BB:CC:DD:EE:FF": nil,
	})

	if device.StaticAssigned != nil {
		t.Fatal("expected matched nil map value to be ignored")
	}

	mergeLanDeviceStaticAssignment(device, map[string]*models.LANStaticAssigned{
		"AA:BB:CC:DD:EE:FF": {
			AssignedIP:  "10.0.0.10",
			AssignedMac: "AA:BB:CC:DD:EE:FF",
			TagName:     "office",
		},
	})

	if device.StaticAssigned == nil {
		t.Fatal("expected static assignment to be initialized")
	}
	if got := device.StaticAssigned.AssignedIP; got != "192.168.100.2" {
		t.Fatalf("expected initialized assigned ip %q, got %q", "192.168.100.2", got)
	}
	if got := device.StaticAssigned.AssignedMac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected initialized assigned mac %q, got %q", "AA:BB:CC:DD:EE:FF", got)
	}
}

func TestMergeLanDeviceSpeedLimitSpeedOnly(t *testing.T) {
	t.Parallel()

	device := &models.LANDevice{
		IP:         "192.168.100.2",
		Mac:        "AA:BB:CC:DD:EE:FF",
		SpeedLimit: &models.LANCtrlSpeedLimitItem{IP: "192.168.100.2", Mac: "AA:BB:CC:DD:EE:FF", NetworkAccess: true},
	}

	mergeLanDeviceSpeedLimit(device, nil, map[string]*models.LANCtrlSpeedLimitItem{
		"192.168.100.2": {
			IP:            "192.168.100.88",
			UploadSpeed:   2048,
			DownloadSpeed: 4096,
			Comment:       "limited",
		},
	})

	if got := device.SpeedLimit.IP; got != "192.168.100.88" {
		t.Fatalf("expected speed-only ip %q, got %q", "192.168.100.88", got)
	}
	if got := device.SpeedLimit.Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected mac to stay %q, got %q", "AA:BB:CC:DD:EE:FF", got)
	}
	if got := device.SpeedLimit.NetworkAccess; !got {
		t.Fatal("expected speed-only branch to keep network access enabled")
	}
	if got := device.SpeedLimit.Enabled; !got {
		t.Fatal("expected speed-only branch to enable speed limit")
	}
}

func TestMergeLanDeviceSpeedLimitBlockOnly(t *testing.T) {
	t.Parallel()

	device := &models.LANDevice{
		IP:         "192.168.100.2",
		Mac:        "AA:BB:CC:DD:EE:FF",
		SpeedLimit: &models.LANCtrlSpeedLimitItem{IP: "192.168.100.2", Mac: "AA:BB:CC:DD:EE:FF", NetworkAccess: true},
	}

	mergeLanDeviceSpeedLimit(device, map[string]*models.LANCtrlSpeedLimitItem{
		"AA:BB:CC:DD:EE:FF": {Mac: "AA:BB:CC:DD:EE:FF"},
	}, nil)

	if got := device.SpeedLimit.Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected block-only mac %q, got %q", "AA:BB:CC:DD:EE:FF", got)
	}
	if got := device.SpeedLimit.NetworkAccess; got {
		t.Fatal("expected block-only branch to disable network access")
	}
	if got := device.SpeedLimit.Enabled; !got {
		t.Fatal("expected block-only branch to enable speed limit")
	}
}

func TestMergeLanDeviceSpeedLimitInitializesNilPointerAndIgnoresNilMapValues(t *testing.T) {
	t.Parallel()

	device := &models.LANDevice{
		IP:  "192.168.100.2",
		Mac: "AA:BB:CC:DD:EE:FF",
	}

	mergeLanDeviceSpeedLimit(device, map[string]*models.LANCtrlSpeedLimitItem{
		"11:22:33:44:55:66": nil,
		"AA:BB:CC:DD:EE:FF": nil,
	}, map[string]*models.LANCtrlSpeedLimitItem{
		"10.0.0.9":      nil,
		"192.168.100.2": nil,
	})

	if device.SpeedLimit == nil {
		t.Fatal("expected speed limit to be initialized")
	}
	if got := device.SpeedLimit.IP; got != "192.168.100.2" {
		t.Fatalf("expected initialized speed limit ip %q, got %q", "192.168.100.2", got)
	}
	if got := device.SpeedLimit.Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected initialized speed limit mac %q, got %q", "AA:BB:CC:DD:EE:FF", got)
	}
	if got := device.SpeedLimit.NetworkAccess; !got {
		t.Fatal("expected initialized speed limit network access true")
	}
}

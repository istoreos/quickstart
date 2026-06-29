package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestBuildDeviceInventoryItemInitializesDefaults(t *testing.T) {
	<-initDone
	initMutex.Lock()
	original := d
	d = make(map[int]interface{})
	parse("AA:BB:CC", "Test Vendor")
	t.Cleanup(func() {
		initMutex.Lock()
		d = original
		initMutex.Unlock()
	})
	initMutex.Unlock()

	inputMAC := "aa:bb:cc:dd:ee:ff"
	item, ok := buildDeviceInventoryItem("192.168.100.2", inputMAC)
	if !ok {
		t.Fatalf("expected helper to accept valid ip/mac")
	}
	if item == nil {
		t.Fatalf("expected device item")
	}
	if got := item.IP; got != "192.168.100.2" {
		t.Fatalf("expected IP 192.168.100.2, got %q", got)
	}
	if got := item.Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected normalized MAC, got %q", got)
	}
	if got := item.Vendor; got != "Test Vendor" {
		t.Fatalf("expected vendor to match current lookup behavior, got %q", got)
	}
	if got := item.Intr; got != "lan" {
		t.Fatalf("expected intr lan, got %q", got)
	}
	if item.StaticAssigned == nil {
		t.Fatalf("expected StaticAssigned to be initialized")
	}
	if got := item.StaticAssigned.AssignedMac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected static assigned mac to be normalized MAC, got %q", got)
	}
	if got := item.StaticAssigned.AssignedIP; got != "192.168.100.2" {
		t.Fatalf("expected static assigned ip to be input IP, got %q", got)
	}
	if item.SpeedLimit == nil {
		t.Fatalf("expected SpeedLimit to be initialized")
	}
	if got := item.SpeedLimit.IP; got != "192.168.100.2" {
		t.Fatalf("expected speed limit ip to be input IP, got %q", got)
	}
	if got := item.SpeedLimit.Mac; got != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("expected speed limit mac to be normalized MAC, got %q", got)
	}
	if got := item.SpeedLimit.NetworkAccess; !got {
		t.Fatalf("expected speed limit network access to default true")
	}
}

func TestBuildDeviceInventoryItemRejectsMissingIPOrMac(t *testing.T) {
	if _, ok := buildDeviceInventoryItem("", "aa:bb:cc:dd:ee:ff"); ok {
		t.Fatalf("expected missing ip to be rejected")
	}
	if _, ok := buildDeviceInventoryItem("192.168.100.2", ""); ok {
		t.Fatalf("expected missing mac to be rejected")
	}
}

func TestBuildTrafficStatSnapshotCopiesValues(t *testing.T) {
	snapshot := buildTrafficStatSnapshot(128, 256)
	if got := snapshot.UploadSpeed; got != 128 {
		t.Fatalf("expected upload speed 128, got %d", got)
	}
	if got := snapshot.DownloadSpeed; got != 256 {
		t.Fatalf("expected download speed 256, got %d", got)
	}
}

func TestBuildHostHintHostnameStripsDomainOnce(t *testing.T) {
	if got := buildHostHintHostname("office-printer.local"); got != "office-printer" {
		t.Fatalf("expected hostname office-printer, got %q", got)
	}
}

func TestBuildHostHintHostnameKeepsPlainName(t *testing.T) {
	if got := buildHostHintHostname("nas"); got != "" {
		t.Fatalf("expected plain name to return empty string, got %q", got)
	}
}

func TestDefaultDeviceListReadersBestEffortSpeedLimits(t *testing.T) {
	blockMap, speedMap, err := buildBestEffortSpeedLimitMaps(nil, nil, errors.New("boom"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(blockMap) != 0 {
		t.Fatalf("expected empty block map on error, got %d entries", len(blockMap))
	}
	if len(speedMap) != 0 {
		t.Fatalf("expected empty speed map on error, got %d entries", len(speedMap))
	}

	blockMap, speedMap, err = buildBestEffortSpeedLimitMaps([]*models.LANCtrlSpeedLimitItem{
		{Mac: "AA:BB:CC:DD:EE:FF"},
	}, []*models.LANCtrlSpeedLimitItem{
		{IP: "192.168.100.2"},
	}, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if _, ok := blockMap["AA:BB:CC:DD:EE:FF"]; !ok {
		t.Fatalf("expected block map to contain normalized entry")
	}
	if _, ok := speedMap["192.168.100.2"]; !ok {
		t.Fatalf("expected speed map to contain ip entry")
	}
}

func TestDefaultDeviceListReadersPreloadConfigs(t *testing.T) {
	originalDeviceListLoadConfig := lanDeviceListLoadConfig
	originalRuleStoreLoadConfig := lanSpeedLimitRuleLoadConfig
	var deviceListCalls []string
	var ruleStoreCalls []string
	lanDeviceListLoadConfig = func(name string, forceReload bool) error {
		if !forceReload {
			t.Fatalf("expected forceReload=true for %s", name)
		}
		deviceListCalls = append(deviceListCalls, name)
		return nil
	}
	lanSpeedLimitRuleLoadConfig = func(name string, forceReload bool) error {
		if !forceReload {
			t.Fatalf("expected forceReload=true for %s", name)
		}
		ruleStoreCalls = append(ruleStoreCalls, name)
		return nil
	}
	t.Cleanup(func() {
		lanDeviceListLoadConfig = originalDeviceListLoadConfig
		lanSpeedLimitRuleLoadConfig = originalRuleStoreLoadConfig
	})

	preloadStaticAssignmentConfigs()
	preloadSpeedLimitConfigs()

	wantDeviceList := []string{"dhcp", "floatip"}
	if len(deviceListCalls) != len(wantDeviceList) {
		t.Fatalf("expected %d device list load calls, got %d: %v", len(wantDeviceList), len(deviceListCalls), deviceListCalls)
	}
	for i := range wantDeviceList {
		if deviceListCalls[i] != wantDeviceList[i] {
			t.Fatalf("expected device list call %d to load %q, got %q", i, wantDeviceList[i], deviceListCalls[i])
		}
	}

	wantRuleStore := []string{"eqos", "firewall"}
	if len(ruleStoreCalls) != len(wantRuleStore) {
		t.Fatalf("expected %d rule store load calls, got %d: %v", len(wantRuleStore), len(ruleStoreCalls), ruleStoreCalls)
	}
	for i := range wantRuleStore {
		if ruleStoreCalls[i] != wantRuleStore[i] {
			t.Fatalf("expected rule store call %d to load %q, got %q", i, wantRuleStore[i], ruleStoreCalls[i])
		}
	}
}

type fakeSharedStaticAssignmentListReader struct {
	readStaticAssignmentsFn func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error)
}

type fakeLanSpeedLimitRuleStore struct {
	readRuleListsFn func(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error)
}

func (store *fakeLanSpeedLimitRuleStore) ReadRuleLists(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error) {
	if store.readRuleListsFn != nil {
		return store.readRuleListsFn(ctx)
	}
	return []*models.LANCtrlSpeedLimitItem{}, []*models.LANCtrlSpeedLimitItem{}, nil
}

func (reader *fakeSharedStaticAssignmentListReader) ReadStaticAssignments(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error) {
	if reader.readStaticAssignmentsFn != nil {
		return reader.readStaticAssignmentsFn(ctx, tagList)
	}
	return []*models.LANStaticAssigned{}, nil
}

func TestDefaultStaticAssignmentReaderReusesSharedListReader(t *testing.T) {
	t.Parallel()

	originalCtor := newDefaultStaticAssignmentListReader
	defer func() {
		newDefaultStaticAssignmentListReader = originalCtor
	}()

	called := false
	newDefaultStaticAssignmentListReader = func() StaticAssignmentListReader {
		called = true
		return &fakeSharedStaticAssignmentListReader{
			readStaticAssignmentsFn: func(ctx context.Context, tagList []*models.LANCtrlDhcpTagInfo) ([]*models.LANStaticAssigned, error) {
				_ = ctx
				_ = tagList
				return []*models.LANStaticAssigned{
					{AssignedMac: "AA:BB:CC:DD:EE:FF", AssignedIP: "192.168.100.30"},
				}, nil
			},
		}
	}

	reader := NewDefaultStaticAssignmentReader()
	items, err := reader.ReadStaticAssignments(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !called {
		t.Fatal("expected shared list reader constructor to be used")
	}
	if got := items["AA:BB:CC:DD:EE:FF"]; got == nil || got.AssignedIP != "192.168.100.30" {
		t.Fatalf("unexpected shared static assignment item: %+v", items)
	}
}

func TestDefaultLanDeviceSpeedLimitReaderUsesSharedRuleStore(t *testing.T) {
	t.Parallel()

	reader := &defaultLanDeviceSpeedLimitReader{
		store: &fakeLanSpeedLimitRuleStore{
			readRuleListsFn: func(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error) {
				_ = ctx
				return []*models.LANCtrlSpeedLimitItem{
						{Mac: "AA:BB:CC:DD:EE:FF", NetworkAccess: false},
					}, []*models.LANCtrlSpeedLimitItem{
						{IP: "192.168.100.8", NetworkAccess: true},
					}, nil
			},
		},
	}

	blockMap, speedMap, err := reader.ReadSpeedLimits(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := blockMap["AA:BB:CC:DD:EE:FF"]; got == nil || got.NetworkAccess {
		t.Fatalf("unexpected block map: %+v", blockMap)
	}
	if got := speedMap["192.168.100.8"]; got == nil || !got.NetworkAccess {
		t.Fatalf("unexpected speed map: %+v", speedMap)
	}
}

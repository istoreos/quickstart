package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var lanStaticDeviceListStoreTestMu sync.Mutex

type fakeStaticDeviceDhcpStore struct {
	loadLanStateFn func(ctx context.Context) (*LanDhcpState, error)
}

func (store *fakeStaticDeviceDhcpStore) LoadLanState(ctx context.Context) (*LanDhcpState, error) {
	if store.loadLanStateFn != nil {
		return store.loadLanStateFn(ctx)
	}
	return &LanDhcpState{}, nil
}

func (store *fakeStaticDeviceDhcpStore) ApplyTagConfig(ctx context.Context, input DhcpTagConfigInput) error {
	_ = ctx
	_ = input
	return nil
}

func (store *fakeStaticDeviceDhcpStore) ApplyGatewayConfig(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error {
	_ = ctx
	_ = input
	_ = lanStatus
	return nil
}

func TestBuildStaticAssignmentItemMapsKnownTag(t *testing.T) {
	t.Parallel()

	tagMap := map[string]*models.LANCtrlDhcpTagInfo{
		"guest": {
			TagName:  "guest",
			TagTitle: "Guest",
			Gateway:  "192.168.100.254",
		},
	}

	item, ok := buildStaticAssignmentItem(
		"aa:bb:cc:dd:ee:ff",
		"192.168.100.20",
		true,
		"printer",
		"guest",
		true,
		"",
		false,
		tagMap,
	)
	if !ok {
		t.Fatal("expected item to be built")
	}
	if item.AssignedMac != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("AssignedMac = %q, want %q", item.AssignedMac, "AA:BB:CC:DD:EE:FF")
	}
	if item.AssignedIP != "192.168.100.20" {
		t.Fatalf("AssignedIP = %q, want %q", item.AssignedIP, "192.168.100.20")
	}
	if !item.BindIP {
		t.Fatal("expected BindIP to be true")
	}
	if item.Hostname != "printer" {
		t.Fatalf("Hostname = %q, want %q", item.Hostname, "printer")
	}
	if item.DhcpGateway != "192.168.100.254" || item.TagName != "guest" || item.TagTitle != "Guest" {
		t.Fatalf("unexpected tag mapping: %+v", item)
	}
}

func TestBuildStaticAssignmentItemUsesTagTitleOnlyFallback(t *testing.T) {
	t.Parallel()

	item, ok := buildStaticAssignmentItem(
		"aa:bb:cc:dd:ee:01",
		"",
		false,
		"default-item",
		"",
		false,
		"default",
		true,
		map[string]*models.LANCtrlDhcpTagInfo{},
	)
	if !ok {
		t.Fatal("expected item to be built")
	}
	if item.BindIP {
		t.Fatal("expected BindIP to stay false")
	}
	if item.TagTitle != "default" {
		t.Fatalf("TagTitle = %q, want %q", item.TagTitle, "default")
	}
	if item.TagName != "" {
		t.Fatalf("TagName = %q, want empty string", item.TagName)
	}
	if item.DhcpGateway != "" {
		t.Fatalf("DhcpGateway = %q, want empty string", item.DhcpGateway)
	}
}

func TestBuildStaticAssignmentItemLeavesUnknownTagFieldsEmpty(t *testing.T) {
	t.Parallel()

	item, ok := buildStaticAssignmentItem(
		"aa:bb:cc:dd:ee:02",
		"192.168.100.21",
		true,
		"unknown-tag-device",
		"missing",
		true,
		"",
		false,
		map[string]*models.LANCtrlDhcpTagInfo{},
	)
	if !ok {
		t.Fatal("expected item to be built")
	}
	if item.DhcpGateway != "" || item.TagName != "" || item.TagTitle != "" {
		t.Fatalf("expected unknown tag fields to stay empty, got %+v", item)
	}
}

func TestBuildStaticAssignmentItemSkipsHostWithoutMac(t *testing.T) {
	t.Parallel()

	if _, ok := buildStaticAssignmentItem(
		"",
		"192.168.100.30",
		true,
		"broken-host",
		"",
		false,
		"",
		false,
		map[string]*models.LANCtrlDhcpTagInfo{},
	); ok {
		t.Fatal("expected host without mac to be skipped")
	}
}

func TestDefaultStaticAssignmentListReaderUsesDhcpSectionsAndTagFallbacks(t *testing.T) {
	lanStaticDeviceListStoreTestMu.Lock()
	defer lanStaticDeviceListStoreTestMu.Unlock()

	originalLoadConfig := lanStaticDeviceListLoadConfig
	originalGetSections := lanStaticDeviceListGetSections
	originalGetLast := lanStaticDeviceListGetLast
	t.Cleanup(func() {
		lanStaticDeviceListLoadConfig = originalLoadConfig
		lanStaticDeviceListGetSections = originalGetSections
		lanStaticDeviceListGetLast = originalGetLast
	})

	loadCalls := []string{}
	lanStaticDeviceListLoadConfig = func(name string, forceReload bool) error {
		loadCalls = append(loadCalls, name)
		if !forceReload {
			t.Fatalf("expected forceReload=true for %s", name)
		}
		return nil
	}
	lanStaticDeviceListGetSections = func(config, sectionType string) ([]string, bool) {
		if config != "dhcp" || sectionType != "host" {
			t.Fatalf("unexpected get sections call: %s %s", config, sectionType)
		}
		return []string{"host_ok", "host_default", "host_skip"}, true
	}
	lanStaticDeviceListGetLast = func(config, section, option string) (string, bool) {
		values := map[string]map[string]map[string]struct {
			value string
			ok    bool
		}{
			"dhcp": {
				"host_ok": {
					"mac":       {value: "aa:bb:cc:dd:ee:ff", ok: true},
					"ip":        {value: "192.168.100.20", ok: true},
					"tag":       {value: "guest", ok: true},
					"tag_title": {value: "ignored", ok: true},
					"name":      {value: "printer", ok: true},
				},
				"host_default": {
					"mac":       {value: "11:22:33:44:55:66", ok: true},
					"tag":       {value: "", ok: false},
					"tag_title": {value: "default", ok: true},
					"name":      {value: "fallback", ok: true},
				},
				"host_skip": {
					"mac": {value: "", ok: false},
				},
			},
		}

		if got, ok := values[config][section][option]; ok {
			return got.value, got.ok
		}
		return "", false
	}

	reader := NewDefaultStaticAssignmentListReader()
	items, err := reader.ReadStaticAssignments(context.Background(), []*models.LANCtrlDhcpTagInfo{
		{TagName: "guest", TagTitle: "Guest", Gateway: "192.168.100.254"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(loadCalls) != 1 || loadCalls[0] != "dhcp" {
		t.Fatalf("unexpected load calls: %+v", loadCalls)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 static assignments, got %d", len(items))
	}
	if got := items[0]; got.AssignedMac != "AA:BB:CC:DD:EE:FF" || got.DhcpGateway != "192.168.100.254" || got.TagTitle != "Guest" {
		t.Fatalf("unexpected tagged item: %+v", got)
	}
	if got := items[1]; got.AssignedMac != "11:22:33:44:55:66" || got.TagTitle != "default" || got.TagName != "" || got.BindIP {
		t.Fatalf("unexpected default-tag item: %+v", got)
	}
}

func TestDefaultStaticAssignmentListReaderReturnsEmptyWithoutHostSections(t *testing.T) {
	lanStaticDeviceListStoreTestMu.Lock()
	defer lanStaticDeviceListStoreTestMu.Unlock()

	originalLoadConfig := lanStaticDeviceListLoadConfig
	originalGetSections := lanStaticDeviceListGetSections
	t.Cleanup(func() {
		lanStaticDeviceListLoadConfig = originalLoadConfig
		lanStaticDeviceListGetSections = originalGetSections
	})

	lanStaticDeviceListLoadConfig = func(name string, forceReload bool) error {
		if name != "dhcp" || !forceReload {
			t.Fatalf("unexpected load config call: %s force=%v", name, forceReload)
		}
		return nil
	}
	lanStaticDeviceListGetSections = func(config, sectionType string) ([]string, bool) {
		if config != "dhcp" || sectionType != "host" {
			t.Fatalf("unexpected get sections call: %s %s", config, sectionType)
		}
		return nil, false
	}

	reader := NewDefaultStaticAssignmentListReader()
	items, err := reader.ReadStaticAssignments(context.Background(), []*models.LANCtrlDhcpTagInfo{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty assignments, got %d", len(items))
	}
}

func TestLanStaticDeviceDhcpTagReaderReturnsStoreError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("load lan state failed")
	reader := &defaultLanStaticDeviceDhcpTagReader{
		store: &fakeStaticDeviceDhcpStore{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return nil, wantErr
			},
		},
	}

	_, err := reader.ReadDhcpTags(context.Background(), LanStatusSnapshot{LanAddr: "192.168.100.1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected error %v, got %v", wantErr, err)
	}
}

func TestLanStaticDeviceDhcpTagReaderBuildsAutoCreatedTagsFromSharedState(t *testing.T) {
	t.Parallel()

	reader := &defaultLanStaticDeviceDhcpTagReader{
		store: &fakeStaticDeviceDhcpStore{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{
					FloatIP: &FloatIPSnapshot{
						Enabled: true,
						SetIP:   "192.168.100.2",
						CheckIP: "1.1.1.1",
					},
				}, nil
			},
		},
	}

	tagList, err := reader.ReadDhcpTags(context.Background(), LanStatusSnapshot{
		LanAddr:          "192.168.100.1",
		Nexthop:          "192.168.100.254",
		IsDefaultGateway: true,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(tagList) != 4 {
		t.Fatalf("expected 4 tags, got %d", len(tagList))
	}
	if tagList[2].TagTitle != "floatip" || tagList[2].Gateway != "192.168.100.2" {
		t.Fatalf("unexpected floatip tag: %+v", tagList[2])
	}
	if tagList[3].TagTitle != "bypass" || tagList[3].Gateway != "1.1.1.1" {
		t.Fatalf("unexpected bypass tag: %+v", tagList[3])
	}
}

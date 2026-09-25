package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

type fakeNetworkRulesStore struct {
	snapshot networkRulesSnapshot
	applies  int
	err      error
}

func (store *fakeNetworkRulesStore) Read(context.Context) (networkRulesSnapshot, error) {
	return store.snapshot, store.err
}
func (store *fakeNetworkRulesStore) Apply(context.Context, []*models.NetworkRule) error {
	store.applies++
	return store.err
}

func networkRulesFixture(count int) []*models.NetworkRule {
	rules := make([]*models.NetworkRule, 0, count)
	for index := 0; index < count; index++ {
		rules = append(rules, &models.NetworkRule{ID: stableNetworkRuleID("static", string(rune(index))), Kind: "static", MAC: "AA:BB:CC:DD:EE:01", Status: "active"})
	}
	return rules
}

func TestNetworkRulesBulkPlanLimitsAndVersionConflict(t *testing.T) {
	rules := networkRulesFixture(129)
	store := &fakeNetworkRulesStore{snapshot: networkRulesSnapshot{Rules: rules, Version: "v1"}}
	module := NewNetworkRulesModule(store)
	ids := make([]string, 129)
	for index := range rules {
		ids[index] = rules[index].ID
	}
	tooMany, _ := module.Plan(context.Background(), &models.NetworkRulesBulkRequest{Action: "delete", RuleIDs: ids})
	if tooMany.Result.Plan.Error == nil || tooMany.Result.Plan.CanApply {
		t.Fatalf("plan=%#v", tooMany.Result.Plan)
	}
	stale, _ := module.Apply(context.Background(), &models.NetworkRulesBulkRequest{Action: "delete", RuleIDs: ids[:1], ExpectedVersion: "stale"})
	if stale.Result.Error == nil || stale.Result.Error.Code != "conflict" || store.applies != 0 {
		t.Fatalf("stale=%#v applies=%d", stale.Result, store.applies)
	}
	applied, _ := module.Apply(context.Background(), &models.NetworkRulesBulkRequest{Action: "delete", RuleIDs: ids[:2], ExpectedVersion: "v1"})
	if !applied.Result.Changed || applied.Result.Plan.AffectedCount != 2 || store.applies != 1 {
		t.Fatalf("applied=%#v applies=%d", applied.Result, store.applies)
	}
}

func TestSystemNetworkRulesAggregatesStaticSpeedAccessRouteAndOrphans(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.100.20", Hostname: "known", Online: true}}}}}, &now, 2048)
	policyStore := availablePolicyStore()
	policyStore.rules = &models.DevicePolicyRulesResult{Static: []*models.LANStaticAssigned{{AssignedMac: "AA:BB:CC:DD:EE:01", AssignedIP: "192.168.100.20", BindIP: true}}, Speed: []*models.LANCtrlSpeedLimitItem{{Mac: "AA:BB:CC:DD:EE:01", IP: "192.168.100.20", NetworkAccess: false}, {IP: "192.168.100.99", NetworkAccess: true, UploadSpeed: 10, DownloadSpeed: 20}}}
	gatewayState := gatewayPolicyTestState()
	gatewayState.Hosts = []gatewayPolicyHost{{MAC: "AA:BB:CC:DD:EE:01", TagName: "legacy_route"}}
	store := &systemNetworkRulesStore{inventory: inventory, policy: newDevicePolicyModuleForTest(policyStore), gateway: NewGatewayPolicyModule(inventory, &fakeGatewayPolicyStore{state: gatewayState})}
	snapshot, err := store.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	kinds, orphaned := map[string]int{}, 0
	for _, rule := range snapshot.Rules {
		kinds[rule.Kind]++
		if rule.Orphaned {
			orphaned++
		}
	}
	if kinds["static"] != 1 || kinds["route"] != 1 || kinds["access"] != 1 || kinds["speed"] != 1 || orphaned != 1 || snapshot.Version == "" {
		t.Fatalf("kinds=%#v orphaned=%d rules=%#v", kinds, orphaned, snapshot.Rules)
	}
}

func TestClearNetworkRuleRoutePreservesAddressReservation(t *testing.T) {
	directory := t.TempDir()
	config := "config host 'device'\n\toption mac 'AA:BB:CC:DD:EE:01'\n\toption ip '192.168.100.20'\n\toption name 'printer'\n\toption tag 'route'\n\toption tag_title 'bypass'\n"
	if err := os.WriteFile(filepath.Join(directory, "dhcp"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if err := clearNetworkRuleRouteAt(directory, "AA:BB:CC:DD:EE:01"); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(directory)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	if tag, ok := tree.GetLast("dhcp", "device", "tag"); ok || tag != "" {
		t.Fatalf("tag=%q ok=%v", tag, ok)
	}
	if ip, _ := tree.GetLast("dhcp", "device", "ip"); ip != "192.168.100.20" {
		t.Fatalf("ip=%q", ip)
	}
	if name, _ := tree.GetLast("dhcp", "device", "name"); name != "printer" {
		t.Fatalf("name=%q", name)
	}
}

func TestClearNetworkRuleStaticPreservesInternetRoute(t *testing.T) {
	directory := t.TempDir()
	config := "config host 'device'\n\toption mac 'AA:BB:CC:DD:EE:01'\n\toption ip '192.168.100.20'\n\toption name 'printer'\n\toption tag 'route'\n\toption tag_title 'bypass'\n"
	if err := os.WriteFile(filepath.Join(directory, "dhcp"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	if err := clearNetworkRuleStaticAt(directory, "AA:BB:CC:DD:EE:01"); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(directory)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	if ip, ok := tree.GetLast("dhcp", "device", "ip"); ok || ip != "" {
		t.Fatalf("ip=%q ok=%v", ip, ok)
	}
	if tag, _ := tree.GetLast("dhcp", "device", "tag"); tag != "route" {
		t.Fatalf("tag=%q", tag)
	}
}

func TestNetworkRulesBatchFailureRestoresAllSnapshots(t *testing.T) {
	originalStatic, originalSpeed, originalRoute, originalTake, originalRestore := networkRulesDeleteStatic, networkRulesDeleteSpeed, networkRulesClearRoute, networkRulesTakeSnapshots, networkRulesRestoreSnapshots
	t.Cleanup(func() {
		networkRulesDeleteStatic, networkRulesDeleteSpeed, networkRulesClearRoute, networkRulesTakeSnapshots, networkRulesRestoreSnapshots = originalStatic, originalSpeed, originalRoute, originalTake, originalRestore
	})
	networkRulesTakeSnapshots = func([]string) ([]networkRulesFileSnapshot, error) {
		return []networkRulesFileSnapshot{{Path: "dhcp", Data: []byte("before"), Exists: true}}, nil
	}
	networkRulesDeleteStatic = func(context.Context, *models.NetworkRule) error { return nil }
	networkRulesDeleteSpeed = func(context.Context, *models.NetworkRule) error { return errors.New("second write failed") }
	restored := false
	networkRulesRestoreSnapshots = func(context.Context, []networkRulesFileSnapshot) error { restored = true; return nil }
	err := (&systemNetworkRulesStore{}).Apply(context.Background(), []*models.NetworkRule{{Kind: "static"}, {Kind: "speed"}})
	if err == nil || !restored {
		t.Fatalf("err=%v restored=%v", err, restored)
	}
}

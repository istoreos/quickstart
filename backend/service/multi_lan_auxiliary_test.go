package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMultiLANAuditSameMACHasSeparateEventsAndRestartState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	module := NewNetworkAuditModule(path, nil)
	const id = "mac:02:00:00:00:00:55"
	for _, network := range []string{"lan", "office"} {
		ctx := inventoryLANContext(network)
		response := trafficInventoryResponse(id, "192.0.2.55")
		response.Result.Devices[0].NetworkID = network
		response.Result.Devices[0].Online = true
		module.observeScopedInventory(ctx, response)
	}
	a, err := module.forContext(inventoryLANContext("lan")).List(id, 20)
	if err != nil {
		t.Fatal(err)
	}
	b, err := module.forContext(inventoryLANContext("office")).List(id, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Result.Events) != 1 || len(b.Result.Events) != 1 || a.Result.Events[0].NetworkID != "lan" || b.Result.Events[0].NetworkID != "office" || a.Result.Events[0].ID == b.Result.Events[0].ID {
		t.Fatalf("audit identity collision: a=%+v b=%+v", a.Result, b.Result)
	}
	restarted := NewNetworkAuditModule(path, nil)
	response := trafficInventoryResponse(id, "192.0.2.55")
	response.Result.Devices[0].NetworkID = "office"
	response.Result.Devices[0].Online = true
	restarted.observeScopedInventory(inventoryLANContext("office"), response)
	repeated, err := restarted.forContext(inventoryLANContext("office")).List(id, 20)
	if err != nil || len(repeated.Result.Events) != 1 {
		t.Fatal("restart repeated new-device event", err)
	}
}

func TestMultiLANBundleScopeAndOldReaderProtection(t *testing.T) {
	path := t.TempDir()
	groups := NewDeviceGroupModule(newJSONDeviceGroupStore(filepath.Join(path, "groups.json")), nil)
	traffic := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(path, "traffic.json")), nil)
	module := NewPolicyBundleModule(groups, traffic, NewNetworkAuditModule(filepath.Join(path, "audit.json"), nil))
	now := time.Now()
	groups.now = func() time.Time { return now }
	office := inventoryLANContext("office")
	exported, err := module.Export(office)
	if err != nil {
		t.Fatal(err)
	}
	bundle := exported.Result.Bundle
	if bundle.NetworkID != "office" || bundle.SchemaVersion != 2 {
		t.Fatal("old readers could accept an additional-LAN backup")
	}
	rejected, err := module.ImportPlan(inventoryLANContext("lan"), &models.PolicyImportRequest{Bundle: bundle})
	if err != nil || rejected.Result.Error == nil || rejected.Result.Error.Code != "bundle_network_mismatch" {
		t.Fatal("cross-network import accepted", err)
	}
	legacy := *bundle
	legacy.SchemaVersion = 1
	legacy.NetworkID = ""
	rejected, err = module.ImportPlan(office, &models.PolicyImportRequest{Bundle: &legacy})
	if err != nil || rejected.Result.Error == nil || rejected.Result.Error.Code != "bundle_network_mismatch" {
		t.Fatal("unscoped old backup assigned to new LAN", err)
	}
}

func TestMultiLANRuleCleanupKeepsSameMACOtherLAN(t *testing.T) {
	dir := t.TempDir()
	const mac = "02:00:00:00:00:55"
	raw := "config dnsmasq 'main'\nconfig dhcp 'pool_a'\n option interface 'lan'\n option instance 'main'\nconfig dhcp 'pool_b'\n option interface 'office'\n option instance 'main'\n"
	for _, item := range []struct{ name, network, ip, device string }{{"a", "lan", "192.0.2.55", "br-lan"}, {"b", "office", "198.51.100.55", "br-office"}} {
		raw += fmt.Sprintf("config host '%s'\n option mac '%s'\n option ip '%s'\n option name 'printer-%s'\n option tag 'route_%s'\n option quickstart_network '%s'\n option quickstart_instance 'main'\n list match_tag '%s'\n", item.name, mac, item.ip, item.name, item.name, item.network, item.device)
	}
	path := filepath.Join(dir, "dhcp")
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "office", Device: "br-office"})
	for _, kind := range []string{"route", "static"} {
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		var err error
		if kind == "route" {
			err = clearNetworkRuleRouteAt(ctx, dir, mac)
		} else {
			err = clearNetworkRuleStaticAt(ctx, dir, mac)
		}
		if err != nil {
			t.Fatal(err)
		}
		tree := uci.NewTree(dir)
		if err := tree.LoadConfig("dhcp", true); err != nil {
			t.Fatal(err)
		}
		ip, _ := tree.GetLast("dhcp", "a", "ip")
		tag, _ := tree.GetLast("dhcp", "a", "tag")
		if ip != "192.0.2.55" || tag != "route_a" {
			t.Fatal("cleanup mutated the other LAN")
		}
		ip, _ = tree.GetLast("dhcp", "b", "ip")
		tag, _ = tree.GetLast("dhcp", "b", "tag")
		if kind == "route" && (ip != "198.51.100.55" || tag != "") {
			t.Fatal("route cleanup changed its reservation")
		}
		if kind == "static" && (ip != "" || tag != "route_b") {
			t.Fatal("reservation cleanup changed its route")
		}
	}
}

type rejectingImportApplier struct{}

func (rejectingImportApplier) Apply(context.Context, *models.EffectiveGroupPolicy) error {
	return errors.New("injected group failure")
}

func TestMultiLANImportRestoresRemovedQuotaAndFirewallOnApplyFailure(t *testing.T) {
	dir := t.TempDir()
	oldDir, oldApply, oldReload := deviceRestrictionConfigDir, deviceAccessRestrictionApply, networkRulesReloadAfterRestore
	deviceRestrictionConfigDir = func() string { return dir }
	deviceAccessRestrictionApply = func(context.Context) error { return nil }
	networkRulesReloadAfterRestore = func(context.Context) error { return nil }
	defer func() {
		deviceRestrictionConfigDir, deviceAccessRestrictionApply, networkRulesReloadAfterRestore = oldDir, oldApply, oldReload
	}()
	raw := "config zone 'shared'\n option name 'shared'\n list network 'office'\nconfig zone 'wan'\n option name 'wan'\n list network 'wan'\n"
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	const mac = "02:00:00:00:00:55"
	const id = "mac:02:00:00:00:00:55"
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{NetworkID: "office", MAC: mac, IPv4: "198.51.100.55", Online: true}}}}}, &now, 100)
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "office", Device: "br-office", FirewallZones: []string{"shared"}})
	policy := newDevicePolicyModuleForTest(&systemDevicePolicyStore{inventory: inventory})
	traffic := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(dir, "traffic.json")), policy)
	if _, err := traffic.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}); err != nil {
		t.Fatal(err)
	}
	if err := traffic.Record(ctx, insightTraffic(id, 0, 0)); err != nil {
		t.Fatal(err)
	}
	if err := traffic.Record(ctx, insightTraffic(id, 120, 0)); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(dir, "firewall"))
	if err != nil {
		t.Fatal(err)
	}
	groups := NewDeviceGroupModule(newJSONDeviceGroupStore(filepath.Join(dir, "groups.json")), rejectingImportApplier{})
	groups.inventory = inventory
	module := NewPolicyBundleModule(groups, traffic, nil)
	access := false
	bundle := &models.PolicyBundle{SchemaVersion: 2, NetworkID: "office", Scope: "device_groups_and_quotas", Groups: []*models.DeviceGroup{{ID: "new", Name: "new", Members: []string{id}, Policy: &models.GroupPolicy{Access: &access}}}, DevicePolicies: map[string]*models.GroupPolicy{}, Quotas: map[string]*models.TrafficQuota{}}
	plan, err := module.ImportPlan(ctx, &models.PolicyImportRequest{Bundle: bundle})
	if err != nil || plan.Result.Error != nil {
		t.Fatalf("plan=%+v %v", plan, err)
	}
	response, err := module.ImportApply(ctx, &models.PolicyImportRequest{Bundle: bundle, Confirmed: true, PlanChecksum: plan.Result.Plan.Checksum, ExpectedGroupVersion: plan.Result.Plan.GroupVersion})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "rolled_back" || !strings.Contains(response.Result.Error.Message, "imported group policies") {
		t.Fatalf("import failed to recover: %+v %v", response, err)
	}
	after, err := os.ReadFile(filepath.Join(dir, "firewall"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("quota firewall was not restored", err)
	}
	status, err := traffic.Get(ctx, id, "today")
	if err != nil || status.Result.Quota.UsedBytes != 120 || !status.Result.Quota.Exceeded {
		t.Fatal("quota usage was reset during rollback", err)
	}
	remaining, err := groups.List(ctx)
	if err != nil || len(remaining.Result.Groups) != 0 {
		t.Fatal("group document was not restored", err)
	}
}

func TestMultiLANWebhookUsesLocalSinkAndNetworkDeduplication(t *testing.T) {
	events := make(chan models.NetworkAuditEvent, 4)
	sink := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var event models.NetworkAuditEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Error(err)
		}
		events <- event
		w.WriteHeader(http.StatusNoContent)
	}))
	defer sink.Close()
	path := filepath.Join(t.TempDir(), "audit.json")
	audit := NewNetworkAuditModule(path, nil)
	const id = "mac:02:00:00:00:00:55"
	observe := func(module *NetworkAuditModule, network string) {
		response := trafficInventoryResponse(id, "192.0.2.55")
		response.Result.Devices[0].NetworkID = network
		response.Result.Devices[0].Online = true
		module.observeScopedInventory(inventoryLANContext(network), response)
	}
	for _, network := range []string{"lan", "office"} {
		if _, err := audit.forContext(inventoryLANContext(network)).SetWebhook(&models.WebhookConfig{Enabled: true, URL: sink.URL, Events: []string{"new_device"}}); err != nil {
			t.Fatal(err)
		}
		observe(audit, network)
	}
	seen := map[string]string{}
	for index := 0; index < 2; index++ {
		select {
		case event := <-events:
			if event.DeviceID == id || event.NetworkID == "" {
				t.Fatal("webhook lost scope or leaked raw device identity")
			}
			seen[event.NetworkID] = event.ID
		case <-time.After(2 * time.Second):
			t.Fatal("local webhook timed out")
		}
	}
	if len(seen) != 2 || seen["lan"] == seen["office"] {
		t.Fatalf("cross-network dedup collision: %v", seen)
	}
	observe(NewNetworkAuditModule(path, nil), "office")
	select {
	case event := <-events:
		t.Fatalf("restart duplicated event: %+v", event)
	case <-time.After(100 * time.Millisecond):
	}
}

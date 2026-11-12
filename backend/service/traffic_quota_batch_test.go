package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func TestQuota256PhysicalRulesUseOneReload(t *testing.T) {
	dir := t.TempDir()
	oldDir, oldApply := deviceRestrictionConfigDir, deviceAccessRestrictionApply
	deviceRestrictionConfigDir = func() string { return dir }
	reloads := 0
	deviceAccessRestrictionApply = func(context.Context) error { reloads++; return nil }
	defer func() { deviceRestrictionConfigDir, deviceAccessRestrictionApply = oldDir, oldApply }()
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte("config zone 'local'\n option name 'lan'\nconfig zone 'outside'\n option name 'wan'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	source := deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{}}
	response := &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{Items: []*models.DeviceTrafficItem{}}}
	requests := []*models.TrafficQuotaRequest{}
	for i := 1; i <= 256; i++ {
		mac := fmt.Sprintf("02:00:00:00:%02x:%02x", i>>8, i&255)
		id := "mac:" + mac
		source.ARP = append(source.ARP, deviceInventoryObservation{MAC: mac, IPv4: fmt.Sprintf("192.0.2.%d", i%250+2), Online: true})
		response.Result.Items = append(response.Result.Items, &models.DeviceTrafficItem{DeviceID: id, State: "ready", UploadBytes: 120})
		requests = append(requests, &models.TrafficQuotaRequest{DeviceID: id, Enabled: false})
	}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{source}}, &now, 1024)
	inventory.cacheTTL = time.Second * 10
	policy := newDevicePolicyModuleForTest(&systemDevicePolicyStore{inventory: inventory})
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(dir, "traffic.json")), policy)
	module.now = func() time.Time { return now }
	if err := module.load(); err != nil {
		t.Fatal(err)
	}
	for _, item := range response.Result.Items {
		module.document.Quotas[item.DeviceID] = &models.TrafficQuota{DeviceID: item.DeviceID, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}
		module.document.Baselines[item.DeviceID] = trafficCounterBaseline{}
	}
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "lan", Device: "br-lan", State: "up", FirewallZones: []string{"lan"}})
	if err := module.Record(ctx, response); err != nil {
		t.Fatal(err)
	}
	if reloads != 1 {
		t.Fatalf("block reloads=%d", reloads)
	}
	for _, item := range response.Result.Items {
		allowed, err := readDeviceAccessPolicyAt(ctx, dir, item.DeviceID[4:])
		if err != nil || allowed || !module.document.QuotaRuntime[item.DeviceID].Blocked {
			t.Fatal("device was not blocked", item.DeviceID, err)
		}
	}
	if _, err := module.SetQuotaBatch(ctx, requests); err != nil {
		t.Fatal(err)
	}
	if reloads != 2 {
		t.Fatalf("cancel reloads=%d", reloads)
	}
	for _, item := range response.Result.Items {
		allowed, err := readDeviceAccessPolicyAt(ctx, dir, item.DeviceID[4:])
		if err != nil || !allowed {
			t.Fatal("device did not recover", item.DeviceID, err)
		}
	}
}

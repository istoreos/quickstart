package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

type accessLiveScopeReader struct {
	scope atomic.Pointer[models.LANScope]
}

func (reader *accessLiveScopeReader) Read(context.Context) ([]*models.LANScope, error) {
	return []*models.LANScope{reader.scope.Load()}, nil
}

func runLegacyQuotaPackets(t *testing.T, base, phase string) {
	t.Helper()
	dir := filepath.Join(base, "config")
	now := time.Now()
	var reloads atomic.Int64
	oldDir, oldApply, oldRestriction := deviceRestrictionConfigDir, deviceAccessRestrictionApply, deviceRestrictionApply
	deviceRestrictionConfigDir = func() string { return dir }
	deviceRestrictionApply = func(context.Context, []string) error { return nil }
	deviceAccessRestrictionApply = func(ctx context.Context) error {
		out, err := exec.CommandContext(ctx, "sh", filepath.Join(base, "load-firewall")).CombinedOutput()
		reloads.Add(1)
		if err != nil {
			return fmt.Errorf("isolated fw4: %w: %s", err, out)
		}
		return nil
	}
	defer func() {
		deviceRestrictionConfigDir, deviceAccessRestrictionApply, deviceRestrictionApply = oldDir, oldApply, oldRestriction
	}()
	const mac = "02:00:00:00:00:55"
	const id = "mac:02:00:00:00:00:55"
	device := os.Getenv("QUICKSTART_MULTI_LAN_TRAFFIC_LIVE_DEVICE")
	if device == "" {
		device = "lana"
	}
	if device != "lana" && device != "lana_new" {
		t.Fatal("unsafe live device")
	}
	scope := &models.LANScope{NetworkID: "lan", Device: device, State: "up", FirewallZones: []string{"shared"}}
	ctx := withLANScope(context.Background(), scope)
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{NetworkID: "lan", MAC: mac, IPv4: "192.0.2.55", Online: true}}}}}, &now, 100)
	policy := newDevicePolicyModuleForTest(&systemDevicePolicyStore{inventory: inventory})
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(base, "legacy-quota.json")), policy)
	if phase == "legacy-seed" {
		if err := writeDeviceAccessPolicyAt(ctx, dir, mac, false); err != nil {
			t.Fatal(err)
		}
		tree := uci.NewTree(dir)
		if err := tree.LoadConfig("firewall", true); err != nil {
			t.Fatal(err)
		}
		sections, _ := tree.GetSections("firewall", "rule")
		for _, section := range sections {
			source, _ := tree.GetLast("firewall", section, "src_mac")
			if normalizeInventoryMAC(source) != mac {
				continue
			}
			for _, option := range []string{"quickstart_access_version", "quickstart_network", "device", "direction"} {
				tree.Del("firewall", section, option)
			}
		}
		if err := tree.Commit(); err != nil {
			t.Fatal(err)
		}
		if err := deviceAccessRestrictionApply(ctx); err != nil {
			t.Fatal(err)
		}
		if err := module.load(); err != nil {
			t.Fatal(err)
		}
		start, _ := trafficQuotaPeriod(now, "daily")
		module.document.Quotas[id] = &models.TrafficQuota{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}
		module.document.QuotaRuntime[id] = trafficQuotaRuntime{PeriodStart: start.Format(time.RFC3339), Blocked: true, RestoreAccess: true}
		if err := module.persist(now); err != nil {
			t.Fatal(err)
		}
		return
	}
	if phase == "legacy-startup" || phase == "legacy-rebind" {
		if err := module.load(); err != nil {
			t.Fatal(err)
		}
		delete(module.document.Quotas, id)
		delete(module.document.QuotaRuntime, id)
		if err := module.persist(now); err != nil {
			t.Fatal(err)
		}
		reader := &accessLiveScopeReader{}
		initial := *scope
		if phase == "legacy-rebind" {
			initial.Device = "lana"
		}
		reader.scope.Store(&initial)
		resolver := &LANScopeModule{reader: reader, now: time.Now}
		backend := &ServiceBackend{lanScopes: resolver}
		if phase == "legacy-rebind" {
			current, err := policy.Get(ctx, id)
			if err != nil || current.Result.Policy.Capabilities["access"].State != "error" {
				t.Fatal("stale binding reported available", err, current)
			}
		}
		applier := &defaultGroupPolicyApplier{policy: policy}
		if err := applier.Apply(ctx, &models.EffectiveGroupPolicy{DeviceID: id, NetworkAccess: true, Managed: []string{"access"}}); err != nil {
			t.Fatal(err)
		}
		worker, stop := context.WithCancel(context.Background())
		backend.StartLANPolicyReconciliation(worker)
		if phase == "legacy-rebind" {
			reader.scope.Store(scope)
			deadline := time.Now().Add(40 * time.Second)
			for reloads.Load() == 0 && time.Now().Before(deadline) {
				time.Sleep(100 * time.Millisecond)
			}
			if reloads.Load() != 1 {
				stop()
				t.Fatal("periodic worker did not restore rebound rule", reloads.Load())
			}
		}
		stop()
		if err := reconcileDeviceAccessBindings(context.WithValue(ctx, lanScopeResolverKey{}, resolver)); err != nil {
			t.Fatal(err)
		}
		if err := applier.Apply(ctx, &models.EffectiveGroupPolicy{DeviceID: id, NetworkAccess: false, Managed: []string{"access"}}); err != nil {
			t.Fatal(err)
		}
	} else if phase == "legacy-cancel" {
		result, err := module.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: id, Enabled: false})
		if err != nil || result.Result.Error != nil {
			t.Fatal(err, result)
		}
	} else if phase == "legacy-reset" || phase == "legacy-enforce" {
		if phase == "legacy-reset" {
			module.now = func() time.Time { return now.Add(24 * time.Hour) }
		}
		if err := module.Record(ctx, &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{}}); err != nil {
			t.Fatal(err)
		}
	} else if phase == "legacy-notify" {
		result, err := module.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 1000, Action: "notify"})
		if err != nil || result.Result.Error != nil {
			t.Fatal(err, result)
		}
	} else if phase == "legacy-group-cancel" {
		applier := &defaultGroupPolicyApplier{policy: policy, traffic: module}
		result := applier.ApplyBatch(ctx, []*models.EffectiveGroupPolicy{{DeviceID: id, Managed: []string{"quota"}, Quota: &models.GroupQuotaPolicy{Enabled: false}}}, now)
		if result.Failed != 0 || result.Reloads["firewall"] != 1 {
			t.Fatalf("quota-only group failed: %+v", result)
		}
	} else if phase == "legacy-recover" {
		result, err := policy.Apply(ctx, &models.DevicePolicyApplyRequest{DeviceID: id, Kind: "access", Access: &models.DeviceAccessPolicy{NetworkAccess: true}})
		if err != nil || result.Result.Error != nil {
			t.Fatal(err, result)
		}
	} else if phase != "legacy-verify" {
		t.Fatal("unknown legacy packet phase")
	}
	result, err := module.Get(ctx, id, "today")
	expectedReview := phase != "legacy-recover"
	if err != nil || result.Result.Error != nil || result.Result.LegacyAccessReviewRequired != expectedReview {
		t.Fatal("incorrect legacy review state", err, result)
	}
	expectedReloads := 1
	if phase == "legacy-verify" {
		expectedReloads = 0
	}
	if reloads.Load() != int64(expectedReloads) {
		t.Fatalf("reloads=%d, expected %d", reloads.Load(), expectedReloads)
	}
	fmt.Printf("LEGACY={\"phase\":%q,\"reloads\":%d,\"reviewRequired\":%t}\n", phase, reloads.Load(), result.Result.LegacyAccessReviewRequired)
}

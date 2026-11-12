package service

import (
	"context"
	"github.com/istoreos/quickstart/backend/models"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMultiLANTrafficHistoryEpochAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traffic.json")
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(path), nil)
	now := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	module.now = func() time.Time { return now }
	id := "mac:02:00:00:00:00:55"
	record := func(network, epoch string, up int64, state string) {
		t.Helper()
		now = now.Add(6 * time.Minute)
		ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: network})
		response := insightTraffic(id, up, 0)
		item := response.Result.Items[0]
		item.NetworkID = network
		item.CounterEpoch = epoch
		item.State = state
		if err := module.Record(ctx, response); err != nil {
			t.Fatal(err)
		}
	}
	record("lan", "a", 100, "ready")
	record("office", "b", 500, "ready")
	record("lan", "a", 160, "ready")
	record("office", "b", 800, "ready")
	record("office", "c", 2000, "ready") // Higher counter in a new runtime is still a baseline.
	record("office", "c", 2050, "ready")
	record("office", "c", 9000, "stale")
	restarted := NewTrafficInsightsModule(newTrafficInsightsFileStore(path), nil)
	restarted.now = module.now
	for _, tc := range []struct {
		network string
		want    int64
	}{{"lan", 60}, {"office", 350}} {
		response, err := restarted.Get(withLANScope(context.Background(), &models.LANScope{NetworkID: tc.network}), id, "today")
		if err != nil || response.Result.UploadBytes != tc.want {
			t.Fatalf("%s: %+v %v", tc.network, response, err)
		}
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("default path changed:", err)
	}
	if _, err := os.Stat(lanStatePath(path, "office")); err != nil {
		t.Fatal("scoped path missing:", err)
	}
}

func TestMultiLANQuotaAccessOwnershipSurvivesManualPause(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte("config zone 'shared'\n option name 'lan'\n list network 'office'\nconfig zone 'wan'\n option name 'wan'\n list network 'wan'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "office", Device: "br-office", FirewallZones: []string{"lan"}})
	quota := context.WithValue(ctx, accessOwnerKey{}, "quota")
	mac := "02:00:00:00:00:55"
	for _, tc := range []struct {
		ctx   context.Context
		allow bool
	}{{quota, false}, {ctx, false}, {quota, true}} {
		if err := writeDeviceAccessPolicyAt(tc.ctx, dir, mac, tc.allow); err != nil {
			t.Fatal(err)
		}
	}
	allowed, err := readDeviceAccessPolicyAt(ctx, dir, mac)
	if err != nil || allowed {
		t.Fatalf("quota removal cleared manual pause: allowed=%v err=%v", allowed, err)
	}
	if err := writeDeviceAccessPolicyAt(ctx, dir, mac, true); err != nil {
		t.Fatal(err)
	}
	allowed, err = readDeviceAccessPolicyAt(ctx, dir, mac)
	if err != nil || !allowed {
		t.Fatal("manual pause could not be released", err)
	}
}

func TestMultiLANQuotaCriticalUsageSavedBeforeBlocking(t *testing.T) {
	policyStore := availablePolicyStore()
	policy := newDevicePolicyModuleForTest(policyStore)
	path := filepath.Join(t.TempDir(), "traffic.json")
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(path), policy)
	now := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	module.now = func() time.Time { return now }
	id := policyStore.policy.DeviceID
	if _, err := module.SetQuota(context.Background(), &models.TrafficQuotaRequest{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}); err != nil {
		t.Fatal(err)
	}
	if err := module.Record(context.Background(), insightTraffic(id, 0, 0)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := module.Record(context.Background(), insightTraffic(id, 120, 0)); err != nil {
		t.Fatal(err)
	}
	document, _, err := newTrafficInsightsFileStore(path).Load()
	if err != nil || !document.QuotaRuntime[id].Blocked || document.Daily[id][0].UploadBytes != 120 {
		t.Fatalf("critical block was not durable: %+v %v", document, err)
	}
	restarted := NewTrafficInsightsModule(newTrafficInsightsFileStore(path), policy)
	restarted.now = module.now
	sample := insightTraffic(id, 10, 0)
	sample.Result.Items[0].CounterEpoch = "new-engine"
	if err := restarted.Record(context.Background(), sample); err != nil {
		t.Fatal(err)
	}
	response, err := restarted.Get(context.Background(), id, "today")
	if err != nil || response.Result.Quota.UsedBytes != 120 || policyStore.policy.Access.NetworkAccess {
		t.Fatal("engine restart refreshed quota", err)
	}
}

func TestMultiLANBackgroundResumesSavedQuotasWithFreshBinding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traffic.json")
	document := emptyTrafficInsightsDocument()
	document.Quotas["mac:a"] = &models.TrafficQuota{DeviceID: "mac:a", Enabled: true, Period: "daily", LimitBytes: 100, Action: "notify"}
	if _, err := newTrafficInsightsFileStore(lanStatePath(path, "office")).Save(&document); err != nil {
		t.Fatal(err)
	}
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(path), nil)
	resolver := testLANScopeModule()
	var devices []string
	module.collector = func(ctx context.Context) (*models.DeviceTrafficResponse, error) {
		if selectedLAN(ctx) != "office" {
			t.Fatalf("background targeted %s", selectedLAN(ctx))
		}
		devices = append(devices, selectedLANScope(ctx).Device)
		return insightTraffic("mac:a", 100, 0), nil
	}
	module.AttachLANScopes(resolver)
	module.collectAllQuotaSamples()
	resolver.reader = fakeLANScopeReader{scopes: []*models.LANScope{{NetworkID: "office", Device: "br-recreated", State: "up"}}}
	module.collectAllQuotaSamples()
	resolver.reader = fakeLANScopeReader{scopes: []*models.LANScope{{NetworkID: "lan", Device: "br-lan", State: "up"}}}
	module.collectAllQuotaSamples()
	if len(devices) != 2 || devices[0] != "br-office" || devices[1] != "br-recreated" {
		t.Fatalf("background binding was reused or defaulted: %v", devices)
	}
}

func TestMultiLANQuotaWarmupCountsFirstBurstAndRejectsUnusableSampling(t *testing.T) {
	for _, state := range []string{"warming_up", "partial", "stale"} {
		t.Run(state, func(t *testing.T) {
			policyStore := availablePolicyStore()
			id := policyStore.policy.DeviceID
			module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json")), newDevicePolicyModuleForTest(policyStore))
			module.collector = func(context.Context) (*models.DeviceTrafficResponse, error) {
				response := insightTraffic(id, 0, 0)
				response.Result.Health = &models.DeviceInventoryHealth{State: state}
				response.Result.Items[0].State = state
				response.Result.Items[0].CounterEpoch = "runtime"
				response.Result.Items[0].NetworkID = "office"
				return response, nil
			}
			ctx := inventoryLANContext("office")
			result, err := module.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 100, Action: "notify"})
			if err != nil {
				t.Fatal(err)
			}
			if state != "warming_up" {
				if result.Result.Error == nil || result.Result.Error.Code != "traffic_sampling_unavailable" {
					t.Fatalf("unusable sampling accepted: %+v", result)
				}
				child := module.forContext(ctx)
				if child.loaded && len(child.document.Quotas) != 0 {
					t.Fatal("unusable quota was saved")
				}
				return
			}
			if result.Result.Error != nil {
				t.Fatal(result.Result.Error)
			}
			response := insightTraffic(id, 120, 0)
			response.Result.Items[0].CounterEpoch = "runtime"
			response.Result.Items[0].NetworkID = "office"
			if err := module.Record(ctx, response); err != nil {
				t.Fatal(err)
			}
			status, err := module.Get(ctx, id, "today")
			if err != nil || status.Result.Quota.UsedBytes != 120 || !status.Result.Quota.Exceeded {
				t.Fatalf("first burst discarded: %+v %v", status, err)
			}
		})
	}
}

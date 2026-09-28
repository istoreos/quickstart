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

func insightTraffic(deviceID string, up, down int64) *models.DeviceTrafficResponse {
	return &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{Items: []*models.DeviceTrafficItem{{DeviceID: deviceID, UploadBytes: up, DownloadBytes: down, State: "ready"}}}}
}

func TestTrafficInsightsPersistsTrendsAndThrottlesWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traffic.json")
	store := newTrafficInsightsFileStore(path)
	writes := 0
	store.persist = func(path string, raw []byte) error { writes++; return persistClassificationOverrides(path, raw) }
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60))
	module := NewTrafficInsightsModule(store, nil)
	module.now = func() time.Time { return now }
	if err := module.Record(context.Background(), insightTraffic("mac:a", 100, 200)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if err := module.Record(context.Background(), insightTraffic("mac:a", 160, 280)); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatalf("writes=%d, want first changed bucket write", writes)
	}
	now = now.Add(time.Minute)
	if err := module.Record(context.Background(), insightTraffic("mac:a", 200, 300)); err != nil {
		t.Fatal(err)
	}
	if writes != 1 {
		t.Fatalf("write throttle failed: %d", writes)
	}
	now = now.Add(5 * time.Minute)
	if err := module.Record(context.Background(), insightTraffic("mac:a", 250, 350)); err != nil {
		t.Fatal(err)
	}
	if writes != 2 {
		t.Fatalf("throttled persistence missing: %d", writes)
	}

	restarted := NewTrafficInsightsModule(newTrafficInsightsFileStore(path), nil)
	restarted.now = module.now
	response, err := restarted.Get(context.Background(), "mac:a", "today")
	if err != nil || response.Result.UploadBytes != 150 || response.Result.DownloadBytes != 150 || len(response.Result.Buckets) != 1 {
		t.Fatalf("restart response=%#v err=%v", response, err)
	}
	if response.Result.Timezone != "CST" || response.Result.StorageBytes <= 0 {
		t.Fatalf("metadata=%#v", response.Result)
	}
}

func TestTrafficInsightsCounterResetAndIdentityIsolation(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json")), nil)
	module.now = func() time.Time { return now }
	for _, sample := range []struct {
		id       string
		up, down int64
	}{{"mac:a", 1000, 1000}, {"mac:a", 100, 100}, {"mac:a", 150, 175}, {"mac:b", 500, 700}, {"mac:b", 550, 720}} {
		now = now.Add(6 * time.Minute)
		if err := module.Record(context.Background(), insightTraffic(sample.id, sample.up, sample.down)); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := module.Get(context.Background(), "mac:a", "today")
	b, _ := module.Get(context.Background(), "mac:b", "today")
	if a.Result.UploadBytes != 50 || a.Result.DownloadBytes != 75 || b.Result.UploadBytes != 50 || b.Result.DownloadBytes != 20 {
		t.Fatalf("reset or identity attribution failed: a=%#v b=%#v", a.Result, b.Result)
	}
}

func TestTrafficQuotaPeriodBoundariesAndBlockRestore(t *testing.T) {
	policyStore := availablePolicyStore()
	policy := newDevicePolicyModuleForTest(policyStore)
	now := time.Date(2026, 9, 28, 23, 55, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json")), policy)
	module.now = func() time.Time { return now }
	deviceID := policyStore.policy.DeviceID
	if _, err := module.SetQuota(context.Background(), &models.TrafficQuotaRequest{DeviceID: deviceID, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}); err != nil {
		t.Fatal(err)
	}
	if err := module.Record(context.Background(), insightTraffic(deviceID, 0, 0)); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if err := module.Record(context.Background(), insightTraffic(deviceID, 60, 50)); err != nil {
		t.Fatal(err)
	}
	if policyStore.policy.Access.NetworkAccess || policyStore.applyCalls != 1 {
		t.Fatalf("quota did not block: %#v calls=%d", policyStore.policy.Access, policyStore.applyCalls)
	}
	now = now.Add(10 * time.Minute) // next local day
	if err := module.Record(context.Background(), insightTraffic(deviceID, 70, 60)); err != nil {
		t.Fatal(err)
	}
	if !policyStore.policy.Access.NetworkAccess || policyStore.applyCalls != 2 {
		t.Fatalf("quota reset did not restore: %#v calls=%d", policyStore.policy.Access, policyStore.applyCalls)
	}
	status := module.quotaStatus(deviceID, now)
	if status.Exceeded || status.UsedBytes != 20 {
		t.Fatalf("new period status=%#v", status)
	}
}

func TestTrafficQuotaNotifyNeverBlocksAccess(t *testing.T) {
	policyStore := availablePolicyStore()
	policy := newDevicePolicyModuleForTest(policyStore)
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json")), policy)
	module.now = func() time.Time { return now }
	deviceID := policyStore.policy.DeviceID
	if _, err := module.SetQuota(context.Background(), &models.TrafficQuotaRequest{DeviceID: deviceID, Enabled: true, Period: "daily", LimitBytes: 100, Action: "notify"}); err != nil {
		t.Fatal(err)
	}
	_ = module.Record(context.Background(), insightTraffic(deviceID, 0, 0))
	now = now.Add(time.Minute)
	_ = module.Record(context.Background(), insightTraffic(deviceID, 60, 50))
	status := module.quotaStatus(deviceID, now)
	if !status.Exceeded || !policyStore.policy.Access.NetworkAccess || policyStore.applyCalls != 0 {
		t.Fatalf("notify quota must report without blocking: status=%#v access=%v calls=%d", status, policyStore.policy.Access.NetworkAccess, policyStore.applyCalls)
	}
}

func TestTrafficQuotaEditRestoresBlockImmediately(t *testing.T) {
	policyStore := availablePolicyStore()
	policy := newDevicePolicyModuleForTest(policyStore)
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json")), policy)
	module.now = func() time.Time { return now }
	deviceID := policyStore.policy.DeviceID
	if _, err := module.SetQuota(context.Background(), &models.TrafficQuotaRequest{DeviceID: deviceID, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}); err != nil {
		t.Fatal(err)
	}
	_ = module.Record(context.Background(), insightTraffic(deviceID, 0, 0))
	now = now.Add(time.Minute)
	_ = module.Record(context.Background(), insightTraffic(deviceID, 60, 50))
	if policyStore.policy.Access.NetworkAccess {
		t.Fatal("quota did not block")
	}
	response, err := module.SetQuota(context.Background(), &models.TrafficQuotaRequest{DeviceID: deviceID, Enabled: true, Period: "daily", LimitBytes: 1000, Action: "notify"})
	if err != nil || response.Result.Error != nil || !policyStore.policy.Access.NetworkAccess || module.document.QuotaRuntime[deviceID].Blocked {
		t.Fatalf("quota edit did not restore access: response=%#v err=%v runtime=%#v", response, err, module.document.QuotaRuntime[deviceID])
	}
}

func TestTrafficQuotaReassertsBlockAfterAnotherPolicyAllowsAccess(t *testing.T) {
	t.Parallel()

	policyStore := availablePolicyStore()
	policy := newDevicePolicyModuleForTest(policyStore)
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	module := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json")), policy)
	module.now = func() time.Time { return now }
	deviceID := policyStore.policy.DeviceID
	_, _ = module.SetQuota(context.Background(), &models.TrafficQuotaRequest{DeviceID: deviceID, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"})
	_ = module.Record(context.Background(), insightTraffic(deviceID, 0, 0))
	now = now.Add(time.Minute)
	_ = module.Record(context.Background(), insightTraffic(deviceID, 60, 50))
	policyStore.policy.Access.NetworkAccess = true
	now = now.Add(time.Minute)
	_ = module.Record(context.Background(), insightTraffic(deviceID, 70, 60))
	if policyStore.policy.Access.NetworkAccess || policyStore.applyCalls != 2 || !module.document.QuotaRuntime[deviceID].RestoreAccess {
		t.Fatalf("quota was bypassed: access=%v calls=%d runtime=%#v", policyStore.policy.Access.NetworkAccess, policyStore.applyCalls, module.document.QuotaRuntime[deviceID])
	}
}

func TestTrafficInsightsTwentyDevicesThirtyDaysStayWithinBudget(t *testing.T) {
	document := emptyTrafficInsightsDocument()
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	for device := 0; device < 20; device++ {
		id := fmt.Sprintf("mac:%02d", device)
		for day := 0; day < 30; day++ {
			addTrafficBucket(document.Daily, id, start.AddDate(0, 0, day), int64(1000+day), int64(2000+day))
			for hour := 0; hour < 24; hour++ {
				addTrafficBucket(document.Hourly, id, start.AddDate(0, 0, 28).Add(time.Duration(day%2*24+hour)*time.Hour), 10, 20)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "traffic.json")
	store := newTrafficInsightsFileStore(path)
	started := time.Now()
	size, err := store.Save(&document)
	if err != nil {
		t.Fatal(err)
	}
	if size > trafficInsightsBudget || time.Since(started) > 250*time.Millisecond {
		t.Fatalf("size=%d duration=%s", size, time.Since(started))
	}
	if info, err := os.Stat(path); err != nil || info.Size() != size {
		t.Fatalf("stat=%v err=%v size=%d", info, err, size)
	}
	loaded, _, err := store.Load()
	if err != nil || len(loaded.Daily) != 20 {
		t.Fatalf("load failed: devices=%d err=%v", len(loaded.Daily), err)
	}
}

func TestTrafficInsightsTwentyDevicesTwentyFourHoursStayWithinRuntimeBudget(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traffic.json")
	store := newTrafficInsightsFileStore(path)
	writes := 0
	store.persist = func(path string, raw []byte) error {
		writes++
		return persistClassificationOverrides(path, raw)
	}
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	module := NewTrafficInsightsModule(store, nil)
	module.now = func() time.Time { return now }
	response := &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{}}
	for device := 0; device < 20; device++ {
		response.Result.Items = append(response.Result.Items, &models.DeviceTrafficItem{DeviceID: fmt.Sprintf("mac:%02d", device), State: "ready"})
	}
	started := time.Now()
	for minute := 0; minute < 24*60; minute++ {
		now = now.Add(time.Minute)
		for _, item := range response.Result.Items {
			item.UploadBytes += 1024
			item.DownloadBytes += 2048
		}
		if err := module.Record(context.Background(), response); err != nil {
			t.Fatal(err)
		}
	}
	if writes > 24*12+1 {
		t.Fatalf("flash write budget exceeded: writes=%d", writes)
	}
	if module.storageSize > trafficInsightsBudget || len(module.document.Hourly) != 20 || time.Since(started) > 5*time.Second {
		t.Fatalf("runtime budget exceeded: bytes=%d devices=%d duration=%s", module.storageSize, len(module.document.Hourly), time.Since(started))
	}
}

func TestTrafficInsightsBaselineIdentityChurnIsBounded(t *testing.T) {
	now := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	store := newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json"))
	store.persist = func(string, []byte) error { return nil }
	module := NewTrafficInsightsModule(store, nil)
	module.now = func() time.Time { return now }
	for index := 0; index < trafficInsightsDeviceLimit+100; index++ {
		now = now.Add(time.Millisecond)
		if err := module.Record(context.Background(), insightTraffic(fmt.Sprintf("mac:%05d", index), 100, 200)); err != nil {
			t.Fatal(err)
		}
	}
	if len(module.document.Baselines) != trafficInsightsDeviceLimit || len(module.document.BaselineSeen) != trafficInsightsDeviceLimit {
		t.Fatalf("unbounded baselines: values=%d seen=%d", len(module.document.Baselines), len(module.document.BaselineSeen))
	}
}

func TestTrafficInsightsBackgroundCollectorRunsOnlyForEnabledQuota(t *testing.T) {
	t.Parallel()

	store := newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "traffic.json"))
	store.persist = func(string, []byte) error { return nil }
	module := NewTrafficInsightsModule(store, nil)
	module.loaded = true
	module.document = emptyTrafficInsightsDocument()
	calls := 0
	module.collector = func(context.Context) (*models.DeviceTrafficResponse, error) {
		calls++
		return insightTraffic("mac:a", int64(100*calls), int64(200*calls)), nil
	}
	module.collectQuotaSample()
	if calls != 0 {
		t.Fatalf("collector ran without an enabled quota: %d", calls)
	}
	module.document.Quotas["mac:a"] = &models.TrafficQuota{DeviceID: "mac:a", Enabled: true, Period: "daily", LimitBytes: 1024, Action: "notify"}
	module.collectQuotaSample()
	module.collectQuotaSample()
	if calls != 2 || len(module.document.Hourly["mac:a"]) != 1 {
		t.Fatalf("quota collector calls=%d hourly=%#v", calls, module.document.Hourly["mac:a"])
	}
}

func TestTrafficQuotaPeriodsUseRouterTimezone(t *testing.T) {
	location := time.FixedZone("router", -5*60*60)
	now := time.Date(2026, 9, 27, 23, 30, 0, 0, location) // Sunday
	daily, dailyNext := trafficQuotaPeriod(now, "daily")
	weekly, weeklyNext := trafficQuotaPeriod(now, "weekly")
	monthly, monthlyNext := trafficQuotaPeriod(now, "monthly")
	if daily.Hour() != 0 || dailyNext.Day() != 28 || weekly.Weekday() != time.Monday || weeklyNext.Sub(weekly) != 7*24*time.Hour || monthly.Day() != 1 || monthlyNext.Month() != time.October {
		t.Fatalf("periods daily=%v/%v weekly=%v/%v monthly=%v/%v", daily, dailyNext, weekly, weeklyNext, monthly, monthlyNext)
	}
}

package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	trafficInsightsWriteInterval = 5 * time.Minute
	trafficInsightsCollectEvery  = time.Minute
	trafficInsightsQuotaLimit    = 512
	trafficInsightsActionBackoff = 5 * time.Minute
)

type TrafficInsightsModule struct {
	mu          sync.Mutex
	store       *trafficInsightsFileStore
	policy      *DevicePolicyModule
	now         func() time.Time
	loaded      bool
	document    trafficInsightsDocument
	storageSize int64
	lastWrite   time.Time
	dirty       bool
	audit       *NetworkAuditModule
	collector   func(context.Context) (*models.DeviceTrafficResponse, error)
	collectOnce sync.Once
	collectWake chan struct{}
}

func NewTrafficInsightsModule(store *trafficInsightsFileStore, policy *DevicePolicyModule) *TrafficInsightsModule {
	return &TrafficInsightsModule{store: store, policy: policy, now: time.Now}
}

func NewDefaultTrafficInsightsModule(policy *DevicePolicyModule) *TrafficInsightsModule {
	return NewTrafficInsightsModule(newTrafficInsightsFileStore(defaultTrafficInsightsPath), policy)
}

// AttachCollector keeps quota enforcement alive without turning ordinary
// traffic history into an always-on sampler. The single worker samples once a
// minute only while at least one quota is enabled.
func (module *TrafficInsightsModule) AttachCollector(collector func(context.Context) (*models.DeviceTrafficResponse, error)) {
	if module == nil || collector == nil {
		return
	}
	module.mu.Lock()
	module.collector = collector
	if module.collectWake == nil {
		module.collectWake = make(chan struct{}, 1)
	}
	wake := module.collectWake
	module.mu.Unlock()
	module.collectOnce.Do(func() { go module.collectorWorker(wake) })
	module.wakeCollector()
}

func (module *TrafficInsightsModule) collectorWorker(wake <-chan struct{}) {
	ticker := time.NewTicker(trafficInsightsCollectEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-wake:
		}
		module.collectQuotaSample()
	}
}

func (module *TrafficInsightsModule) collectQuotaSample() {
	module.mu.Lock()
	if err := module.load(); err != nil {
		module.mu.Unlock()
		return
	}
	enabled := false
	for _, quota := range module.document.Quotas {
		if quota != nil && quota.Enabled {
			enabled = true
			break
		}
	}
	collector := module.collector
	module.mu.Unlock()
	if !enabled || collector == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	response, err := collector(ctx)
	if err == nil {
		_ = module.Record(ctx, response)
	}
	cancel()
}

func (module *TrafficInsightsModule) wakeCollector() {
	module.mu.Lock()
	wake := module.collectWake
	module.mu.Unlock()
	if wake != nil {
		select {
		case wake <- struct{}{}:
		default:
		}
	}
}

func (module *TrafficInsightsModule) Record(ctx context.Context, response *models.DeviceTrafficResponse) error {
	if module == nil || module.store == nil || response == nil || response.Result == nil {
		return nil
	}
	module.mu.Lock()
	defer module.mu.Unlock()
	if err := module.load(); err != nil {
		return err
	}
	now := module.now()
	hourChanged := false
	for _, item := range response.Result.Items {
		if item == nil || item.DeviceID == "" || item.State != "ready" {
			continue
		}
		baseline, exists := module.document.Baselines[item.DeviceID]
		uploadDelta, downloadDelta := int64(0), int64(0)
		if exists && item.UploadBytes >= baseline.UploadBytes {
			uploadDelta = item.UploadBytes - baseline.UploadBytes
		}
		if exists && item.DownloadBytes >= baseline.DownloadBytes {
			downloadDelta = item.DownloadBytes - baseline.DownloadBytes
		}
		module.document.Baselines[item.DeviceID] = trafficCounterBaseline{UploadBytes: item.UploadBytes, DownloadBytes: item.DownloadBytes}
		module.document.BaselineSeen[item.DeviceID] = now.UTC().Format(time.RFC3339Nano)
		if uploadDelta == 0 && downloadDelta == 0 {
			continue
		}
		if addTrafficBucket(module.document.Hourly, item.DeviceID, now.Truncate(time.Hour), uploadDelta, downloadDelta) {
			hourChanged = true
		}
		addTrafficBucket(module.document.Daily, item.DeviceID, localDayStart(now), uploadDelta, downloadDelta)
		addTrafficBucket(module.document.Monthly, item.DeviceID, localMonthStart(now), uploadDelta, downloadDelta)
		module.dirty = true
	}
	module.prune(now)
	module.pruneBaselines()
	module.reconcileQuotas(ctx, now)
	if module.dirty && (module.lastWrite.IsZero() || now.Sub(module.lastWrite) >= trafficInsightsWriteInterval || hourChanged) {
		return module.persist(now)
	}
	return nil
}

func (module *TrafficInsightsModule) Get(_ context.Context, deviceID, selectedRange string) (*models.TrafficInsightsResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	if err := module.load(); err != nil {
		return nil, err
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return trafficInsightsFailure("validation_failed", "deviceId is required"), nil
	}
	if selectedRange == "" {
		selectedRange = "today"
	}
	now := module.now()
	var source []*models.TrafficInsightBucket
	var cutoff time.Time
	switch selectedRange {
	case "today":
		source, cutoff = module.document.Hourly[deviceID], localDayStart(now)
	case "week":
		source, cutoff = module.document.Daily[deviceID], localWeekStart(now)
	case "month":
		source, cutoff = module.document.Daily[deviceID], localMonthStart(now)
	default:
		return trafficInsightsFailure("validation_failed", "range must be today, week or month"), nil
	}
	buckets := filterTrafficBuckets(source, cutoff)
	var upload, download int64
	for _, bucket := range buckets {
		upload += bucket.UploadBytes
		download += bucket.DownloadBytes
	}
	local, bandix := trafficInsightsCapabilities()
	return &models.TrafficInsightsResponse{Result: &models.TrafficInsightsResult{
		DeviceID: deviceID, Range: selectedRange, Buckets: buckets, UploadBytes: upload, DownloadBytes: download,
		Quota: module.quotaStatus(deviceID, now), Local: local, Bandix: bandix, Timezone: now.Location().String(),
		StorageBytes: module.storageSize, StorageBudget: trafficInsightsBudget,
		WriteIntervalS: int64(trafficInsightsWriteInterval.Seconds()),
	}}, nil
}

func (module *TrafficInsightsModule) SetQuota(ctx context.Context, request *models.TrafficQuotaRequest) (*models.TrafficInsightsResponse, error) {
	module.mu.Lock()
	defer func() {
		module.mu.Unlock()
		module.wakeCollector()
	}()
	if err := module.load(); err != nil {
		return nil, err
	}
	if err := validateTrafficQuotaRequest(request); err != nil {
		return trafficInsightsFailure("validation_failed", err.Error()), nil
	}
	if _, exists := module.document.Quotas[request.DeviceID]; !exists && request.Enabled && len(module.document.Quotas) >= trafficInsightsQuotaLimit {
		return trafficInsightsFailure("capacity_exceeded", "traffic quota limit reached"), nil
	}
	if !request.Enabled {
		if runtimeState := module.document.QuotaRuntime[request.DeviceID]; runtimeState.Blocked {
			if !module.applyQuotaAccess(ctx, request.DeviceID, runtimeState.RestoreAccess) {
				return trafficInsightsFailure("apply_failed", "could not restore network access before disabling quota"), nil
			}
		}
		delete(module.document.Quotas, request.DeviceID)
		delete(module.document.QuotaRuntime, request.DeviceID)
	} else {
		module.document.Quotas[request.DeviceID] = &models.TrafficQuota{DeviceID: request.DeviceID, Enabled: true, Period: request.Period, LimitBytes: request.LimitBytes, Action: request.Action}
	}
	module.dirty = true
	module.reconcileQuotas(ctx, module.now())
	if err := module.persist(module.now()); err != nil {
		return nil, err
	}
	// Do not duplicate query logic while holding the lock.
	status := module.quotaStatus(request.DeviceID, module.now())
	local, bandix := trafficInsightsCapabilities()
	return &models.TrafficInsightsResponse{Result: &models.TrafficInsightsResult{
		DeviceID: request.DeviceID, Range: "today", Buckets: []*models.TrafficInsightBucket{}, Quota: status,
		Local: local, Bandix: bandix, Timezone: module.now().Location().String(), StorageBytes: module.storageSize,
		StorageBudget: trafficInsightsBudget, WriteIntervalS: int64(trafficInsightsWriteInterval.Seconds()),
	}}, nil
}

// SetQuotaBatch validates and persists an entire group quota plan once. The
// returned snapshot lets the group coordinator restore the quota document if
// a later firewall/DHCP/service step fails.
func (module *TrafficInsightsModule) SetQuotaBatch(ctx context.Context, requests []*models.TrafficQuotaRequest) (*trafficInsightsDocument, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	if err := module.load(); err != nil {
		return nil, err
	}
	previous := cloneTrafficInsightsDocument(module.document)
	projected := len(module.document.Quotas)
	seenNew := map[string]bool{}
	for _, request := range requests {
		if err := validateTrafficQuotaRequest(request); err != nil {
			return nil, err
		}
		if request.Enabled {
			if _, exists := module.document.Quotas[request.DeviceID]; !exists && !seenNew[request.DeviceID] {
				projected++
				seenNew[request.DeviceID] = true
			}
		} else if _, exists := module.document.Quotas[request.DeviceID]; exists {
			projected--
		}
	}
	if projected > trafficInsightsQuotaLimit {
		return nil, errors.New("traffic quota limit reached")
	}
	for _, request := range requests {
		if !request.Enabled {
			if runtimeState := module.document.QuotaRuntime[request.DeviceID]; runtimeState.Blocked && !module.applyQuotaAccess(ctx, request.DeviceID, runtimeState.RestoreAccess) {
				module.document = previous
				return nil, errors.New("could not restore network access before disabling quota")
			}
			delete(module.document.Quotas, request.DeviceID)
			delete(module.document.QuotaRuntime, request.DeviceID)
			continue
		}
		module.document.Quotas[request.DeviceID] = &models.TrafficQuota{DeviceID: request.DeviceID, Enabled: true, Period: request.Period, LimitBytes: request.LimitBytes, Action: request.Action}
	}
	module.dirty = true
	module.reconcileQuotas(ctx, module.now())
	if err := module.persist(module.now()); err != nil {
		module.document = previous
		return nil, err
	}
	return &previous, nil
}

func (module *TrafficInsightsModule) RestoreQuotaBatch(snapshot trafficInsightsDocument) error {
	module.mu.Lock()
	defer module.mu.Unlock()
	copyValue := cloneTrafficInsightsDocument(snapshot)
	if _, err := module.store.Save(&copyValue); err != nil {
		return err
	}
	module.document, module.loaded, module.dirty = copyValue, true, false
	return nil
}

func (module *TrafficInsightsModule) load() error {
	if module.loaded {
		return nil
	}
	document, size, err := module.store.Load()
	if err != nil {
		return err
	}
	module.document, module.storageSize, module.loaded = document, size, true
	return nil
}

func (module *TrafficInsightsModule) persist(now time.Time) error {
	size, err := module.store.Save(&module.document)
	if err != nil {
		return err
	}
	module.storageSize, module.lastWrite, module.dirty = size, now, false
	return nil
}

func (module *TrafficInsightsModule) prune(now time.Time) {
	pruneTrafficSeries(module.document.Hourly, now.Add(-48*time.Hour))
	pruneTrafficSeries(module.document.Daily, localDayStart(now).AddDate(0, 0, -35))
	pruneTrafficSeries(module.document.Monthly, localMonthStart(now).AddDate(0, -13, 0))
}

func (module *TrafficInsightsModule) pruneBaselines() {
	if len(module.document.Baselines) <= trafficInsightsDeviceLimit {
		return
	}
	type baselineAge struct{ id, seen string }
	ages := make([]baselineAge, 0, len(module.document.Baselines))
	for id := range module.document.Baselines {
		ages = append(ages, baselineAge{id: id, seen: module.document.BaselineSeen[id]})
	}
	sort.Slice(ages, func(left, right int) bool {
		if ages[left].seen != ages[right].seen {
			return ages[left].seen < ages[right].seen
		}
		return ages[left].id < ages[right].id
	})
	for index := 0; index < len(ages)-trafficInsightsDeviceLimit; index++ {
		delete(module.document.Baselines, ages[index].id)
		delete(module.document.BaselineSeen, ages[index].id)
	}
	module.dirty = true
}

func (module *TrafficInsightsModule) quotaStatus(deviceID string, now time.Time) *models.TrafficQuotaStatus {
	quota := module.document.Quotas[deviceID]
	if quota == nil {
		return &models.TrafficQuotaStatus{}
	}
	start, next := trafficQuotaPeriod(now, quota.Period)
	var used int64
	for _, bucket := range module.document.Daily[deviceID] {
		parsed, err := time.Parse(time.RFC3339, bucket.Start)
		if err == nil && !parsed.Before(start) {
			used += bucket.UploadBytes + bucket.DownloadBytes
		}
	}
	return &models.TrafficQuotaStatus{Quota: quota, UsedBytes: used, Exceeded: used >= quota.LimitBytes, PeriodStart: start.Format(time.RFC3339), NextResetAt: next.Format(time.RFC3339)}
}

func (module *TrafficInsightsModule) reconcileQuotas(ctx context.Context, now time.Time) {
	for deviceID, quota := range module.document.Quotas {
		if quota == nil || !quota.Enabled {
			continue
		}
		status := module.quotaStatus(deviceID, now)
		runtimeState := module.document.QuotaRuntime[deviceID]
		if runtimeState.PeriodStart != status.PeriodStart {
			if runtimeState.Blocked {
				if !module.applyQuotaAccess(ctx, deviceID, runtimeState.RestoreAccess) {
					continue
				}
			}
			runtimeState = trafficQuotaRuntime{PeriodStart: status.PeriodStart}
			module.document.QuotaRuntime[deviceID] = runtimeState
			module.dirty = true
		}
		if runtimeState.Blocked && (!status.Exceeded || quota.Action != "block") {
			if !module.applyQuotaAccess(ctx, deviceID, runtimeState.RestoreAccess) {
				continue
			}
			runtimeState.Blocked, runtimeState.RestoreAccess = false, false
			module.document.QuotaRuntime[deviceID] = runtimeState
			module.dirty = true
		}
		if runtimeState.Blocked && status.Exceeded && quota.Action == "block" && module.currentQuotaAccess(ctx, deviceID) {
			// Another policy source may have restored access after the quota
			// blocked it. Reassert the quota without changing the original
			// access value that must be restored at the next reset.
			module.applyQuotaAccess(ctx, deviceID, false)
		}
		if status.Exceeded && quota.Action == "block" && !runtimeState.Blocked {
			currentAccess := true
			currentAccess = module.currentQuotaAccess(ctx, deviceID)
			if module.applyQuotaAccess(ctx, deviceID, false) {
				runtimeState.Blocked, runtimeState.RestoreAccess = true, currentAccess
				module.document.QuotaRuntime[deviceID] = runtimeState
				module.dirty = true
			}
		}
		if status.Exceeded && !runtimeState.Notified {
			if module.audit != nil {
				module.audit.Record(deviceID, "traffic_quota", "quota_exceeded", "exceeded", quota.Period)
			}
			runtimeState.Notified = true
			module.document.QuotaRuntime[deviceID] = runtimeState
			module.dirty = true
		}
	}
}

func (module *TrafficInsightsModule) currentQuotaAccess(ctx context.Context, deviceID string) bool {
	if module.policy == nil {
		return true
	}
	current, err := module.policy.Get(ctx, deviceID)
	if err == nil && current != nil && current.Result != nil && current.Result.Policy != nil && current.Result.Policy.Access != nil {
		return current.Result.Policy.Access.NetworkAccess
	}
	return true
}

func (module *TrafficInsightsModule) applyQuotaAccess(ctx context.Context, deviceID string, access bool) bool {
	if module.policy == nil {
		return false
	}
	response, err := module.policy.Apply(ctx, &models.DevicePolicyApplyRequest{DeviceID: deviceID, Kind: "access", Access: &models.DeviceAccessPolicy{NetworkAccess: access}})
	return err == nil && response != nil && response.Result != nil && response.Result.Error == nil
}

func addTrafficBucket(series map[string][]*models.TrafficInsightBucket, deviceID string, start time.Time, upload, download int64) bool {
	key := start.Format(time.RFC3339)
	buckets := series[deviceID]
	for _, bucket := range buckets {
		if bucket.Start == key {
			bucket.UploadBytes += upload
			bucket.DownloadBytes += download
			return false
		}
	}
	series[deviceID] = append(buckets, &models.TrafficInsightBucket{Start: key, UploadBytes: upload, DownloadBytes: download})
	sort.Slice(series[deviceID], func(i, j int) bool { return series[deviceID][i].Start < series[deviceID][j].Start })
	return true
}

func filterTrafficBuckets(source []*models.TrafficInsightBucket, cutoff time.Time) []*models.TrafficInsightBucket {
	result := make([]*models.TrafficInsightBucket, 0, len(source))
	for _, bucket := range source {
		parsed, err := time.Parse(time.RFC3339, bucket.Start)
		if err == nil && !parsed.Before(cutoff) {
			copyValue := *bucket
			result = append(result, &copyValue)
		}
	}
	return sortedTrafficBuckets(result)
}

func pruneTrafficSeries(series map[string][]*models.TrafficInsightBucket, cutoff time.Time) {
	for deviceID, buckets := range series {
		kept := buckets[:0]
		for _, bucket := range buckets {
			parsed, err := time.Parse(time.RFC3339, bucket.Start)
			if err == nil && !parsed.Before(cutoff) {
				kept = append(kept, bucket)
			}
		}
		if len(kept) == 0 {
			delete(series, deviceID)
		} else {
			series[deviceID] = kept
		}
	}
}

func localDayStart(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}
func localMonthStart(now time.Time) time.Time {
	y, m, _ := now.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, now.Location())
}
func localWeekStart(now time.Time) time.Time {
	day := localDayStart(now)
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset)
}

func trafficQuotaPeriod(now time.Time, period string) (time.Time, time.Time) {
	switch period {
	case "daily":
		start := localDayStart(now)
		return start, start.AddDate(0, 0, 1)
	case "weekly":
		start := localWeekStart(now)
		return start, start.AddDate(0, 0, 7)
	default:
		start := localMonthStart(now)
		return start, start.AddDate(0, 1, 0)
	}
}

func validateTrafficQuotaRequest(request *models.TrafficQuotaRequest) error {
	if request == nil || !validDeviceIdentityKey(request.DeviceID) {
		return errors.New("deviceId is required")
	}
	if !request.Enabled {
		return nil
	}
	if request.Period != "daily" && request.Period != "weekly" && request.Period != "monthly" {
		return errors.New("period must be daily, weekly or monthly")
	}
	if request.LimitBytes <= 0 {
		return errors.New("limitBytes must be greater than zero")
	}
	if request.Action != "notify" && request.Action != "block" {
		return errors.New("action must be notify or block")
	}
	return nil
}

func trafficInsightsCapabilities() (*models.TrafficInsightsCapability, *models.TrafficInsightsCapability) {
	local := &models.TrafficInsightsCapability{State: "available", Adapter: "local", Limitations: []string{"hourly_precision", "reboot_write_window"}}
	bandix := &models.TrafficInsightsCapability{State: "not_installed", Adapter: "bandix", Reason: "bandix_not_installed", Limitations: []string{"ebpf_kernel_support", "single_interface", "hardware_offload"}}
	if _, err := os.Stat("/usr/bin/bandix"); err == nil {
		bandix.State, bandix.Reason = "available", ""
		if raw, readErr := os.ReadFile("/etc/config/firewall"); readErr == nil && (strings.Contains(string(raw), "option flow_offloading '1'") || strings.Contains(string(raw), "option flow_offloading_hw '1'")) {
			bandix.State, bandix.Reason = models.CapabilityAvailable, "hardware_offload_enabled"
		}
	}
	return local, bandix
}

func trafficInsightsFailure(code, message string) *models.TrafficInsightsResponse {
	return &models.TrafficInsightsResponse{Result: &models.TrafficInsightsResult{Buckets: []*models.TrafficInsightBucket{}, Error: &models.DevicePolicyError{Code: code, Message: message}}}
}

func (module *TrafficInsightsModule) String() string {
	return fmt.Sprintf("TrafficInsights(%d bytes)", module.storageSize)
}

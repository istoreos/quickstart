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
	trafficInsightsCollectEvery  = 15 * time.Second
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
	network     string
	root        *TrafficInsightsModule
	children    map[string]*TrafficInsightsModule
	resolver    *LANScopeModule
}

func NewTrafficInsightsModule(store *trafficInsightsFileStore, policy *DevicePolicyModule) *TrafficInsightsModule {
	return &TrafficInsightsModule{store: store, policy: policy, now: time.Now}
}

func NewDefaultTrafficInsightsModule(policy *DevicePolicyModule) *TrafficInsightsModule {
	return NewTrafficInsightsModule(newTrafficInsightsFileStore(defaultTrafficInsightsPath), policy)
}

// AttachCollector keeps quota enforcement alive without turning ordinary
// traffic history into an always-on sampler. The single worker samples every
// 15 seconds only while at least one quota is enabled. This also keeps the
// legacy sampler alive beyond its 30-second inactivity timeout.
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
		module.collectAllQuotaSamples()
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
	collector, resolver := module.collector, module.resolver
	module.mu.Unlock()
	if !enabled || collector == nil || (module.network != "" && resolver == nil) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if resolver != nil {
		scope, err := resolver.resolve(ctx, inventoryLANID(module.network), true)
		if err != nil || scope.State == "down" || scope.ReadOnlyReason != "" {
			return
		}
		ctx = context.WithValue(withLANScope(ctx, scope), lanScopeResolverKey{}, resolver)
	}
	response, err := collector(ctx)
	if err == nil {
		_ = module.Record(ctx, response)
	}
}

func (module *TrafficInsightsModule) wakeCollector() {
	if module.root != nil {
		module.root.wakeCollector()
		return
	}
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
	locked, release, err := lockNetworkConfiguration(ctx, deviceRestrictionConfigDir())
	if err != nil {
		return err
	}
	defer release()
	ctx = locked
	if scoped := module.forContext(ctx); scoped != module {
		return scoped.Record(ctx, response)
	}
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
		if item == nil || item.DeviceID == "" || (item.NetworkID != "" && item.NetworkID != inventoryLANID(module.network)) {
			continue
		}
		baseline, exists := module.document.Baselines[item.DeviceID]
		if item.State != "ready" {
			if item.State == "warming_up" && item.CounterEpoch != "" && item.UploadBytes == 0 && item.DownloadBytes == 0 && (!exists || baseline.Epoch != item.CounterEpoch) {
				module.document.Baselines[item.DeviceID] = trafficCounterBaseline{Epoch: item.CounterEpoch}
				module.document.BaselineSeen[item.DeviceID] = now.UTC().Format(time.RFC3339Nano)
			}
			continue
		}
		uploadDelta, downloadDelta := int64(0), int64(0)
		if exists && baseline.Epoch == item.CounterEpoch && item.UploadBytes >= baseline.UploadBytes {
			uploadDelta = item.UploadBytes - baseline.UploadBytes
		}
		if exists && baseline.Epoch == item.CounterEpoch && item.DownloadBytes >= baseline.DownloadBytes {
			downloadDelta = item.DownloadBytes - baseline.DownloadBytes
		}
		module.document.Baselines[item.DeviceID] = trafficCounterBaseline{UploadBytes: item.UploadBytes, DownloadBytes: item.DownloadBytes, Epoch: item.CounterEpoch}
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
	critical := false
	scope := selectedLANScope(ctx)
	if groupBatchAccumulatorFrom(ctx) == nil && (scope == nil || scope.State != "down" && scope.ReadOnlyReason == "") {
		var err error
		critical, err = module.reconcileQuotas(ctx, now)
		if err != nil {
			return err
		}
	}
	if module.dirty && (module.lastWrite.IsZero() || now.Sub(module.lastWrite) >= trafficInsightsWriteInterval || hourChanged || critical) {
		return module.persist(now)
	}
	return nil
}

func (module *TrafficInsightsModule) Get(ctx context.Context, deviceID, selectedRange string) (*models.TrafficInsightsResponse, error) {
	if scoped := module.forContext(ctx); scoped != module {
		return scoped.Get(ctx, deviceID, selectedRange)
	}
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
	review, err := module.legacyAccessReview(ctx, deviceID)
	if err != nil {
		return trafficInsightsFailure("access_review_unavailable", "暂时无法确认原有上网限制，请重试"), nil
	}
	local, bandix := trafficInsightsCapabilities()
	return &models.TrafficInsightsResponse{Result: &models.TrafficInsightsResult{
		LegacyAccessReviewRequired: review, DeviceID: deviceID, Range: selectedRange, Buckets: buckets, UploadBytes: upload, DownloadBytes: download,
		Quota: module.quotaStatus(deviceID, now), Local: local, Bandix: bandix, Timezone: now.Location().String(),
		StorageBytes: module.storageSize, StorageBudget: trafficInsightsBudget,
		WriteIntervalS: int64(trafficInsightsWriteInterval.Seconds()),
	}}, nil
}

func (module *TrafficInsightsModule) SetQuota(ctx context.Context, request *models.TrafficQuotaRequest) (*models.TrafficInsightsResponse, error) {
	locked, release, err := lockNetworkConfiguration(ctx, deviceRestrictionConfigDir())
	if err != nil {
		return nil, err
	}
	defer release()
	ctx = locked
	if scoped := module.forContext(ctx); scoped != module {
		return scoped.SetQuota(ctx, request)
	}
	if err := validateTrafficQuotaRequest(request); err != nil {
		return trafficInsightsFailure("validation_failed", err.Error()), nil
	}
	if request.Enabled && module.policy != nil {
		current, err := module.policy.Get(ctx, request.DeviceID)
		if err != nil || current == nil || current.Result == nil || current.Result.Error != nil || current.Result.Policy == nil {
			return trafficInsightsFailure("device_not_found", "设备不属于当前网络"), nil
		}
		if request.Action == "block" {
			capability := current.Result.Policy.Capabilities["access"]
			if capability == nil || capability.State != "available" {
				return trafficInsightsFailure("quota_enforcement_unavailable", "当前网络无法执行配额封禁"), nil
			}
		}
	}
	if _, err := module.SetQuotaBatch(ctx, []*models.TrafficQuotaRequest{request}); err != nil {
		if errors.Is(err, errNetworkConfigRecoveryRequired) {
			return trafficInsightsFailure("recovery_required", err.Error()), nil
		}
		if strings.Contains(err.Error(), "traffic_sampling_unavailable") || strings.Contains(err.Error(), "traffic quota limit reached") {
			code := "traffic_sampling_unavailable"
			message := "当前无法确认流量采样，请恢复采样后再设置配额"
			if strings.Contains(err.Error(), "traffic quota limit reached") {
				code = "capacity_exceeded"
				message = "traffic quota limit reached"
			}
			return trafficInsightsFailure(code, message), nil
		}
		return nil, err
	}
	return module.Get(ctx, request.DeviceID, "today")
}

// SetQuotaBatch validates and persists an entire group quota plan once. The
// returned snapshot lets the group coordinator restore the quota document if
// a later firewall/DHCP/service step fails.
func (module *TrafficInsightsModule) SetQuotaBatch(ctx context.Context, requests []*models.TrafficQuotaRequest) (*trafficInsightsDocument, error) {
	locked, release, err := lockNetworkConfiguration(ctx, deviceRestrictionConfigDir())
	if err != nil {
		return nil, err
	}
	defer release()
	ctx = locked
	if scoped := module.forContext(ctx); scoped != module {
		return scoped.SetQuotaBatch(ctx, requests)
	}
	for _, request := range requests {
		if err := validateTrafficQuotaRequest(request); err != nil {
			return nil, err
		}
	}
	for _, request := range requests {
		if request.Enabled {
			if err := module.prepareQuotaSample(ctx); err != nil {
				return nil, err
			}
			break
		}
	}
	module.mu.Lock()
	defer func() { module.mu.Unlock(); module.wakeCollector() }()
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
	var backup devicePolicyBackup
	if module.policy != nil && !groupBatchDefersReload(ctx) {
		var err error
		backup, err = module.policy.store.Backup(ctx, "access")
		if err != nil {
			return nil, err
		}
	}
	originalCtx := ctx
	ctx, localBatch := module.quotaAccessBatch(ctx)
	physicalAttempted := localBatch == nil
	fail := func(cause error) (*trafficInsightsDocument, error) {
		module.document = previous
		_, saveErr := module.store.Save(&previous)
		var accessErr error
		if backup != nil && physicalAttempted {
			accessErr = module.policy.store.Restore(originalCtx, backup)
		}
		if saveErr != nil || accessErr != nil {
			return nil, &networkRecoveryError{fmt.Errorf("quota batch recovery_required: %v; %v; %v", cause, saveErr, accessErr)}
		}
		module.dirty = false
		return nil, cause
	}
	if err := module.stageQuotaAccessBindings(ctx); err != nil {
		return fail(err)
	}
	for _, request := range requests {
		if !request.Enabled {
			if runtimeState := module.document.QuotaRuntime[request.DeviceID]; runtimeState.Blocked && !module.releaseQuotaAccess(ctx, request.DeviceID, runtimeState.RestoreAccess) {
				return fail(errors.New("could not restore network access before disabling quota"))
			}
			delete(module.document.Quotas, request.DeviceID)
			delete(module.document.QuotaRuntime, request.DeviceID)
			continue
		}
		module.document.Quotas[request.DeviceID] = &models.TrafficQuota{DeviceID: request.DeviceID, Enabled: true, Period: request.Period, LimitBytes: request.LimitBytes, Action: request.Action}
	}
	module.dirty = true
	if _, err := module.reconcileQuotas(ctx, module.now()); err != nil {
		return fail(err)
	}
	if localBatch.needsAccessCommit() {
		physicalAttempted = true
	}
	if err := commitQuotaAccessBatch(originalCtx, localBatch); err != nil {
		return fail(err)
	}
	if err := module.persist(module.now()); err != nil {
		return fail(err)
	}
	return &previous, nil
}

func (module *TrafficInsightsModule) RestoreQuotaBatch(ctx context.Context, snapshot trafficInsightsDocument) error {
	if scoped := module.forContext(ctx); scoped != module {
		return scoped.RestoreQuotaBatch(ctx, snapshot)
	}
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

// One sample updates every device before enforcement starts. Persist that
// usage once before the first action, then checkpoint runtime results together.
type quotaCheckpointKey struct{}
type quotaCheckpoint struct{ done bool }

func (module *TrafficInsightsModule) reconcileQuotas(ctx context.Context, now time.Time) (changed bool, resultErr error) {
	originalCtx := ctx
	// Notifications need no firewall dependency. Only physical access changes
	// use the same bounded compiler/reload path as group batches.
	accessWork := false
	for id, quota := range module.document.Quotas {
		state := module.document.QuotaRuntime[id]
		if quota != nil && (quota.Enabled && quota.Action == "block" || state.Blocked) {
			accessWork = true
			break
		}
	}
	if accessWork {
		deferred, batch := module.quotaAccessBatch(ctx)
		if batch != nil {
			backup, err := module.policy.store.Backup(ctx, "access")
			if err != nil {
				return false, err
			}
			previous := make(map[string]trafficQuotaRuntime, len(module.document.QuotaRuntime))
			for id, state := range module.document.QuotaRuntime {
				previous[id] = state
			}
			ctx = deferred
			defer func() {
				physicalAttempted := resultErr == nil && batch.needsAccessCommit()
				if physicalAttempted {
					resultErr = commitQuotaAccessBatch(originalCtx, batch)
				}
				if resultErr != nil {
					module.document.QuotaRuntime = previous
					var restoreErr error
					if physicalAttempted {
						restoreErr = module.policy.store.Restore(originalCtx, backup)
					}
					saveErr := module.persist(now)
					if restoreErr != nil || saveErr != nil {
						resultErr = &networkRecoveryError{fmt.Errorf("quota recovery_required: %v; %v; %v", resultErr, restoreErr, saveErr)}
					}
					changed = true
				}
			}()
		}
	}

	if accessWork {
		if err := module.stageQuotaAccessBindings(ctx); err != nil {
			return false, err
		}
	}

	ctx = context.WithValue(ctx, quotaCheckpointKey{}, &quotaCheckpoint{})
	before := make(map[string]trafficQuotaRuntime, len(module.document.QuotaRuntime))
	for id, runtime := range module.document.QuotaRuntime {
		before[id] = runtime
	}
	for deviceID, quota := range module.document.Quotas {
		if quota == nil || !quota.Enabled {
			continue
		}
		status := module.quotaStatus(deviceID, now)
		runtimeState := module.document.QuotaRuntime[deviceID]
		if runtimeState.PeriodStart != status.PeriodStart {
			if runtimeState.Blocked {
				if !module.releaseQuotaAccess(ctx, deviceID, runtimeState.RestoreAccess) {
					return true, errors.New("quota_enforcement_failed")
				}
			}
			runtimeState = trafficQuotaRuntime{PeriodStart: status.PeriodStart}
			module.document.QuotaRuntime[deviceID] = runtimeState
			module.dirty = true
		}
		if runtimeState.Blocked && (!status.Exceeded || quota.Action != "block") {
			if !module.releaseQuotaAccess(ctx, deviceID, runtimeState.RestoreAccess) {
				return true, errors.New("quota_enforcement_failed")
			}
			runtimeState.Blocked, runtimeState.RestoreAccess = false, false
			module.document.QuotaRuntime[deviceID] = runtimeState
			module.dirty = true
		}
		if runtimeState.Blocked && status.Exceeded && quota.Action == "block" && module.currentQuotaAccess(ctx, deviceID) {
			// Another policy source may have restored access after the quota
			// blocked it. Reassert the quota without changing the original
			// access value that must be restored at the next reset.
			if !module.applyQuotaAccess(ctx, deviceID, false) {
				return true, errors.New("quota_enforcement_failed")
			}
		}
		if status.Exceeded && quota.Action == "block" && !runtimeState.Blocked {
			currentAccess := true
			currentAccess = module.currentQuotaAccess(ctx, deviceID)
			if module.applyQuotaAccess(ctx, deviceID, false) {
				runtimeState.Blocked, runtimeState.RestoreAccess = true, currentAccess
				module.document.QuotaRuntime[deviceID] = runtimeState
				module.dirty = true
			} else {
				return true, errors.New("quota_enforcement_failed")
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
	if len(before) != len(module.document.QuotaRuntime) {
		return true, nil
	}
	for id, runtime := range module.document.QuotaRuntime {
		if before[id] != runtime {
			return true, nil
		}
	}
	return false, nil
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
	checkpoint, _ := ctx.Value(quotaCheckpointKey{}).(*quotaCheckpoint)
	if module.dirty && (checkpoint == nil || !checkpoint.done) {
		if err := module.persist(module.now()); err != nil {
			return false
		}
	}
	if checkpoint != nil {
		checkpoint.done = true
	}
	if module.policy == nil {
		return false
	}
	if _, system := module.policy.store.(*systemDevicePolicyStore); system {
		ctx = context.WithValue(ctx, accessOwnerKey{}, "quota")
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

// Quota rules have independent ownership. Removing a quota cannot remove a
// pause that the user or a group applied while the quota was exceeded.
func (module *TrafficInsightsModule) releaseQuotaAccess(ctx context.Context, id string, original bool) bool {
	if module.policy != nil {
		if _, system := module.policy.store.(*systemDevicePolicyStore); system {
			original = true
		}
	}
	return module.applyQuotaAccess(ctx, id, original)
}

func (module *TrafficInsightsModule) legacyAccessReview(ctx context.Context, id string) (bool, error) {
	if module.policy == nil {
		return false, nil
	}
	if _, system := module.policy.store.(*systemDevicePolicyStore); !system {
		return false, nil
	}
	return legacyAccessReviewForDevice(ctx, id)
}

func (module *TrafficInsightsModule) quotaAccessBatch(ctx context.Context) (context.Context, *groupBatchAccumulator) {
	if module.policy == nil || groupBatchDefersReload(ctx) {
		return ctx, nil
	}
	if _, system := module.policy.store.(*systemDevicePolicyStore); !system {
		return ctx, nil
	}
	batch := &groupBatchAccumulator{}
	return context.WithValue(ctx, groupBatchContextKey{}, batch), batch
}

func commitQuotaAccessBatch(ctx context.Context, batch *groupBatchAccumulator) error {
	if !batch.needsAccessCommit() {
		return nil
	}
	if err := groupBatchWriteAccess(ctx, deviceRestrictionConfigDir(), batch.restrictions); err != nil {
		return err
	}
	return deviceAccessRestrictionApply(ctx)
}

func (module *TrafficInsightsModule) stageQuotaAccessBindings(ctx context.Context) error {
	batch := groupBatchAccumulatorFrom(ctx)
	if module.policy == nil || batch == nil || batch.accessBindingsPending {
		return nil
	}
	if _, system := module.policy.store.(*systemDevicePolicyStore); !system {
		return nil
	}
	pending, err := accessBindingsNeedCommit(ctx)
	if err == nil {
		batch.accessBindingsPending = pending
	}
	return err
}

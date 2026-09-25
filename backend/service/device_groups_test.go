package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func boolPointer(value bool) *bool { return &value }

func TestDeviceGroupPolicyPrecedenceAndExplanation(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) // Monday
	state := deviceGroupState{
		GlobalPolicy: &models.GroupPolicy{Access: boolPointer(false)},
		Groups: []*models.DeviceGroup{
			{ID: "low", Priority: 10, Members: []string{"mac:a"}, Policy: &models.GroupPolicy{Access: boolPointer(true), TargetID: "float:home"}},
			{ID: "high", Priority: 20, Members: []string{"mac:a"}, Policy: &models.GroupPolicy{Speed: &models.GroupSpeedPolicy{Enabled: true, UploadSpeed: 100, DownloadSpeed: 200}}},
		},
		DevicePolicies: map[string]*models.GroupPolicy{"mac:a": {TargetID: "self"}},
	}
	effective := resolveDeviceGroupPolicy(state, "mac:a", now)
	if !effective.NetworkAccess || effective.TargetID != "self" || effective.Speed.DownloadSpeed != 200 {
		t.Fatalf("unexpected effective policy: %#v", effective)
	}
	wantSources := []string{"system_default", "global", "group:low", "group:high", "device"}
	if fmt.Sprint(effective.Sources) != fmt.Sprint(wantSources) {
		t.Fatalf("sources = %v, want %v", effective.Sources, wantSources)
	}
}

func TestDeviceGroupScheduleWeekdayAndCrossMidnight(t *testing.T) {
	schedule := &models.GroupSchedule{ID: "bedtime", Enabled: true, Days: []int{1}, StartMinute: 22 * 60, EndMinute: 7 * 60, Action: "block"}
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"monday before", time.Date(2026, 9, 21, 21, 59, 0, 0, time.UTC), false},
		{"monday night", time.Date(2026, 9, 21, 23, 0, 0, 0, time.UTC), true},
		{"tuesday inherited", time.Date(2026, 9, 22, 6, 59, 0, 0, time.UTC), true},
		{"tuesday ended", time.Date(2026, 9, 22, 7, 0, 0, 0, time.UTC), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := groupScheduleActive(schedule, test.now); got != test.want {
				t.Fatalf("active = %v, want %v", got, test.want)
			}
		})
	}
}

func TestDeviceGroupScheduleSundayMondayAndDST(t *testing.T) {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	sundayNight := &models.GroupSchedule{ID: "sunday", Enabled: true, Days: []int{0}, StartMinute: 22 * 60, EndMinute: 7 * 60, Action: "block"}
	if !groupScheduleActive(sundayNight, time.Date(2026, 9, 28, 6, 30, 0, 0, location)) {
		t.Fatal("Sunday cross-midnight schedule did not carry into Monday")
	}
	dstWindow := &models.GroupSchedule{ID: "dst", Enabled: true, Days: []int{0}, StartMinute: 60, EndMinute: 4 * 60, Action: "block"}
	// Spring-forward has no 02:xx wall-clock hour. Evaluation remains based on
	// the configured local wall clock and therefore stays active at 03:30.
	if !groupScheduleActive(dstWindow, time.Date(2026, 3, 8, 3, 30, 0, 0, location)) {
		t.Fatal("DST spring-forward window was not deterministic")
	}
	firstFallback := time.Date(2026, 11, 1, 1, 30, 0, 0, location)
	secondFallback := firstFallback.Add(time.Hour)
	if !groupScheduleActive(dstWindow, firstFallback) || !groupScheduleActive(dstWindow, secondFallback) {
		t.Fatal("both repeated fall-back wall-clock evaluations must have the same result")
	}
}

func TestDeviceGroupStorePersistsAcrossRestartAndDeleteKeepsOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "groups.json")
	first := newJSONDeviceGroupStore(path)
	state, err := first.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state, err = first.Mutate(context.Background(), &models.DeviceGroupMutationRequest{
		Action: "set_device_policy", DeviceID: "mac:a", DevicePolicy: &models.GroupPolicy{Access: boolPointer(false)},
	}, state.Version)
	if err != nil {
		t.Fatal(err)
	}
	state, err = first.Mutate(context.Background(), &models.DeviceGroupMutationRequest{
		Action: "upsert_group", Group: &models.DeviceGroup{ID: "children", Name: "儿童设备", Priority: 10, Members: []string{"mac:a"}, Policy: &models.GroupPolicy{}},
	}, state.Version)
	if err != nil {
		t.Fatal(err)
	}
	second := newJSONDeviceGroupStore(path)
	restarted, err := second.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(restarted.Groups) != 1 || restarted.DevicePolicies["mac:a"] == nil {
		t.Fatalf("restart lost state: %#v", restarted)
	}
	deleted, err := second.Mutate(context.Background(), &models.DeviceGroupMutationRequest{Action: "delete_group", GroupID: "children"}, restarted.Version)
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted.Groups) != 0 || deleted.DevicePolicies["mac:a"] == nil {
		t.Fatalf("delete removed device override: %#v", deleted)
	}
}

func TestDeviceGroupCapacityAndEventBounds(t *testing.T) {
	document := emptyDeviceGroupDocument()
	for index := 0; index < deviceGroupLimit; index++ {
		members := make([]string, 16)
		for member := range members {
			members[member] = fmt.Sprintf("mac:%03d:%02d", index, member)
		}
		document.Groups = append(document.Groups, &models.DeviceGroup{
			ID: fmt.Sprintf("group_%03d", index), Name: fmt.Sprintf("Group %d", index), Members: members, Policy: &models.GroupPolicy{},
		})
	}
	for index := 0; index < deviceGroupPolicyLimit-deviceGroupLimit; index++ {
		document.DevicePolicies[fmt.Sprintf("device:%03d", index)] = &models.GroupPolicy{}
	}
	if err := validateDeviceGroupDocument(document); err != nil {
		t.Fatalf("document at exact limits rejected: %v", err)
	}
	document.Groups[0].Members = append(document.Groups[0].Members, "one-too-many")
	if err := validateDeviceGroupDocument(document); err == nil {
		t.Fatal("member overflow was accepted")
	}

	module := NewDeviceGroupModule(nil, nil)
	for index := 0; index < deviceGroupEventLimit+20; index++ {
		module.recordEvent(&models.DeviceGroupEvent{State: "degraded", Reason: fmt.Sprint(index)})
	}
	if len(module.events) != deviceGroupEventLimit || module.events[0].Reason != "20" {
		t.Fatalf("event ring not bounded: len=%d first=%q", len(module.events), module.events[0].Reason)
	}
}

func TestDeviceGroupDocumentRejectsAmbiguousOrOversizedIdentities(t *testing.T) {
	t.Parallel()

	tests := []deviceGroupDocument{
		{SchemaVersion: 1, Groups: []*models.DeviceGroup{
			{ID: "same", Name: "First", Members: []string{}, Policy: &models.GroupPolicy{}},
			{ID: " same ", Name: "Second", Members: []string{}, Policy: &models.GroupPolicy{}},
		}, DevicePolicies: map[string]*models.GroupPolicy{}},
		{SchemaVersion: 1, Groups: []*models.DeviceGroup{{ID: "group", Name: "Group", Members: []string{strings.Repeat("x", 257)}, Policy: &models.GroupPolicy{}}}, DevicePolicies: map[string]*models.GroupPolicy{}},
		{SchemaVersion: 1, Groups: []*models.DeviceGroup{}, DevicePolicies: map[string]*models.GroupPolicy{" mac:a": {}}},
		{SchemaVersion: 1, Groups: []*models.DeviceGroup{{ID: "group", Name: "Group", Members: []string{}, Policy: &models.GroupPolicy{Schedules: []*models.GroupSchedule{{ID: "bad id", Enabled: true, Days: []int{1}, StartMinute: 1, EndMinute: 2, Action: "block"}}}}}, DevicePolicies: map[string]*models.GroupPolicy{}},
	}
	for index, document := range tests {
		if err := validateDeviceGroupDocument(document); err == nil {
			t.Fatalf("case %d should fail", index)
		}
	}
}

type countingGroupPolicyApplier struct {
	calls    int
	policies []*models.EffectiveGroupPolicy
}

func (applier *countingGroupPolicyApplier) Apply(_ context.Context, policy *models.EffectiveGroupPolicy) error {
	applier.calls++
	copyValue := *policy
	applier.policies = append(applier.policies, &copyValue)
	return nil
}

type countingBatchGroupPolicyApplier struct {
	batchCalls int
	policies   int
	failAt     int
}

func (applier *countingBatchGroupPolicyApplier) Apply(context.Context, *models.EffectiveGroupPolicy) error {
	return errors.New("single-device apply must not be used")
}

func (applier *countingBatchGroupPolicyApplier) ApplyBatch(_ context.Context, policies []*models.EffectiveGroupPolicy, now time.Time) *models.DeviceGroupBatchResult {
	applier.batchCalls++
	applier.policies += len(policies)
	result := &models.DeviceGroupBatchResult{
		Status: "all_success", Planned: len(policies), Reloads: map[string]int{"firewall": 1},
		Items: make([]*models.DeviceGroupBatchItem, 0, len(policies)), EvaluatedAt: now.Format(time.RFC3339),
	}
	for index, policy := range policies {
		item := &models.DeviceGroupBatchItem{DeviceID: policy.DeviceID, Status: "applied", Sources: append([]string(nil), policy.Sources...)}
		if applier.failAt >= 0 && index == applier.failAt {
			item.Status, item.Reason = "failed", "injected_failure"
			result.Failed++
		} else {
			result.Applied++
		}
		result.Items = append(result.Items, item)
	}
	result.Status = groupBatchStatus(result.Applied, result.Failed, result.Planned)
	return result
}

type memoryDeviceGroupStore struct{ state deviceGroupState }

func (store *memoryDeviceGroupStore) Read(context.Context) (deviceGroupState, error) {
	return store.state, nil
}
func (store *memoryDeviceGroupStore) Mutate(context.Context, *models.DeviceGroupMutationRequest, string) (deviceGroupState, error) {
	return store.state, nil
}

type staticGroupInventory struct{ ids []string }

func (inventory staticGroupInventory) Snapshot(context.Context) (*models.DeviceInventoryResponse, error) {
	devices := make([]*models.DeviceInventoryItem, 0, len(inventory.ids))
	for _, id := range inventory.ids {
		devices = append(devices, &models.DeviceInventoryItem{DeviceID: id})
	}
	return &models.DeviceInventoryResponse{Result: &models.DeviceInventoryResult{Devices: devices}}, nil
}

func TestDeviceGroupReconcileUsesBoundedSinglePass(t *testing.T) {
	members := make([]string, 200)
	for index := range members {
		members[index] = fmt.Sprintf("mac:%03d", index)
	}
	store := &memoryDeviceGroupStore{state: deviceGroupState{
		Groups:         []*models.DeviceGroup{{ID: "all", Members: members, Policy: &models.GroupPolicy{Access: boolPointer(true)}}},
		DevicePolicies: map[string]*models.GroupPolicy{},
	}}
	applier := &countingGroupPolicyApplier{}
	module := NewDeviceGroupModule(store, applier)
	before := runtime.NumGoroutine()
	module.runDue(context.Background(), time.Now())
	if applier.calls != len(members) {
		t.Fatalf("apply calls = %d, want %d", applier.calls, len(members))
	}
	if growth := runtime.NumGoroutine() - before; growth > 1 {
		t.Fatalf("reconcile spawned per-device goroutines: growth=%d", growth)
	}
	module.runDue(context.Background(), time.Now())
	if applier.calls != len(members) {
		t.Fatalf("unchanged policies were applied again: %d", applier.calls)
	}
}

func TestDeviceGroupReconcileUsesOneBatchAndReportsPartialSources(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	store := &memoryDeviceGroupStore{state: deviceGroupState{
		GlobalPolicy: &models.GroupPolicy{Access: boolPointer(true)},
		Groups: []*models.DeviceGroup{{
			ID: "family", Priority: 10, Members: []string{"mac:a", "mac:b", "mac:c"},
			Policy: &models.GroupPolicy{Speed: &models.GroupSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 20}},
		}},
		DevicePolicies: map[string]*models.GroupPolicy{"mac:a": {TargetID: "self"}},
	}}
	applier := &countingBatchGroupPolicyApplier{failAt: 1}
	module := NewDeviceGroupModule(nil, applier)
	module.store = store // Avoid starting a background scheduler in this unit test.
	module.reconcileState(context.Background(), store.state, nil, now)

	if applier.batchCalls != 1 || applier.policies != 3 {
		t.Fatalf("batch calls=%d policies=%d, want 1/3", applier.batchCalls, applier.policies)
	}
	response := module.response(context.Background(), store.state, now)
	batch := response.Result.LastBatch
	if batch == nil || batch.Status != "partial" || batch.Applied != 2 || batch.Failed != 1 || batch.Reloads["firewall"] != 1 {
		t.Fatalf("unexpected batch result: %#v", batch)
	}
	if got := batch.Items[0].Sources; fmt.Sprint(got) != fmt.Sprint([]string{"system_default", "global", "group:family", "device"}) {
		t.Fatalf("source chain=%v", got)
	}
	if len(module.lastApplied) != 2 {
		t.Fatalf("failed item entered dedupe cache: %#v", module.lastApplied)
	}
}

func TestDefaultGroupBatchReloadsEachServiceAtMostOnce(t *testing.T) {
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	policy := newDevicePolicyModuleForTest(availablePolicyStore())
	applier := &defaultGroupPolicyApplier{policy: policy}
	oldRestrictions := groupBatchReloadRestrictions
	oldNetwork := groupBatchReloadNetwork
	t.Cleanup(func() {
		groupBatchReloadRestrictions = oldRestrictions
		groupBatchReloadNetwork = oldNetwork
	})
	restrictionReloads, networkReloads := 0, 0
	groupBatchReloadRestrictions = func(_ context.Context, configs []string) error {
		restrictionReloads++
		if fmt.Sprint(configs) != "[firewall eqos]" {
			t.Fatalf("reload configs=%v", configs)
		}
		return nil
	}
	groupBatchReloadNetwork = func(context.Context) error { networkReloads++; return nil }

	policies := []*models.EffectiveGroupPolicy{
		{DeviceID: "mac:a", NetworkAccess: false, Speed: &models.GroupSpeedPolicy{Enabled: true, UploadSpeed: 100, DownloadSpeed: 200}, Managed: []string{"access", "speed"}, Sources: []string{"system_default", "group:family"}},
		{DeviceID: "mac:a", NetworkAccess: true, Speed: &models.GroupSpeedPolicy{Enabled: true, UploadSpeed: 150, DownloadSpeed: 250}, Managed: []string{"access", "speed"}, Sources: []string{"system_default", "device"}},
	}
	result := applier.ApplyBatch(context.Background(), policies, now)
	if result.Status != "all_success" || result.Applied != 2 || restrictionReloads != 1 || networkReloads != 0 {
		t.Fatalf("result=%#v restrictionReloads=%d networkReloads=%d", result, restrictionReloads, networkReloads)
	}
	if result.Reloads["firewall"] != 1 || result.Reloads["eqos"] != 1 {
		t.Fatalf("reload report=%v", result.Reloads)
	}
}

func TestDefaultGroupBatchPreflightFailureWritesNothing(t *testing.T) {
	store := availablePolicyStore()
	store.policy.Capabilities["speed"] = &models.DevicePolicyCapability{State: "not_installed", Reason: "eqos is not installed"}
	applier := &defaultGroupPolicyApplier{policy: newDevicePolicyModuleForTest(store)}
	result := applier.ApplyBatch(context.Background(), []*models.EffectiveGroupPolicy{
		{DeviceID: "mac:a", NetworkAccess: false, Managed: []string{"access"}, Sources: []string{"system_default", "group:family"}},
		{DeviceID: "mac:b", Speed: &models.GroupSpeedPolicy{Enabled: true, UploadSpeed: 100, DownloadSpeed: 200}, Managed: []string{"speed"}, Sources: []string{"system_default", "group:family"}},
	}, time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC))
	if result.Status != "failed" || result.Failed != 2 || store.applyCalls != 0 || len(result.Reloads) != 0 {
		t.Fatalf("result=%#v writes=%d", result, store.applyCalls)
	}
	if !strings.Contains(result.Items[0].Reason, "preflight_failed") {
		t.Fatalf("missing preflight reason: %#v", result.Items)
	}
}

func TestCollectedGroupBatchCommitsEachConfigDomainOnce(t *testing.T) {
	oldAccess, oldSpeed, oldNetwork := groupBatchWriteAccess, groupBatchWriteSpeed, groupBatchWriteNetwork
	t.Cleanup(func() {
		groupBatchWriteAccess, groupBatchWriteSpeed, groupBatchWriteNetwork = oldAccess, oldSpeed, oldNetwork
	})
	accessCalls, speedCalls, networkCalls := 0, 0, 0
	groupBatchWriteAccess = func(_ string, changes []groupBatchRestrictionChange) error {
		accessCalls++
		if len(changes) != 256 {
			t.Fatalf("access changes=%d", len(changes))
		}
		return nil
	}
	groupBatchWriteSpeed = func(_ string, changes []groupBatchRestrictionChange) error {
		speedCalls++
		if len(changes) != 256 {
			t.Fatalf("speed changes=%d", len(changes))
		}
		return nil
	}
	groupBatchWriteNetwork = func(_ string, changes []groupBatchNetworkChange) error {
		networkCalls++
		if len(changes) != 256 {
			t.Fatalf("network changes=%d", len(changes))
		}
		return nil
	}
	accumulator := &groupBatchAccumulator{}
	for index := 0; index < 256; index++ {
		current := &models.DevicePolicy{DeviceID: fmt.Sprintf("mac:%03d", index)}
		accumulator.restrictions = append(accumulator.restrictions,
			groupBatchRestrictionChange{request: &models.DevicePolicyApplyRequest{Kind: "access", Access: &models.DeviceAccessPolicy{}}, current: current},
			groupBatchRestrictionChange{request: &models.DevicePolicyApplyRequest{Kind: "speed", Speed: &models.DeviceSpeedPolicy{}}, current: current},
		)
		accumulator.network = append(accumulator.network, groupBatchNetworkChange{})
	}
	if err := applyCollectedGroupBatch(accumulator); err != nil {
		t.Fatal(err)
	}
	if accessCalls != 1 || speedCalls != 1 || networkCalls != 1 {
		t.Fatalf("commit calls access=%d speed=%d network=%d", accessCalls, speedCalls, networkCalls)
	}
}

func TestDefaultGroupBatchPersistsAllQuotasOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traffic.json")
	store := newTrafficInsightsFileStore(path)
	persistCalls := 0
	store.persist = func(path string, raw []byte) error {
		persistCalls++
		return persistClassificationOverrides(path, raw)
	}
	traffic := NewTrafficInsightsModule(store, nil)
	applier := &defaultGroupPolicyApplier{traffic: traffic}
	result := applier.ApplyBatch(context.Background(), []*models.EffectiveGroupPolicy{
		{DeviceID: "mac:a", Managed: []string{"quota"}, Quota: &models.GroupQuotaPolicy{Enabled: true, Period: "monthly", LimitBytes: 1 << 30, Action: "notify"}},
		{DeviceID: "mac:b", Managed: []string{"quota"}, Quota: &models.GroupQuotaPolicy{Enabled: true, Period: "weekly", LimitBytes: 2 << 30, Action: "notify"}},
	}, time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC))
	if result.Status != "all_success" || result.Applied != 2 || persistCalls != 1 {
		t.Fatalf("result=%#v persistCalls=%d", result, persistCalls)
	}
	if len(traffic.document.Quotas) != 2 {
		t.Fatalf("quota count=%d", len(traffic.document.Quotas))
	}
}

func TestDeviceGroupRepeatedTickAndTimeJumpDoNotDuplicateApply(t *testing.T) {
	schedule := &models.GroupSchedule{ID: "school-night", Enabled: true, Days: []int{1}, StartMinute: 22 * 60, EndMinute: 7 * 60, Action: "block"}
	store := &memoryDeviceGroupStore{state: deviceGroupState{
		Groups:         []*models.DeviceGroup{{ID: "children", Members: []string{"mac:a"}, Policy: &models.GroupPolicy{Schedules: []*models.GroupSchedule{schedule}}}},
		DevicePolicies: map[string]*models.GroupPolicy{},
	}}
	applier := &countingBatchGroupPolicyApplier{failAt: -1}
	module := NewDeviceGroupModule(nil, applier)
	module.store = store
	mondayNight := time.Date(2026, 9, 21, 23, 0, 0, 0, time.UTC)
	module.runDue(context.Background(), mondayNight)
	module.runDue(context.Background(), mondayNight)                  // repeated tick
	module.runDue(context.Background(), mondayNight.Add(-time.Hour))  // backward jump, same state
	module.runDue(context.Background(), mondayNight.Add(8*time.Hour)) // cross-midnight transition
	module.runDue(context.Background(), mondayNight.Add(48*time.Hour))
	if applier.batchCalls != 2 || applier.policies != 2 {
		t.Fatalf("batch calls=%d policies=%d, want one block and one restore", applier.batchCalls, applier.policies)
	}
}

func TestDeviceGroupBatchPerformanceBudgets(t *testing.T) {
	for _, count := range []int{20, 256, 2048} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			members := make([]string, count)
			for index := range members {
				members[index] = fmt.Sprintf("mac:%04d", index)
			}
			store := &memoryDeviceGroupStore{state: deviceGroupState{
				Groups:         []*models.DeviceGroup{{ID: "all", Members: members, Policy: &models.GroupPolicy{Access: boolPointer(false)}}},
				DevicePolicies: map[string]*models.GroupPolicy{},
			}}
			applier := &countingBatchGroupPolicyApplier{failAt: -1}
			module := NewDeviceGroupModule(nil, applier)
			module.store = store
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			started := time.Now()
			module.runDue(context.Background(), time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC))
			elapsed := time.Since(started)
			runtime.ReadMemStats(&after)
			allocated := after.TotalAlloc - before.TotalAlloc
			if applier.batchCalls != 1 || applier.policies != count {
				t.Fatalf("batch calls=%d policies=%d", applier.batchCalls, applier.policies)
			}
			if elapsed > 3*time.Second || allocated > 64<<20 {
				t.Fatalf("resource budget exceeded: elapsed=%s allocated=%d", elapsed, allocated)
			}
			if module.lastBatch == nil || len(module.lastBatch.Items) != count || len(module.lastApplied) != count {
				t.Fatalf("bounded batch state mismatch: result=%#v cache=%d", module.lastBatch, len(module.lastApplied))
			}
		})
	}
}

func TestDeviceGroupHundredThousandMemberChurnRemainsBounded(t *testing.T) {
	applier := &countingBatchGroupPolicyApplier{failAt: -1}
	module := NewDeviceGroupModule(nil, applier)
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	previous := deviceGroupState{Groups: []*models.DeviceGroup{}, DevicePolicies: map[string]*models.GroupPolicy{}}
	for index := 0; index < 100_000; index++ {
		current := deviceGroupState{
			Groups: []*models.DeviceGroup{{
				ID: "moving", Members: []string{fmt.Sprintf("mac:%06d", index)},
				Policy: &models.GroupPolicy{Access: boolPointer(false)},
			}},
			DevicePolicies: map[string]*models.GroupPolicy{},
		}
		module.reconcileState(context.Background(), current, &previous, now)
		previous = current
	}
	if len(module.lastApplied) > 1 || len(module.events) > deviceGroupEventLimit || module.lastBatch == nil || len(module.lastBatch.Items) > 2 {
		t.Fatalf("unbounded churn state: cache=%d events=%d batch=%#v", len(module.lastApplied), len(module.events), module.lastBatch)
	}
}

func TestDeviceGroupRestartAfterMissedWindowRestoresPolicy(t *testing.T) {
	schedule := &models.GroupSchedule{ID: "night", Enabled: true, Days: []int{1}, StartMinute: 22 * 60, EndMinute: 7 * 60, Action: "block"}
	state := deviceGroupState{
		Groups:         []*models.DeviceGroup{{ID: "children", Members: []string{"mac:a"}, Policy: &models.GroupPolicy{Schedules: []*models.GroupSchedule{schedule}}}},
		DevicePolicies: map[string]*models.GroupPolicy{},
	}
	applier := &countingBatchGroupPolicyApplier{failAt: -1}
	module := NewDeviceGroupModule(nil, applier) // Empty cache models a process restart.
	module.reconcileState(context.Background(), state, nil, time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC))
	if applier.batchCalls != 1 || module.lastBatch.Items[0].Status != "applied" {
		t.Fatalf("missed-window restart did not converge: %#v", module.lastBatch)
	}
	if effective := resolveDeviceGroupPolicy(state, "mac:a", time.Date(2026, 9, 22, 8, 0, 0, 0, time.UTC)); !effective.NetworkAccess {
		t.Fatalf("post-window policy should restore access: %#v", effective)
	}
}

func TestDeviceGroupScheduleRestoresAndRemovedMemberIsReset(t *testing.T) {
	mondayNight := time.Date(2026, 9, 21, 23, 0, 0, 0, time.UTC)
	schedule := &models.GroupSchedule{ID: "rest", Enabled: true, Days: []int{1}, StartMinute: 22 * 60, EndMinute: 7 * 60, Action: "block"}
	store := &memoryDeviceGroupStore{state: deviceGroupState{Groups: []*models.DeviceGroup{{ID: "children", Members: []string{"mac:a"}, Policy: &models.GroupPolicy{Schedules: []*models.GroupSchedule{schedule}}}}, DevicePolicies: map[string]*models.GroupPolicy{}}}
	applier := &countingGroupPolicyApplier{}
	module := NewDeviceGroupModule(store, applier)
	module.runDue(context.Background(), mondayNight)
	module.runDue(context.Background(), mondayNight.Add(8*time.Hour))
	if applier.calls != 2 || applier.policies[0].NetworkAccess || !applier.policies[1].NetworkAccess {
		t.Fatalf("schedule did not block then restore: %#v", applier.policies)
	}
	previous := store.state
	store.state.Groups = []*models.DeviceGroup{}
	module.reconcileState(context.Background(), store.state, &previous, mondayNight)
	last := applier.policies[len(applier.policies)-1]
	if !last.NetworkAccess || len(last.Managed) == 0 || len(module.lastApplied) != 0 {
		t.Fatalf("removed member was not reset: %#v cache=%v", last, module.lastApplied)
	}
}

func TestDeviceGroupGlobalPolicyAppliesToInventoryAndRestores(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	previous := deviceGroupState{GlobalPolicy: &models.GroupPolicy{Access: boolPointer(false)}, Groups: []*models.DeviceGroup{}, DevicePolicies: map[string]*models.GroupPolicy{}}
	current := deviceGroupState{Groups: []*models.DeviceGroup{}, DevicePolicies: map[string]*models.GroupPolicy{}}
	applier := &countingGroupPolicyApplier{}
	module := NewDeviceGroupModule(&memoryDeviceGroupStore{state: previous}, applier)
	module.inventory = staticGroupInventory{ids: []string{"mac:a", "mac:b"}}
	module.reconcileState(context.Background(), previous, nil, now)
	if applier.calls != 2 || applier.policies[0].NetworkAccess || applier.policies[1].NetworkAccess {
		t.Fatalf("global policy did not cover inventory: %#v", applier.policies)
	}
	module.reconcileState(context.Background(), current, &previous, now.Add(time.Minute))
	if applier.calls != 4 || !applier.policies[2].NetworkAccess || !applier.policies[3].NetworkAccess {
		t.Fatalf("global policy removal did not restore inventory: %#v", applier.policies)
	}
	response := module.response(context.Background(), previous, now)
	if len(response.Result.Effective) != 2 {
		t.Fatalf("global effective policies=%d, want 2", len(response.Result.Effective))
	}
}

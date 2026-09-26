package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	deviceGroupLimit       = 128
	deviceGroupMemberLimit = 2048
	deviceGroupPolicyLimit = 512
	deviceGroupEventLimit  = 256
)

type deviceGroupState struct {
	GlobalPolicy   *models.GroupPolicy
	Groups         []*models.DeviceGroup
	DevicePolicies map[string]*models.GroupPolicy
	Version        string
}

type deviceGroupStore interface {
	Read(context.Context) (deviceGroupState, error)
	Mutate(context.Context, *models.DeviceGroupMutationRequest, string) (deviceGroupState, error)
}

type groupPolicyApplier interface {
	Apply(context.Context, *models.EffectiveGroupPolicy) error
}

type groupPolicyBatchApplier interface {
	ApplyBatch(context.Context, []*models.EffectiveGroupPolicy, time.Time) *models.DeviceGroupBatchResult
}

type DeviceGroupModule struct {
	mu          sync.Mutex
	eventsMu    sync.Mutex
	batchMu     sync.Mutex
	store       deviceGroupStore
	applier     groupPolicyApplier
	inventory   deviceInventorySnapshotter
	now         func() time.Time
	events      []*models.DeviceGroupEvent
	lastApplied map[string]string
	lastBatch   *models.DeviceGroupBatchResult
	startOnce   sync.Once
}

func NewDeviceGroupModule(store deviceGroupStore, applier groupPolicyApplier) *DeviceGroupModule {
	module := &DeviceGroupModule{store: store, applier: applier, now: time.Now, events: []*models.DeviceGroupEvent{}, lastApplied: map[string]string{}}
	if store != nil {
		state, err := store.Read(context.Background())
		if err == nil && deviceGroupStateHasWork(state) {
			module.startScheduler()
		}
	}
	return module
}

func NewDefaultDeviceGroupModule(policy *DevicePolicyModule, network *DeviceNetworkPolicyModule, traffic ...*TrafficInsightsModule) *DeviceGroupModule {
	return newDefaultDeviceGroupModuleWithStore(newJSONDeviceGroupStore(defaultDeviceGroupStorePath), policy, network, traffic...)
}

func newDefaultDeviceGroupModuleWithStore(store *jsonDeviceGroupStore, policy *DevicePolicyModule, network *DeviceNetworkPolicyModule, traffic ...*TrafficInsightsModule) *DeviceGroupModule {
	var insights *TrafficInsightsModule
	if len(traffic) > 0 {
		insights = traffic[0]
	}
	module := NewDeviceGroupModule(
		store,
		&defaultGroupPolicyApplier{policy: policy, network: network, traffic: insights},
	)
	if network != nil {
		module.inventory = network.inventory
	}
	return module
}

func (module *DeviceGroupModule) AttachTrafficInsights(traffic *TrafficInsightsModule) {
	if module == nil || traffic == nil {
		return
	}
	module.mu.Lock()
	defer module.mu.Unlock()
	if applier, ok := module.applier.(*defaultGroupPolicyApplier); ok {
		applier.traffic = traffic
	}
}

func (module *DeviceGroupModule) List(ctx context.Context) (*models.DeviceGroupsResponse, error) {
	state, err := module.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	if deviceGroupStateHasWork(state) {
		module.startScheduler()
	}
	return module.response(ctx, state, module.now()), nil
}

func (module *DeviceGroupModule) Mutate(ctx context.Context, request *models.DeviceGroupMutationRequest) (*models.DeviceGroupsResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	current, err := module.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	if validation := validateDeviceGroupMutation(request, current); validation != nil {
		return deviceGroupsFailure(validation), nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != current.Version {
		return deviceGroupsFailure(&models.DevicePolicyError{Code: "conflict", Message: "device groups changed; reload and try again"}), nil
	}
	updated, err := module.store.Mutate(ctx, request, current.Version)
	if err != nil {
		return deviceGroupsFailure(&models.DevicePolicyError{Code: "apply_failed", Message: err.Error()}), nil
	}
	if deviceGroupStateHasWork(updated) {
		module.startScheduler()
	}
	module.reconcileState(ctx, updated, &current, module.now())
	return module.response(ctx, updated, module.now()), nil
}

func (module *DeviceGroupModule) response(ctx context.Context, state deviceGroupState, now time.Time) *models.DeviceGroupsResponse {
	effective := make([]*models.EffectiveGroupPolicy, 0, len(state.DevicePolicies))
	ids := map[string]bool{}
	module.addGlobalPolicyDeviceIDs(ctx, state.GlobalPolicy != nil, ids, now)
	for _, group := range state.Groups {
		for _, id := range group.Members {
			ids[id] = true
		}
	}
	for id := range state.DevicePolicies {
		ids[id] = true
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	for _, id := range ordered {
		effective = append(effective, resolveDeviceGroupPolicy(state, id, now))
	}
	module.eventsMu.Lock()
	events := append([]*models.DeviceGroupEvent(nil), module.events...)
	module.eventsMu.Unlock()
	module.batchMu.Lock()
	lastBatch := module.lastBatch
	module.batchMu.Unlock()
	next := now.Truncate(time.Minute).Add(time.Minute)
	return &models.DeviceGroupsResponse{Result: &models.DeviceGroupsResult{
		GlobalPolicy: state.GlobalPolicy, Groups: state.Groups, DevicePolicies: state.DevicePolicies,
		Effective: effective, Events: events, Timezone: now.Location().String(),
		NextEvaluationAt: next.Format(time.RFC3339), Version: state.Version, LastBatch: lastBatch,
	}}
}

func resolveDeviceGroupPolicy(state deviceGroupState, deviceID string, now time.Time) *models.EffectiveGroupPolicy {
	result := &models.EffectiveGroupPolicy{
		DeviceID: deviceID, NetworkAccess: true, Speed: &models.GroupSpeedPolicy{}, TargetID: "default",
		Quota:   &models.GroupQuotaPolicy{},
		Managed: []string{}, Sources: []string{"system_default"}, Reasons: []string{}, EvaluatedAt: now.Format(time.RFC3339),
	}
	applyGroupPolicy(result, state.GlobalPolicy, "global", now)
	groups := make([]*models.DeviceGroup, 0)
	for _, group := range state.Groups {
		for _, member := range group.Members {
			if member == deviceID {
				groups = append(groups, group)
				break
			}
		}
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Priority != groups[j].Priority {
			return groups[i].Priority < groups[j].Priority
		}
		return groups[i].ID < groups[j].ID
	})
	for _, group := range groups {
		applyGroupPolicy(result, group.Policy, "group:"+group.ID, now)
	}
	if policy := state.DevicePolicies[deviceID]; policy != nil {
		applyGroupPolicy(result, policy, "device", now)
	}
	result.Managed = uniqueSortedStrings(result.Managed)
	return result
}

func applyGroupPolicy(result *models.EffectiveGroupPolicy, policy *models.GroupPolicy, source string, now time.Time) {
	if policy == nil {
		return
	}
	changed := false
	if policy.Access != nil {
		result.NetworkAccess = *policy.Access
		result.Managed = append(result.Managed, "access")
		changed = true
	}
	if policy.Speed != nil {
		copyValue := *policy.Speed
		result.Speed = &copyValue
		result.Managed = append(result.Managed, "speed")
		changed = true
	}
	if policy.TargetID != "" {
		result.TargetID = policy.TargetID
		result.Managed = append(result.Managed, "route")
		changed = true
	}
	if policy.Quota != nil {
		copyValue := *policy.Quota
		result.Quota = &copyValue
		result.Managed = append(result.Managed, "quota")
		changed = true
	}
	for _, schedule := range policy.Schedules {
		if schedule == nil || !schedule.Enabled {
			continue
		}
		switch schedule.Action {
		case "block":
			result.Managed = append(result.Managed, "access")
		case "limit":
			result.Managed = append(result.Managed, "speed")
		}
		if !groupScheduleActive(schedule, now) {
			continue
		}
		switch schedule.Action {
		case "block":
			result.NetworkAccess = false
			result.Managed = append(result.Managed, "access")
			result.Reasons = append(result.Reasons, "schedule:"+schedule.ID)
			changed = true
		case "limit":
			if schedule.Speed != nil {
				copyValue := *schedule.Speed
				result.Speed = &copyValue
				result.Managed = append(result.Managed, "speed")
				result.Reasons = append(result.Reasons, "schedule:"+schedule.ID)
				changed = true
			}
		}
	}
	if changed {
		result.Sources = append(result.Sources, source)
	}
}

func groupScheduleActive(schedule *models.GroupSchedule, now time.Time) bool {
	if schedule == nil || !schedule.Enabled || schedule.StartMinute < 0 || schedule.StartMinute > 1439 ||
		schedule.EndMinute < 0 || schedule.EndMinute > 1439 {
		return false
	}
	minute := now.Hour()*60 + now.Minute()
	days := map[int]bool{}
	for _, day := range schedule.Days {
		days[day] = true
	}
	weekday := int(now.Weekday())
	if schedule.StartMinute < schedule.EndMinute {
		return days[weekday] && minute >= schedule.StartMinute && minute < schedule.EndMinute
	}
	previous := (weekday + 6) % 7
	return (days[weekday] && minute >= schedule.StartMinute) || (days[previous] && minute < schedule.EndMinute)
}

func validateDeviceGroupMutation(request *models.DeviceGroupMutationRequest, state deviceGroupState) *models.DevicePolicyError {
	if request == nil {
		return &models.DevicePolicyError{Code: "validation_failed", Message: "request is required"}
	}
	switch request.Action {
	case "upsert_group":
		if request.Group == nil {
			return &models.DevicePolicyError{Code: "validation_failed", Message: "group is required"}
		}
		if err := validateGroup(request.Group); err != nil {
			return &models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}
		}
		found := false
		members := 0
		policies := len(state.DevicePolicies)
		for _, group := range state.Groups {
			if group.ID == request.Group.ID {
				found = true
				continue
			}
			members += len(group.Members)
			if group.Policy != nil {
				policies++
			}
		}
		if !found && len(state.Groups) >= deviceGroupLimit {
			return &models.DevicePolicyError{Code: "capacity_exceeded", Message: "device group limit reached"}
		}
		members += len(request.Group.Members)
		if request.Group.Policy != nil {
			policies++
		}
		if members > deviceGroupMemberLimit || policies > deviceGroupPolicyLimit {
			return &models.DevicePolicyError{Code: "capacity_exceeded", Message: "device group capacity reached"}
		}
	case "delete_group":
		if strings.TrimSpace(request.GroupID) == "" {
			return &models.DevicePolicyError{Code: "validation_failed", Message: "groupId is required"}
		}
	case "set_device_policy":
		if strings.TrimSpace(request.DeviceID) == "" {
			return &models.DevicePolicyError{Code: "validation_failed", Message: "deviceId is required"}
		}
		if request.DevicePolicy != nil {
			if err := validateGroupPolicy(request.DevicePolicy); err != nil {
				return &models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}
			}
		}
		if _, exists := state.DevicePolicies[request.DeviceID]; !exists && request.DevicePolicy != nil && len(state.DevicePolicies) >= deviceGroupPolicyLimit {
			return &models.DevicePolicyError{Code: "capacity_exceeded", Message: "device policy limit reached"}
		}
	case "set_global_policy":
		if request.GlobalPolicy != nil {
			if err := validateGroupPolicy(request.GlobalPolicy); err != nil {
				return &models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}
			}
		}
	default:
		return &models.DevicePolicyError{Code: "validation_failed", Message: "unsupported device group action"}
	}
	return nil
}

func validateGroup(group *models.DeviceGroup) error {
	group.ID = strings.TrimSpace(group.ID)
	group.Name = strings.TrimSpace(group.Name)
	if group.ID == "" || group.Name == "" {
		return errors.New("group id and name are required")
	}
	if !dhcpTagNamePattern.MatchString(group.ID) {
		return errors.New("group id contains unsupported characters")
	}
	if !utf8.ValidString(group.Name) || utf8.RuneCountInString(group.Name) > 64 {
		return errors.New("group name is too long")
	}
	for _, r := range group.Name {
		if unicode.IsControl(r) {
			return errors.New("group name contains control characters")
		}
	}
	seen := map[string]bool{}
	members := group.Members[:0]
	for _, member := range group.Members {
		member = strings.TrimSpace(member)
		if !validDeviceIdentityKey(member) {
			return errors.New("group member has an invalid device identity")
		}
		if !seen[member] {
			seen[member] = true
			members = append(members, member)
		}
	}
	sort.Strings(members)
	group.Members = members
	if group.Policy == nil {
		group.Policy = &models.GroupPolicy{Schedules: []*models.GroupSchedule{}}
	}
	return validateGroupPolicy(group.Policy)
}

func validateGroupPolicy(policy *models.GroupPolicy) error {
	if policy == nil {
		return nil
	}
	if policy.Speed != nil && policy.Speed.Enabled && (policy.Speed.UploadSpeed <= 0 || policy.Speed.DownloadSpeed <= 0) {
		return errors.New("speed limits must be greater than zero")
	}
	if policy.TargetID != "" && (!utf8.ValidString(policy.TargetID) || len(policy.TargetID) > 128 || strings.TrimSpace(policy.TargetID) != policy.TargetID) {
		return errors.New("invalid internet path target")
	}
	if policy.Quota != nil {
		request := &models.TrafficQuotaRequest{DeviceID: "validation", Enabled: policy.Quota.Enabled, Period: policy.Quota.Period, LimitBytes: policy.Quota.LimitBytes, Action: policy.Quota.Action}
		if err := validateTrafficQuotaRequest(request); err != nil {
			return err
		}
	}
	if len(policy.Schedules) > 64 {
		return errors.New("too many schedules in one policy")
	}
	seen := map[string]bool{}
	for _, schedule := range policy.Schedules {
		if schedule == nil || !dhcpTagNamePattern.MatchString(schedule.ID) || seen[schedule.ID] || len(schedule.Days) == 0 ||
			schedule.StartMinute < 0 || schedule.StartMinute > 1439 || schedule.EndMinute < 0 || schedule.EndMinute > 1439 ||
			schedule.StartMinute == schedule.EndMinute || (schedule.Action != "block" && schedule.Action != "limit") {
			return errors.New("invalid group schedule")
		}
		seen[schedule.ID] = true
		for _, day := range schedule.Days {
			if day < 0 || day > 6 {
				return errors.New("invalid schedule weekday")
			}
		}
		if schedule.Action == "limit" && (schedule.Speed == nil || !schedule.Speed.Enabled || schedule.Speed.UploadSpeed <= 0 || schedule.Speed.DownloadSpeed <= 0) {
			return errors.New("scheduled speed limits must be greater than zero")
		}
	}
	return nil
}

func validDeviceIdentityKey(value string) bool {
	if value == "" || strings.TrimSpace(value) != value || !utf8.ValidString(value) || len(value) > 256 {
		return false
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return false
		}
	}
	return true
}

func deviceGroupsFailure(err *models.DevicePolicyError) *models.DeviceGroupsResponse {
	return &models.DeviceGroupsResponse{Result: &models.DeviceGroupsResult{
		Groups: []*models.DeviceGroup{}, DevicePolicies: map[string]*models.GroupPolicy{},
		Effective: []*models.EffectiveGroupPolicy{}, Events: []*models.DeviceGroupEvent{}, Error: err,
	}}
}

func (module *DeviceGroupModule) startScheduler() {
	module.startOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for now := range ticker.C {
				module.runDue(context.Background(), now)
			}
		}()
	})
}

func (module *DeviceGroupModule) runDue(ctx context.Context, now time.Time) {
	module.mu.Lock()
	defer module.mu.Unlock()
	state, err := module.store.Read(ctx)
	if err != nil {
		module.recordEvent(&models.DeviceGroupEvent{At: now.Format(time.RFC3339), State: "error", Reason: err.Error()})
		return
	}
	module.reconcileState(ctx, state, nil, now)
}

func (module *DeviceGroupModule) reconcileState(ctx context.Context, state deviceGroupState, previous *deviceGroupState, now time.Time) {
	started := time.Now()
	ids := map[string]bool{}
	currentIDs := map[string]bool{}
	module.addGlobalPolicyDeviceIDs(ctx, state.GlobalPolicy != nil || (previous != nil && previous.GlobalPolicy != nil), ids, now)
	if state.GlobalPolicy != nil {
		for id := range ids {
			currentIDs[id] = true
		}
	}
	for _, group := range state.Groups {
		for _, id := range group.Members {
			ids[id], currentIDs[id] = true, true
		}
	}
	for id := range state.DevicePolicies {
		ids[id], currentIDs[id] = true, true
	}
	if previous != nil {
		for _, group := range previous.Groups {
			for _, id := range group.Members {
				ids[id] = true
			}
		}
		for id := range previous.DevicePolicies {
			ids[id] = true
		}
	}
	ordered := make([]string, 0, len(ids))
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	pending := make([]*models.EffectiveGroupPolicy, 0, len(ordered))
	signatures := make(map[string]string, len(ordered))
	current := make(map[string]bool, len(ordered))
	skipped := 0
	for _, id := range ordered {
		effective := resolveDeviceGroupPolicy(state, id, now)
		if previous != nil && len(effective.Managed) == 0 {
			before := resolveDeviceGroupPolicy(*previous, id, now)
			effective.Managed = uniqueSortedStrings(append(effective.Managed, before.Managed...))
		}
		if module.applier == nil || len(effective.Managed) == 0 {
			if !currentIDs[id] {
				delete(module.lastApplied, id)
			}
			skipped++
			continue
		}
		signature := effectiveGroupPolicySignature(effective)
		if module.lastApplied[id] == signature {
			if !currentIDs[id] {
				delete(module.lastApplied, id)
			}
			skipped++
			continue
		}
		pending = append(pending, effective)
		signatures[id] = signature
		current[id] = currentIDs[id]
	}
	batch := &models.DeviceGroupBatchResult{Status: "unchanged", Planned: len(pending), Reloads: map[string]int{}, Items: []*models.DeviceGroupBatchItem{}, EvaluatedAt: now.Format(time.RFC3339)}
	if batchApplier, ok := module.applier.(groupPolicyBatchApplier); ok && len(pending) > 0 {
		batch = batchApplier.ApplyBatch(ctx, pending, now)
	} else {
		for _, effective := range pending {
			item := &models.DeviceGroupBatchItem{DeviceID: effective.DeviceID, Sources: append([]string(nil), effective.Sources...)}
			if err := module.applier.Apply(ctx, effective); err != nil {
				item.Status, item.Reason = "failed", err.Error()
				batch.Failed++
			} else {
				item.Status = "applied"
				batch.Applied++
			}
			batch.Items = append(batch.Items, item)
		}
	}
	batch.Skipped += skipped
	batch.DurationMS = time.Since(started).Milliseconds()
	batch.Status = groupBatchStatus(batch.Applied, batch.Failed, batch.Planned)
	for _, item := range batch.Items {
		if item.Status != "applied" {
			module.recordEvent(&models.DeviceGroupEvent{At: now.Format(time.RFC3339), DeviceID: item.DeviceID, State: "degraded", Reason: item.Reason})
			continue
		}
		if current[item.DeviceID] {
			module.lastApplied[item.DeviceID] = signatures[item.DeviceID]
		} else {
			delete(module.lastApplied, item.DeviceID)
		}
	}
	module.batchMu.Lock()
	module.lastBatch = batch
	module.batchMu.Unlock()
}

func groupBatchStatus(applied, failed, planned int) string {
	if planned == 0 {
		return "unchanged"
	}
	if failed == 0 {
		return "all_success"
	}
	if applied == 0 {
		return "failed"
	}
	return "partial"
}

func (module *DeviceGroupModule) addGlobalPolicyDeviceIDs(ctx context.Context, needed bool, ids map[string]bool, now time.Time) {
	if !needed || module.inventory == nil {
		return
	}
	response, err := module.inventory.Snapshot(ctx)
	if err != nil || response == nil || response.Result == nil {
		reason := "device inventory is unavailable"
		if err != nil {
			reason = err.Error()
		}
		module.recordEvent(&models.DeviceGroupEvent{At: now.Format(time.RFC3339), State: "degraded", Reason: reason})
		return
	}
	for _, device := range response.Result.Devices {
		if device != nil && device.DeviceID != "" {
			ids[device.DeviceID] = true
		}
	}
}

func effectiveGroupPolicySignature(effective *models.EffectiveGroupPolicy) string {
	speed := effective.Speed
	if speed == nil {
		speed = &models.GroupSpeedPolicy{}
	}
	return fmt.Sprintf("%t|%t:%d:%d|%s|%s|%s", effective.NetworkAccess, speed.Enabled, speed.UploadSpeed, speed.DownloadSpeed, effective.TargetID, strings.Join(effective.Managed, ","), strings.Join(effective.Reasons, ","))
}

func (module *DeviceGroupModule) recordEvent(event *models.DeviceGroupEvent) {
	module.eventsMu.Lock()
	defer module.eventsMu.Unlock()
	module.events = append(module.events, event)
	if len(module.events) > deviceGroupEventLimit {
		module.events = module.events[len(module.events)-deviceGroupEventLimit:]
	}
}

func deviceGroupStateHasWork(state deviceGroupState) bool {
	return state.GlobalPolicy != nil || len(state.Groups) > 0 || len(state.DevicePolicies) > 0
}

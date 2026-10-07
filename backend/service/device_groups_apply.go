package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type groupBatchContextKey struct{}

type groupBatchRestrictionChange struct {
	request *models.DevicePolicyApplyRequest
	current *models.DevicePolicy
}

type groupBatchNetworkChange struct {
	input StaticAssignmentWriteInput
	route gatewayPolicyExecutionPlan
}

type groupBatchAccumulator struct {
	restrictions []groupBatchRestrictionChange
	network      []groupBatchNetworkChange
	quotas       []*models.TrafficQuotaRequest
}

func groupBatchAccumulatorFrom(ctx context.Context) *groupBatchAccumulator {
	accumulator, _ := ctx.Value(groupBatchContextKey{}).(*groupBatchAccumulator)
	return accumulator
}

func groupBatchDefersReload(ctx context.Context) bool {
	return groupBatchAccumulatorFrom(ctx) != nil
}

var (
	groupBatchReloadRestrictions = func(ctx context.Context, configs []string) error {
		if err := deviceRestrictionApply(ctx, configs); err != nil {
			return err
		}
		for _, config := range configs {
			if config == "firewall" {
				return deviceAccessRestrictionApply(ctx)
			}
		}
		return nil
	}
	groupBatchReloadNetwork = func(ctx context.Context) error { return deviceNetworkPolicyReload(ctx) }
	groupBatchWriteAccess   = writeDeviceAccessPolicyBatchAt
	groupBatchWriteSpeed    = writeDeviceSpeedPolicyBatchAt
	groupBatchWriteNetwork  = mutateDeviceNetworkPoliciesConfigAt
)

type defaultGroupPolicyApplier struct {
	policy  *DevicePolicyModule
	network *DeviceNetworkPolicyModule
	traffic *TrafficInsightsModule
}

func (applier *defaultGroupPolicyApplier) ApplyBatch(ctx context.Context, policies []*models.EffectiveGroupPolicy, now time.Time) *models.DeviceGroupBatchResult {
	result := &models.DeviceGroupBatchResult{
		Status: "unchanged", Planned: len(policies), Reloads: map[string]int{}, Items: make([]*models.DeviceGroupBatchItem, 0, len(policies)), EvaluatedAt: now.Format(time.RFC3339),
	}
	if len(policies) == 0 {
		return result
	}
	// Validate the complete plan before taking snapshots or writing any managed
	// configuration. A newly unavailable capability must reject the whole batch;
	// desired group rules remain stored and will be retried when it returns.
	for _, policy := range policies {
		if err := applier.preflight(ctx, policy); err != nil {
			return failedGroupBatch(policies, now, "preflight_failed: "+err.Error())
		}
	}
	need := map[string]bool{}
	for _, policy := range policies {
		for _, kind := range policy.Managed {
			switch kind {
			case "access":
				need["firewall"] = true
			case "speed":
				need["speed"] = true
			case "route":
				need["dnsmasq"] = true
			case "quota":
				need["traffic"] = true
			}
		}
	}
	rateLimit := groupRateLimitModule(applier.policy)
	speedService := "eqos"
	if need["speed"] && rateLimit != nil {
		if provider := rateLimit.provider(ctx); provider != nil {
			speedService = provider.Name()
		}
	}
	if need["speed"] {
		need[speedService] = true
	}
	backups := map[string]devicePolicyBackup{}
	if applier.policy != nil {
		for _, pair := range []struct{ service, kind string }{{"firewall", "access"}, {"eqos", "speed"}} {
			if !need[pair.service] {
				continue
			}
			backup, err := applier.policy.store.Backup(ctx, pair.kind)
			if err != nil {
				return failedGroupBatch(policies, now, "snapshot_failed")
			}
			backups[pair.service] = backup
		}
	}
	var networkBackup dhcpConfigSnapshot
	if need["dnsmasq"] {
		var err error
		networkBackup, err = deviceNetworkPolicySnapshot()
		if err != nil {
			return failedGroupBatch(policies, now, "snapshot_failed")
		}
	}

	accumulator := &groupBatchAccumulator{
		restrictions: make([]groupBatchRestrictionChange, 0, len(policies)*2),
		network:      make([]groupBatchNetworkChange, 0, len(policies)),
		quotas:       make([]*models.TrafficQuotaRequest, 0, len(policies)),
	}
	deferred := context.WithValue(ctx, groupBatchContextKey{}, accumulator)
	successful := make([]*models.DeviceGroupBatchItem, 0, len(policies))
	for _, policy := range policies {
		item := &models.DeviceGroupBatchItem{DeviceID: policy.DeviceID, Sources: append([]string(nil), policy.Sources...)}
		if err := applier.Apply(deferred, policy); err != nil {
			item.Status, item.Reason = "failed", err.Error()
			result.Failed++
		} else {
			item.Status = "applied"
			result.Applied++
			successful = append(successful, item)
		}
		result.Items = append(result.Items, item)
	}
	if len(successful) == 0 {
		result.Status = "failed"
		return result
	}
	if result.Failed > 0 {
		result.Applied, result.Failed = 0, len(result.Items)
		for _, item := range result.Items {
			item.Status, item.Reason = "failed", "batch_apply_failed_before_commit"
		}
		result.Status = "failed"
		return result
	}

	var quotaSnapshot *trafficInsightsDocument
	if len(accumulator.quotas) > 0 {
		var quotaErr error
		quotaSnapshot, quotaErr = applier.traffic.SetQuotaBatch(deferred, accumulator.quotas)
		if quotaErr != nil {
			return failedGroupBatch(policies, now, "quota_batch_failed: "+quotaErr.Error())
		}
	}
	commit, reloadErr := applyCollectedGroupBatch(ctx, accumulator, rateLimit)
	configs := make([]string, 0, 2)
	if reloadErr == nil && need["firewall"] {
		configs = append(configs, "firewall")
		result.Reloads["firewall"] = 1
	}
	if reloadErr == nil && need["eqos"] {
		configs = append(configs, "eqos")
		result.Reloads["eqos"] = 1
	}
	if reloadErr == nil && len(configs) > 0 {
		reloadErr = groupBatchReloadRestrictions(ctx, configs)
	}
	if reloadErr == nil && need["dnsmasq"] {
		result.Reloads["dnsmasq"] = 1
		reloadErr = groupBatchReloadNetwork(ctx)
	}
	if reloadErr != nil {
		recoveryFailed := false
		if commit != nil && commit.rollback != nil && commit.rollback(ctx) != nil {
			recoveryFailed = true
		}
		if quotaSnapshot != nil && applier.traffic.RestoreQuotaBatch(*quotaSnapshot) != nil {
			recoveryFailed = true
		}
		for service, backup := range backups {
			_ = service
			if applier.policy.store.Restore(ctx, backup) != nil {
				recoveryFailed = true
			}
		}
		if need["dnsmasq"] && deviceNetworkPolicyRestore(ctx, networkBackup) != nil {
			recoveryFailed = true
		}
		result.Applied = 0
		result.Failed = len(result.Items)
		for _, item := range result.Items {
			item.Status = "rolled_back"
			item.Reason = "batch_reload_failed"
			if recoveryFailed {
				item.Status, item.Reason = "recovery_required", "batch_restore_failed"
			}
		}
		result.Status = "failed"
		return result
	}
	result.Status = groupBatchStatus(result.Applied, result.Failed, result.Planned)
	return result
}

type groupBatchProviderCommit struct{ rollback func(context.Context) error }

func applyCollectedGroupBatch(ctx context.Context, accumulator *groupBatchAccumulator, rateLimit *RateLimitModule) (*groupBatchProviderCommit, error) {
	if accumulator == nil {
		return nil, errors.New("batch accumulator unavailable")
	}
	access := make([]groupBatchRestrictionChange, 0, len(accumulator.restrictions))
	speed := make([]groupBatchRestrictionChange, 0, len(accumulator.restrictions))
	for _, change := range accumulator.restrictions {
		switch change.request.Kind {
		case "access":
			access = append(access, change)
		case "speed":
			speed = append(speed, change)
		}
	}
	configDir := deviceRestrictionConfigDir()
	if len(access) > 0 {
		if err := groupBatchWriteAccess(configDir, access); err != nil {
			return nil, err
		}
	}
	commit := &groupBatchProviderCommit{}
	if len(speed) > 0 {
		provider := rateLimitProvider(nil)
		if rateLimit != nil {
			provider = rateLimit.provider(ctx)
		}
		if provider != nil && provider.Name() != "eqos" {
			rollback, err := applyProviderGroupSpeedBatch(ctx, rateLimit, speed)
			commit.rollback = rollback
			if err != nil {
				commit.rollback = nil // applyProviderGroupSpeedBatch already compensated partial writes.
				return commit, err
			}
		} else if err := groupBatchWriteSpeed(configDir, speed); err != nil {
			return commit, err
		}
	}
	if len(accumulator.network) > 0 {
		if err := groupBatchWriteNetwork(configDir, accumulator.network); err != nil {
			return commit, err
		}
	}
	return commit, nil
}

func groupRateLimitModule(policy *DevicePolicyModule) *RateLimitModule {
	if policy == nil {
		return nil
	}
	store, _ := policy.store.(*systemDevicePolicyStore)
	if store == nil {
		return nil
	}
	return store.rateLimit
}

func applyProviderGroupSpeedBatch(ctx context.Context, module *RateLimitModule, changes []groupBatchRestrictionChange) (func(context.Context) error, error) {
	type appliedChange struct {
		target   rateLimitTarget
		previous models.DeviceSpeedPolicy
	}
	applied := make([]appliedChange, 0, len(changes))
	rollback := func(rollbackContext context.Context) error {
		var failures []string
		for index := len(applied) - 1; index >= 0; index-- {
			if err := module.Apply(rollbackContext, applied[index].target, applied[index].previous); err != nil {
				failures = append(failures, err.Error())
			}
		}
		if len(failures) > 0 {
			return errors.New(strings.Join(failures, "; "))
		}
		return nil
	}
	for _, change := range changes {
		if change.current == nil || change.request == nil || change.request.Speed == nil {
			return rollback, errors.New("invalid provider speed batch change")
		}
		reservedIP := ""
		if change.current.Static != nil {
			reservedIP = change.current.Static.AssignedIP
		}
		target := rateLimitTarget{DeviceID: change.current.DeviceID, MAC: change.current.MAC, DisplayName: change.current.DisplayName, CurrentIPv4: change.current.CurrentIPv4, ReservedIP: reservedIP}
		previous := models.DeviceSpeedPolicy{}
		if change.current.Speed != nil {
			previous = *change.current.Speed
		}
		if err := module.Apply(ctx, target, *change.request.Speed); err != nil {
			_ = rollback(ctx)
			return rollback, err
		}
		applied = append(applied, appliedChange{target: target, previous: previous})
	}
	return rollback, nil
}

func (applier *defaultGroupPolicyApplier) preflight(ctx context.Context, effective *models.EffectiveGroupPolicy) error {
	if effective == nil || effective.DeviceID == "" {
		return errors.New("invalid effective group policy")
	}
	managed := map[string]bool{}
	for _, kind := range effective.Managed {
		managed[kind] = true
	}
	if managed["access"] || managed["speed"] {
		if applier.policy == nil || applier.policy.store == nil {
			return errors.New("device policy service unavailable")
		}
		current, err := applier.policy.store.Get(ctx, effective.DeviceID)
		if err != nil {
			return err
		}
		for _, kind := range []string{"access", "speed"} {
			if !managed[kind] {
				continue
			}
			capability := current.Capabilities[kind]
			if capability == nil || capability.State != "available" {
				reason := kind + " capability unavailable"
				if capability != nil && capability.Reason != "" {
					reason = capability.Reason
				}
				return errors.New(reason)
			}
		}
	}
	if managed["route"] {
		if applier.network == nil || applier.network.gateway == nil {
			return errors.New("internet path service unavailable")
		}
		current, err := applier.network.Get(ctx, effective.DeviceID)
		if err != nil {
			return err
		}
		if current == nil || current.Result == nil || current.Result.Error != nil || current.Result.Policy == nil {
			return errors.New(groupNetworkPolicyResponseError(current))
		}
		plan, err := applier.network.gateway.plan(ctx, &models.GatewayAssignmentRequest{
			Action: "assign", DeviceID: effective.DeviceID, TargetID: effective.TargetID,
			ExpectedVersion: current.Result.Policy.Version,
		})
		if err != nil {
			return err
		}
		if plan.Public == nil || plan.Public.Error != nil || !plan.Public.CanApply {
			reason := "internet path target unavailable"
			if plan.Public != nil && plan.Public.Error != nil && plan.Public.Error.Message != "" {
				reason = plan.Public.Error.Message
			}
			return errors.New(reason)
		}
		if plan.State.DHCP != nil && plan.State.DHCP.DhcpIgnore {
			return errors.New("DHCP authority unavailable")
		}
	}
	if managed["quota"] {
		if applier.traffic == nil || effective.Quota == nil {
			return errors.New("traffic quota service unavailable")
		}
		if err := validateTrafficQuotaRequest(&models.TrafficQuotaRequest{DeviceID: effective.DeviceID, Enabled: effective.Quota.Enabled, Period: effective.Quota.Period, LimitBytes: effective.Quota.LimitBytes, Action: effective.Quota.Action}); err != nil {
			return err
		}
	}
	return nil
}

func failedGroupBatch(policies []*models.EffectiveGroupPolicy, now time.Time, reason string) *models.DeviceGroupBatchResult {
	result := &models.DeviceGroupBatchResult{Status: "failed", Planned: len(policies), Failed: len(policies), Reloads: map[string]int{}, Items: make([]*models.DeviceGroupBatchItem, 0, len(policies)), EvaluatedAt: now.Format(time.RFC3339)}
	for _, policy := range policies {
		result.Items = append(result.Items, &models.DeviceGroupBatchItem{DeviceID: policy.DeviceID, Status: "failed", Reason: reason, Sources: append([]string(nil), policy.Sources...)})
	}
	return result
}

func (applier *defaultGroupPolicyApplier) Apply(ctx context.Context, effective *models.EffectiveGroupPolicy) error {
	if effective == nil || effective.DeviceID == "" {
		return errors.New("invalid effective group policy")
	}
	managed := map[string]bool{}
	for _, kind := range effective.Managed {
		managed[kind] = true
	}
	failures := make([]string, 0, 3)
	if managed["access"] {
		if applier.policy == nil {
			failures = append(failures, "access policy service unavailable")
		} else {
			response, err := applier.policy.Apply(ctx, &models.DevicePolicyApplyRequest{
				DeviceID: effective.DeviceID, Kind: "access",
				Access: &models.DeviceAccessPolicy{NetworkAccess: effective.NetworkAccess},
			})
			if err != nil {
				failures = append(failures, err.Error())
			} else if response == nil || response.Result == nil || response.Result.Error != nil {
				failures = append(failures, groupPolicyResponseError(response))
			}
		}
	}
	if managed["speed"] {
		if applier.policy == nil {
			failures = append(failures, "speed policy service unavailable")
		} else {
			speed := effective.Speed
			if speed == nil {
				speed = &models.GroupSpeedPolicy{}
			}
			response, err := applier.policy.Apply(ctx, &models.DevicePolicyApplyRequest{
				DeviceID: effective.DeviceID, Kind: "speed",
				Speed: &models.DeviceSpeedPolicy{Enabled: speed.Enabled, UploadSpeed: speed.UploadSpeed, DownloadSpeed: speed.DownloadSpeed},
			})
			if err != nil {
				failures = append(failures, err.Error())
			} else if response == nil || response.Result == nil || response.Result.Error != nil {
				failures = append(failures, groupPolicyResponseError(response))
			}
		}
	}
	if managed["route"] {
		if applier.network == nil {
			failures = append(failures, "internet path service unavailable")
		} else {
			current, err := applier.network.Get(ctx, effective.DeviceID)
			if err != nil {
				failures = append(failures, err.Error())
			} else if current == nil || current.Result == nil || current.Result.Error != nil || current.Result.Policy == nil {
				failures = append(failures, groupNetworkPolicyResponseError(current))
			} else {
				response, applyErr := applier.network.Apply(ctx, &models.DeviceNetworkPolicyApplyRequest{
					DeviceID: effective.DeviceID, Static: current.Result.Policy.Static,
					TargetID: effective.TargetID, ExpectedVersion: current.Result.Policy.Version,
				})
				if applyErr != nil {
					failures = append(failures, applyErr.Error())
				} else if response == nil || response.Result == nil || response.Result.Error != nil {
					failures = append(failures, groupNetworkPolicyResponseError(response))
				}
			}
		}
	}
	if managed["quota"] {
		if applier.traffic == nil || effective.Quota == nil {
			failures = append(failures, "traffic quota service unavailable")
		} else if accumulator := groupBatchAccumulatorFrom(ctx); accumulator != nil {
			accumulator.quotas = append(accumulator.quotas, &models.TrafficQuotaRequest{DeviceID: effective.DeviceID, Enabled: effective.Quota.Enabled, Period: effective.Quota.Period, LimitBytes: effective.Quota.LimitBytes, Action: effective.Quota.Action})
		} else {
			response, err := applier.traffic.SetQuota(ctx, &models.TrafficQuotaRequest{DeviceID: effective.DeviceID, Enabled: effective.Quota.Enabled, Period: effective.Quota.Period, LimitBytes: effective.Quota.LimitBytes, Action: effective.Quota.Action})
			if err != nil || response == nil || response.Result == nil || response.Result.Error != nil {
				if err != nil {
					failures = append(failures, err.Error())
				} else {
					failures = append(failures, "traffic quota apply failed")
				}
			}
		}
	}
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "; "))
	}
	return nil
}

func groupPolicyResponseError(response *models.DevicePolicyResponse) string {
	if response != nil && response.Result != nil && response.Result.Error != nil && response.Result.Error.Message != "" {
		return response.Result.Error.Message
	}
	return "device policy apply failed"
}

func groupNetworkPolicyResponseError(response *models.DeviceNetworkPolicyResponse) string {
	if response != nil && response.Result != nil && response.Result.Error != nil && response.Result.Error.Message != "" {
		return response.Result.Error.Message
	}
	return "internet path apply failed"
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type deviceNetworkPolicyWriter interface {
	Apply(context.Context, StaticAssignmentWriteInput, gatewayPolicyExecutionPlan) error
}

type DeviceNetworkPolicyModule struct {
	mu           sync.Mutex
	inventory    *DeviceInventoryModule
	devicePolicy *DevicePolicyModule
	gateway      *GatewayPolicyModule
	writer       deviceNetworkPolicyWriter
	effects      *PolicyEffectModule
	transactions *TaskTransactionJournal
}

func NewDeviceNetworkPolicyModule(inventory *DeviceInventoryModule, devicePolicy *DevicePolicyModule, gateway *GatewayPolicyModule, writer deviceNetworkPolicyWriter, effects ...*PolicyEffectModule) *DeviceNetworkPolicyModule {
	effectModule := NewPolicyEffectModule(newMemoryPolicyEffectStore(), &dnsmasqLeaseObserver{path: "/tmp/dhcp.leases"}, time.Now)
	if len(effects) > 0 && effects[0] != nil {
		effectModule = effects[0]
	}
	return &DeviceNetworkPolicyModule{inventory: inventory, devicePolicy: devicePolicy, gateway: gateway, writer: writer, effects: effectModule, transactions: newMemoryTaskTransactionJournal()}
}

func NewDefaultDeviceNetworkPolicyModule(inventory *DeviceInventoryModule, devicePolicy *DevicePolicyModule, gateway *GatewayPolicyModule) *DeviceNetworkPolicyModule {
	return NewDeviceNetworkPolicyModule(inventory, devicePolicy, gateway, &defaultDeviceNetworkPolicyWriter{}, NewDefaultPolicyEffectModule())
}

func (module *DeviceNetworkPolicyModule) Get(ctx context.Context, deviceID string) (*models.DeviceNetworkPolicyResponse, error) {
	if module == nil || module.devicePolicy == nil || module.gateway == nil {
		return nil, errors.New("device network policy module is unavailable")
	}
	policyResponse, err := module.devicePolicy.Get(ctx, deviceID)
	if err != nil {
		return nil, err
	}
	if policyResponse.Result == nil || policyResponse.Result.Error != nil || policyResponse.Result.Policy == nil {
		return deviceNetworkPolicyFailure(policyResponse.Result.Error), nil
	}
	gatewayState, currentTarget, err := module.currentTarget(ctx, policyResponse.Result.Policy.MAC)
	if err != nil {
		return nil, err
	}
	index := buildGatewayTargetIndex(gatewayState)
	countGatewayReferences(index, gatewayState.Hosts)
	targets := sortedGatewayTargets(index)
	static := publicAddressPolicy(policyResponse.Result.Policy.Static)
	effect := module.effects.Resolve(ctx, deviceID, policyResponse.Result.Policy.MAC, currentTarget, gatewayState.Version, static)
	if gatewayState.DHCP != nil && gatewayState.DHCP.DhcpIgnore {
		effect.Observed = &models.ObservedNetworkEffect{State: "failed", Reason: "dhcp_disabled"}
		effect.NeedsAttention = true
		effect.AttentionReason = "dhcp_authority_unavailable"
	}
	return &models.DeviceNetworkPolicyResponse{Result: &models.DeviceNetworkPolicyResult{Policy: &models.DeviceNetworkPolicy{
		DeviceID: deviceID, Static: static,
		Path:    &models.DeviceInternetPath{TargetID: currentTarget, Effect: effect},
		Targets: targets, Version: gatewayState.Version,
	}}}, nil
}

func (module *DeviceNetworkPolicyModule) Plan(ctx context.Context, request *models.DeviceNetworkPolicyApplyRequest) (*models.DeviceNetworkPolicyPlanResponse, error) {
	result := &models.DeviceNetworkPolicyPlanResult{
		Changes: []*models.PolicyPlanChange{}, ReloadServices: []string{}, RecoveryAction: "restore_task_snapshot",
	}
	if request == nil || request.DeviceID == "" || request.Static == nil || request.TargetID == "" {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "deviceId, static and targetId are required"}
		return &models.DeviceNetworkPolicyPlanResponse{Result: result}, nil
	}
	result.DeviceID = request.DeviceID
	currentResponse, err := module.devicePolicy.Get(ctx, request.DeviceID)
	if err != nil {
		return nil, err
	}
	if currentResponse.Result == nil || currentResponse.Result.Error != nil || currentResponse.Result.Policy == nil {
		result.Error = currentResponse.Result.Error
		return &models.DeviceNetworkPolicyPlanResponse{Result: result}, nil
	}
	current := currentResponse.Result.Policy
	_, previousTarget, err := module.currentTarget(ctx, current.MAC)
	if err != nil {
		return nil, err
	}
	routePlan, err := module.gateway.plan(ctx, &models.GatewayAssignmentRequest{Action: "assign", DeviceID: request.DeviceID, TargetID: request.TargetID})
	if err != nil {
		return nil, err
	}
	result.Version, result.RollbackPoint = routePlan.State.Version, routePlan.State.Version
	result.Current = &models.DesiredNetworkPolicy{TargetID: previousTarget, Static: publicAddressPolicy(current.Static)}
	result.Desired = &models.DesiredNetworkPolicy{TargetID: request.TargetID, Static: request.Static}
	if routePlan.Public.Error != nil || !routePlan.Public.CanApply {
		result.Error = routePlan.Public.Error
		return &models.DeviceNetworkPolicyPlanResponse{Result: result}, nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != routePlan.State.Version {
		result.Error = &models.DevicePolicyError{Code: "conflict", Message: "network configuration changed; plan again"}
		return &models.DeviceNetworkPolicyPlanResponse{Result: result}, nil
	}
	input := StaticAssignmentWriteInput{Action: "add", AssignedMAC: current.MAC, AssignedIP: request.Static.AssignedIP, BindIP: request.Static.BindIP, Hostname: request.Static.Hostname}
	if !request.Static.Enabled {
		input.AssignedIP, input.Hostname, input.BindIP = "", "", false
	}
	input, normalizeErr := normalizeStaticAssignmentInput(input)
	if normalizeErr != nil {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: normalizeErr.Error()}
		return &models.DeviceNetworkPolicyPlanResponse{Result: result}, nil
	}
	if conflict := networkPolicyStaticIPConflict(current.MAC, input, module.devicePolicy, ctx); conflict != nil {
		result.Error = conflict
		return &models.DeviceNetworkPolicyPlanResponse{Result: result}, nil
	}
	if !deviceStaticPolicyEqual(current.Static, request.Static) {
		result.Changes = append(result.Changes, &models.PolicyPlanChange{Kind: "address", Description: "change address reservation"})
	}
	if len(routePlan.Public.Changes) > 0 {
		result.Changes = append(result.Changes, &models.PolicyPlanChange{Kind: "internet_path", Description: "change device internet path"})
	}
	if len(result.Changes) > 0 {
		result.ReloadServices = []string{"dnsmasq"}
		result.RequiresRenewal = true
	}
	result.CanApply = true
	return &models.DeviceNetworkPolicyPlanResponse{Result: result}, nil
}

func (module *DeviceNetworkPolicyModule) Apply(ctx context.Context, request *models.DeviceNetworkPolicyApplyRequest) (*models.DeviceNetworkPolicyResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	if request == nil || request.DeviceID == "" || request.Static == nil || request.TargetID == "" {
		key := ""
		if request != nil {
			key = request.IdempotencyKey
		}
		return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: "validation_failed", Message: "deviceId, static and targetId are required"}), rejectedTaskTransaction("network", key)), nil
	}
	currentResponse, err := module.devicePolicy.Get(ctx, request.DeviceID)
	if err != nil {
		return nil, err
	}
	if currentResponse.Result == nil || currentResponse.Result.Error != nil || currentResponse.Result.Policy == nil {
		return deviceNetworkPolicyFailure(currentResponse.Result.Error), nil
	}
	current := currentResponse.Result.Policy
	_, previousTarget, targetErr := module.currentTarget(ctx, current.MAC)
	if targetErr != nil {
		return nil, targetErr
	}
	routeRequest := &models.GatewayAssignmentRequest{
		Action: "assign", DeviceID: request.DeviceID, TargetID: request.TargetID, ExpectedVersion: request.ExpectedVersion,
	}
	routePlan, err := module.gateway.plan(ctx, routeRequest)
	if err != nil {
		return nil, err
	}
	if routePlan.Public.Error != nil || !routePlan.Public.CanApply {
		return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(routePlan.Public.Error), rejectedTaskTransaction("network", request.IdempotencyKey)), nil
	}
	if routePlan.State.DHCP != nil && routePlan.State.DHCP.DhcpIgnore {
		return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: "validation_failed", Message: "DHCP is disabled; enable it before changing address or internet path"}), rejectedTaskTransaction("network", request.IdempotencyKey)), nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != routePlan.State.Version {
		return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: "conflict", Message: "network configuration changed; reload and try again"}), rejectedTaskTransaction("network", request.IdempotencyKey)), nil
	}
	input := StaticAssignmentWriteInput{
		Action: "add", AssignedMAC: current.MAC, AssignedIP: request.Static.AssignedIP,
		BindIP: request.Static.BindIP, Hostname: request.Static.Hostname,
	}
	if !request.Static.Enabled {
		input.AssignedIP, input.Hostname, input.BindIP = "", "", false
	}
	input, err = normalizeStaticAssignmentInput(input)
	if err != nil {
		return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}), rejectedTaskTransaction("network", request.IdempotencyKey)), nil
	}
	if conflict := networkPolicyStaticIPConflict(current.MAC, input, module.devicePolicy, ctx); conflict != nil {
		return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(conflict), rejectedTaskTransaction("network", request.IdempotencyKey)), nil
	}
	fingerprint, _ := json.Marshal(request)
	begin, beginErr := module.transactions.Begin(ctx, "network", request.IdempotencyKey, string(fingerprint))
	if beginErr != nil {
		return deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: "transaction_unavailable", Message: "could not create transaction record"}), nil
	}
	transaction := begin.Transaction
	if begin.Conflict {
		return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: "conflict", Message: "idempotency key was already used for a different change"}), transaction), nil
	}
	if begin.Replay {
		if transaction.Status != "committed" && transaction.Status != "unchanged" {
			return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: transaction.Status, Message: "previous transaction requires review"}), transaction), nil
		}
		response, replayErr := module.Get(ctx, request.DeviceID)
		if replayErr != nil {
			return nil, replayErr
		}
		return attachDeviceNetworkTransaction(response, transaction), nil
	}
	staticChanged := !deviceStaticPolicyEqual(current.Static, request.Static)
	routeChanged := len(routePlan.Public.Changes) > 0
	if !staticChanged && !routeChanged {
		_ = module.transactions.Advance(ctx, transaction, "verify", "unchanged", "")
		response, getErr := module.Get(ctx, request.DeviceID)
		return attachDeviceNetworkTransaction(response, transaction), getErr
	}
	if module.writer == nil {
		_ = module.transactions.Advance(ctx, transaction, "validate", "failed", "retry")
		return attachDeviceNetworkTransaction(deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: "apply_failed", Message: "network policy writer is unavailable"}), transaction), nil
	}
	_ = module.transactions.Advance(ctx, transaction, "apply", "in_progress", "restore_task_snapshot")
	if err := module.writer.Apply(ctx, input, routePlan); err != nil {
		_ = module.effects.RecordFailure(ctx, request.DeviceID, request.TargetID, previousTarget, routePlan.State.Version, request.Static, "server_apply_failed")
		status, stage, recovery := "rolled_back", "rollback", "retry"
		var failure *networkPolicyApplyError
		if errors.As(err, &failure) {
			status, stage = failure.Status, failure.Stage
			if status == "recovery_required" {
				recovery = "restore_task_snapshot"
			}
		}
		_ = module.transactions.Advance(ctx, transaction, stage, status, recovery)
		response, getErr := module.Get(ctx, request.DeviceID)
		if getErr != nil {
			return nil, getErr
		}
		response.Result.Error = &models.DevicePolicyError{Code: status, Message: err.Error()}
		return attachDeviceNetworkTransaction(response, transaction), nil
	}
	_ = module.transactions.Advance(ctx, transaction, "verify", "in_progress", "rebuild_observation_record")
	response, err := module.Get(ctx, request.DeviceID)
	if response != nil && response.Result != nil && response.Result.Policy != nil {
		if recordErr := module.effects.RecordApplied(ctx, request.DeviceID, request.TargetID, response.Result.Policy.Version, request.Static); recordErr != nil {
			_ = module.transactions.Advance(ctx, transaction, "recover", "recovery_required", "rebuild_observation_record")
			response.Result.Error = &models.DevicePolicyError{Code: "effect_record_failed", Message: "configuration was applied but its observation state could not be saved"}
			return attachDeviceNetworkTransaction(response, transaction), nil
		}
		response, err = module.Get(ctx, request.DeviceID)
		response.Result.Changed = true
	}
	_ = module.transactions.Advance(ctx, transaction, "verify", "committed", "")
	return attachDeviceNetworkTransaction(response, transaction), err
}

func (module *DeviceNetworkPolicyModule) currentTarget(ctx context.Context, mac string) (gatewayPolicySnapshot, string, error) {
	state, err := module.gateway.store.ReadState(ctx)
	if err != nil {
		return gatewayPolicySnapshot{}, "", err
	}
	index := buildGatewayTargetIndex(state)
	targetID := "default"
	for _, host := range state.Hosts {
		if normalizeInventoryMAC(host.MAC) == normalizeInventoryMAC(mac) {
			targetID = index.targetIDForTag(host.TagName)
			break
		}
	}
	return state, targetID, nil
}

func sortedGatewayTargets(index gatewayTargetIndex) []*models.GatewayTarget {
	targets := make([]*models.GatewayTarget, 0, len(index.targets))
	for _, descriptor := range index.targets {
		targets = append(targets, descriptor.Public)
	}
	sort.Slice(targets, func(i, j int) bool {
		left, right := gatewayTargetRank(targets[i].Kind), gatewayTargetRank(targets[j].Kind)
		if left != right {
			return left < right
		}
		return targets[i].ID < targets[j].ID
	})
	return targets
}

func publicAddressPolicy(value *models.DeviceStaticPolicy) *models.DeviceAddressPolicy {
	if value == nil {
		return &models.DeviceAddressPolicy{}
	}
	enabled := value.BindIP || strings.TrimSpace(value.AssignedIP) != "" || strings.TrimSpace(value.Hostname) != ""
	return &models.DeviceAddressPolicy{
		Enabled: enabled, AssignedIP: value.AssignedIP, BindIP: value.BindIP, Hostname: value.Hostname,
	}
}

func deviceStaticPolicyEqual(left *models.DeviceStaticPolicy, right *models.DeviceAddressPolicy) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	publicLeft := publicAddressPolicy(left)
	if !publicLeft.Enabled && !right.Enabled {
		return true
	}
	return publicLeft.Enabled == right.Enabled && publicLeft.AssignedIP == strings.TrimSpace(right.AssignedIP) &&
		publicLeft.BindIP == right.BindIP && publicLeft.Hostname == strings.ToLower(strings.TrimSpace(right.Hostname))
}

func networkPolicyStaticIPConflict(mac string, input StaticAssignmentWriteInput, policy *DevicePolicyModule, ctx context.Context) *models.DevicePolicyError {
	if !input.BindIP || input.AssignedIP == "" || policy == nil {
		return nil
	}
	rules, err := policy.ListRules(ctx)
	if err != nil || rules == nil || rules.Result == nil {
		return nil
	}
	for _, rule := range rules.Result.Static {
		if rule == nil || !rule.BindIP || strings.TrimSpace(rule.AssignedIP) != input.AssignedIP {
			continue
		}
		if normalizeInventoryMAC(rule.AssignedMac) != normalizeInventoryMAC(mac) {
			return &models.DevicePolicyError{Code: "address_conflict", Message: "IPv4 address is already reserved for another device"}
		}
	}
	return nil
}

func deviceNetworkPolicyFailure(err *models.DevicePolicyError) *models.DeviceNetworkPolicyResponse {
	if err == nil {
		err = &models.DevicePolicyError{Code: "not_found", Message: "device network policy is unavailable"}
	}
	return &models.DeviceNetworkPolicyResponse{Result: &models.DeviceNetworkPolicyResult{Error: err}}
}

func attachDeviceNetworkTransaction(response *models.DeviceNetworkPolicyResponse, transaction *models.TaskTransaction) *models.DeviceNetworkPolicyResponse {
	if response != nil && response.Result != nil {
		response.Result.Transaction = transaction
	}
	return response
}

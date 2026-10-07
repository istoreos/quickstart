package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
	"sync"

	"github.com/istoreos/quickstart/backend/models"
)

type devicePolicyBackup interface{}

type devicePolicyStore interface {
	Get(context.Context, string) (*models.DevicePolicy, error)
	ListRules(context.Context) (*models.DevicePolicyRulesResult, error)
	Backup(context.Context, string, ...*models.DevicePolicy) (devicePolicyBackup, error)
	Apply(context.Context, *models.DevicePolicyApplyRequest, *models.DevicePolicy) error
	Restore(context.Context, devicePolicyBackup) error
}

type DevicePolicyModule struct {
	mu           sync.Mutex
	store        devicePolicyStore
	transactions *TaskTransactionJournal
}

type PolicyError struct {
	Code    string
	Message string
}

func (err *PolicyError) Error() string { return err.Message }

func NewDevicePolicyModule(inventory *DeviceInventoryModule) *DevicePolicyModule {
	return &DevicePolicyModule{store: newSystemDevicePolicyStore(inventory), transactions: newMemoryTaskTransactionJournal()}
}

func NewDevicePolicyModuleWithRateLimit(inventory *DeviceInventoryModule, gateway *GatewayPolicyModule) *DevicePolicyModule {
	return &DevicePolicyModule{store: newSystemDevicePolicyStore(inventory, NewDefaultRateLimitModule(gateway)), transactions: newMemoryTaskTransactionJournal()}
}

func newDevicePolicyModuleForTest(store devicePolicyStore) *DevicePolicyModule {
	return &DevicePolicyModule{store: store, transactions: newMemoryTaskTransactionJournal()}
}

func (module *DevicePolicyModule) Get(ctx context.Context, deviceID string) (*models.DevicePolicyResponse, error) {
	if module == nil || module.store == nil {
		return nil, errors.New("device policy module is unavailable")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return devicePolicyFailure("validation_failed", "deviceId is required"), nil
	}
	policy, err := module.store.Get(ctx, deviceID)
	if err != nil {
		return devicePolicyFailure("validation_failed", err.Error()), nil
	}
	return devicePolicySuccess(policy, false), nil
}

func (module *DevicePolicyModule) ListRules(ctx context.Context) (*models.DevicePolicyRulesResponse, error) {
	if module == nil || module.store == nil {
		return nil, errors.New("device policy module is unavailable")
	}
	rules, err := module.store.ListRules(ctx)
	if err != nil {
		return nil, err
	}
	return &models.DevicePolicyRulesResponse{Result: rules}, nil
}

func (module *DevicePolicyModule) Plan(ctx context.Context, request *models.DevicePolicyApplyRequest) (*models.DeviceRestrictionPlanResponse, error) {
	result := &models.DeviceRestrictionPlanResult{Changes: []*models.PolicyPlanChange{}, ReloadServices: []string{}, RecoveryAction: "restore_task_snapshot"}
	if policyErr := validateDevicePolicyRequest(request); policyErr != nil {
		result.Error = &models.DevicePolicyError{Code: policyErr.Code, Message: policyErr.Message}
		return &models.DeviceRestrictionPlanResponse{Result: result}, nil
	}
	result.DeviceID, result.Kind = request.DeviceID, request.Kind
	current, err := module.store.Get(ctx, request.DeviceID)
	if err != nil {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}
		return &models.DeviceRestrictionPlanResponse{Result: result}, nil
	}
	result.Version = devicePolicyVersion(current)
	result.RollbackPoint = result.Version
	if capability := current.Capabilities[request.Kind]; capability == nil || capability.State != "available" {
		code, reason := "validation_failed", "this policy is unavailable"
		if capability != nil && capability.Reason != "" {
			reason = capability.Reason
		}
		if capability != nil && capability.State == "not_installed" {
			code = "dependency_not_installed"
		} else if capability != nil && capability.State == "error" {
			code = "apply_failed"
		}
		result.Error = &models.DevicePolicyError{Code: code, Message: reason}
		return &models.DeviceRestrictionPlanResponse{Result: result}, nil
	}
	if request.Kind == "speed" && request.Speed != nil && request.Speed.Enabled && current.RateLimit != nil && !current.RateLimit.CanApply {
		result.Error = &models.DevicePolicyError{Code: current.RateLimit.Reason, Message: rateLimitReasonMessage(current.RateLimit.Reason)}
		return &models.DeviceRestrictionPlanResponse{Result: result}, nil
	}
	if conflict := module.validateConflict(ctx, request, current); conflict != nil {
		result.Error = &models.DevicePolicyError{Code: conflict.Code, Message: conflict.Message}
		return &models.DeviceRestrictionPlanResponse{Result: result}, nil
	}
	switch request.Kind {
	case "speed":
		currentValue, desiredValue := *current.Speed, *request.Speed
		result.CurrentSpeed, result.DesiredSpeed = &currentValue, &desiredValue
		if current.RateLimit == nil || current.RateLimit.Provider == "eqos" {
			result.ReloadServices = []string{"eqos"}
		}
	case "access":
		currentValue, desiredValue := *current.Access, *request.Access
		result.CurrentAccess, result.DesiredAccess = &currentValue, &desiredValue
		result.ReloadServices = []string{"firewall"}
	case "static":
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "address reservations are planned through DeviceNetworkPolicy"}
		return &models.DeviceRestrictionPlanResponse{Result: result}, nil
	}
	if !devicePolicyNoOp(request, current) {
		result.Changes = append(result.Changes, &models.PolicyPlanChange{Kind: request.Kind, Description: "change device usage restriction"})
	}
	result.CanApply = true
	return &models.DeviceRestrictionPlanResponse{Result: result}, nil
}

func (module *DevicePolicyModule) Apply(ctx context.Context, request *models.DevicePolicyApplyRequest) (*models.DevicePolicyResponse, error) {
	if module == nil || module.store == nil {
		return nil, errors.New("device policy module is unavailable")
	}
	module.mu.Lock()
	defer module.mu.Unlock()
	if policyErr := validateDevicePolicyRequest(request); policyErr != nil {
		key := ""
		if request != nil {
			key = request.IdempotencyKey
		}
		return attachDevicePolicyTransaction(devicePolicyFailure(policyErr.Code, policyErr.Message), rejectedTaskTransaction(devicePolicyTask(request), key)), nil
	}
	current, err := module.store.Get(ctx, request.DeviceID)
	if err != nil {
		return attachDevicePolicyTransaction(devicePolicyFailure("validation_failed", err.Error()), rejectedTaskTransaction(devicePolicyTask(request), request.IdempotencyKey)), nil
	}
	currentVersion := devicePolicyVersion(current)
	if request.ExpectedVersion != "" && request.ExpectedVersion != currentVersion {
		return attachDevicePolicyTransaction(devicePolicyFailure("conflict", "device restrictions changed; plan again"), rejectedTaskTransaction(devicePolicyTask(request), request.IdempotencyKey)), nil
	}
	if capability := current.Capabilities[request.Kind]; capability == nil || capability.State != "available" {
		reason := "this policy is unavailable"
		code := "validation_failed"
		if capability != nil && capability.Reason != "" {
			reason = capability.Reason
		}
		if capability != nil && capability.State == "not_installed" {
			code = "dependency_not_installed"
		} else if capability != nil && capability.State == "error" {
			code = "apply_failed"
		}
		return attachDevicePolicyTransaction(devicePolicyFailure(code, reason), rejectedTaskTransaction(devicePolicyTask(request), request.IdempotencyKey)), nil
	}
	if request.Kind == "speed" && request.Speed != nil && request.Speed.Enabled && current.RateLimit != nil && !current.RateLimit.CanApply {
		code := current.RateLimit.Reason
		return attachDevicePolicyTransaction(devicePolicyFailure(code, rateLimitReasonMessage(code)), rejectedTaskTransaction(devicePolicyTask(request), request.IdempotencyKey)), nil
	}
	if conflict := module.validateConflict(ctx, request, current); conflict != nil {
		return attachDevicePolicyTransaction(devicePolicyFailure(conflict.Code, conflict.Message), rejectedTaskTransaction(devicePolicyTask(request), request.IdempotencyKey)), nil
	}
	fingerprint, _ := json.Marshal(request)
	task := devicePolicyTask(request)
	begin, err := module.transactions.Begin(ctx, task, request.IdempotencyKey, string(fingerprint))
	if err != nil {
		return devicePolicyFailure("transaction_unavailable", "could not create transaction record"), nil
	}
	transaction := begin.Transaction
	if begin.Conflict {
		return attachDevicePolicyTransaction(devicePolicyFailure("conflict", "idempotency key was already used for a different change"), transaction), nil
	}
	if begin.Replay {
		policy, readErr := module.store.Get(ctx, request.DeviceID)
		if readErr != nil {
			return attachDevicePolicyTransaction(devicePolicyFailure("apply_failed", readErr.Error()), transaction), nil
		}
		if transaction.Status == "committed" || transaction.Status == "unchanged" {
			return attachDevicePolicyTransaction(devicePolicySuccess(policy, false), transaction), nil
		}
		return attachDevicePolicyTransaction(devicePolicyFailure(transaction.Status, "previous transaction requires review"), transaction), nil
	}
	if devicePolicyNoOp(request, current) {
		_ = module.transactions.Advance(ctx, transaction, "verify", "unchanged", "")
		return attachDevicePolicyTransaction(devicePolicySuccess(current, false), transaction), nil
	}

	_ = module.transactions.Advance(ctx, transaction, "snapshot", "in_progress", "")
	backup, err := module.store.Backup(ctx, request.Kind, current)
	if err != nil {
		_ = module.transactions.Advance(ctx, transaction, "snapshot", "failed", "retry")
		return attachDevicePolicyTransaction(devicePolicyFailure("apply_failed", "could not create a rollback snapshot"), transaction), nil
	}
	_ = module.transactions.Advance(ctx, transaction, "apply", "in_progress", "restore_task_snapshot")
	if err := module.store.Apply(ctx, request, current); err != nil {
		restoreErr := module.store.Restore(ctx, backup)
		if restoreErr != nil {
			_ = module.transactions.Advance(ctx, transaction, "recover", "recovery_required", "restore_task_snapshot")
			return attachDevicePolicyTransaction(devicePolicyFailure("recovery_required", "policy apply and rollback both failed"), transaction), nil
		}
		_ = module.transactions.Advance(ctx, transaction, "rollback", "rolled_back", "retry")
		return attachDevicePolicyTransaction(devicePolicyFailure("rolled_back", "policy apply failed; original configuration was restored"), transaction), nil
	}
	_ = module.transactions.Advance(ctx, transaction, "verify", "in_progress", "restore_task_snapshot")
	updated, err := module.store.Get(ctx, request.DeviceID)
	if err != nil {
		restoreErr := module.store.Restore(ctx, backup)
		if restoreErr != nil {
			_ = module.transactions.Advance(ctx, transaction, "recover", "recovery_required", "restore_task_snapshot")
			return attachDevicePolicyTransaction(devicePolicyFailure("recovery_required", "verification and rollback both failed"), transaction), nil
		}
		_ = module.transactions.Advance(ctx, transaction, "rollback", "rolled_back", "retry")
		return attachDevicePolicyTransaction(devicePolicyFailure("rolled_back", "verification failed; original configuration was restored"), transaction), nil
	}
	_ = module.transactions.Advance(ctx, transaction, "verify", "committed", "")
	return attachDevicePolicyTransaction(devicePolicySuccess(updated, true), transaction), nil
}

func rateLimitReasonMessage(reason string) string {
	switch reason {
	case "address_reservation_required":
		return "请先为设备预留当前 IPv4 地址，再启用限速"
	case "execution_node_unavailable":
		return "设备当前不通过本机上网，请在实际执行上网的网关上设置限速"
	default:
		return "当前无法安全应用设备限速"
	}
}

func devicePolicyTask(request *models.DevicePolicyApplyRequest) string {
	if request != nil && request.Kind == "static" {
		return "network"
	}
	return "restrictions"
}

func validateDevicePolicyRequest(request *models.DevicePolicyApplyRequest) *PolicyError {
	if request == nil || strings.TrimSpace(request.DeviceID) == "" {
		return &PolicyError{Code: "validation_failed", Message: "deviceId is required"}
	}
	switch request.Kind {
	case "static":
		if request.Static == nil {
			return &PolicyError{Code: "validation_failed", Message: "static policy is required"}
		}
		if request.Static.Enabled {
			address, err := netip.ParseAddr(strings.TrimSpace(request.Static.AssignedIP))
			if err != nil || !address.Is4() {
				return &PolicyError{Code: "validation_failed", Message: "a valid IPv4 address is required"}
			}
			if request.Static.Hostname != "" && !dhcpHostnamePattern.MatchString(strings.TrimSpace(request.Static.Hostname)) {
				return &PolicyError{Code: "validation_failed", Message: "DHCP hostname must be a 1-63 character ASCII label using only letters, numbers, and interior hyphens"}
			}
		}
	case "speed":
		if request.Speed == nil {
			return &PolicyError{Code: "validation_failed", Message: "speed policy is required"}
		}
		if request.Speed.Enabled && (request.Speed.UploadSpeed <= 0 || request.Speed.DownloadSpeed <= 0) {
			return &PolicyError{Code: "validation_failed", Message: "upload and download limits must be greater than zero"}
		}
	case "access":
		if request.Access == nil {
			return &PolicyError{Code: "validation_failed", Message: "access policy is required"}
		}
	default:
		return &PolicyError{Code: "validation_failed", Message: "unknown policy kind"}
	}
	return nil
}

func (module *DevicePolicyModule) validateConflict(ctx context.Context, request *models.DevicePolicyApplyRequest, current *models.DevicePolicy) *PolicyError {
	if request.Kind != "static" || request.Static == nil || !request.Static.Enabled {
		return nil
	}
	rules, err := module.store.ListRules(ctx)
	if err != nil {
		return &PolicyError{Code: "apply_failed", Message: err.Error()}
	}
	for _, rule := range rules.Static {
		if rule == nil || rule.AssignedIP != request.Static.AssignedIP {
			continue
		}
		if normalizeInventoryMAC(rule.AssignedMac) != normalizeInventoryMAC(current.MAC) {
			return &PolicyError{Code: "conflict", Message: "the IPv4 address is already assigned to another device"}
		}
	}
	return nil
}

func devicePolicyNoOp(request *models.DevicePolicyApplyRequest, current *models.DevicePolicy) bool {
	switch request.Kind {
	case "static":
		return request.Static.Enabled == current.Static.Enabled &&
			(!request.Static.Enabled || *request.Static == *current.Static)
	case "speed":
		return request.Speed.Enabled == current.Speed.Enabled &&
			(!request.Speed.Enabled || *request.Speed == *current.Speed)
	case "access":
		return request.Access.NetworkAccess == current.Access.NetworkAccess
	}
	return false
}

func devicePolicySuccess(policy *models.DevicePolicy, changed bool) *models.DevicePolicyResponse {
	if policy != nil {
		policy.Version = devicePolicyVersion(policy)
	}
	return &models.DevicePolicyResponse{Result: &models.DevicePolicyResult{Policy: policy, Changed: changed}}
}

func devicePolicyVersion(policy *models.DevicePolicy) string {
	if policy == nil {
		return ""
	}
	// Optimistic concurrency protects configurable policy, not volatile runtime
	// observations such as ObservedAt, traffic state or capability hints. Those
	// values can legitimately change between Plan and Apply without a user edit.
	stable := struct {
		DeviceID string                     `json:"deviceId"`
		MAC      string                     `json:"mac"`
		Static   *models.DeviceStaticPolicy `json:"static"`
		Speed    *models.DeviceSpeedPolicy  `json:"speed"`
		Access   *models.DeviceAccessPolicy `json:"access"`
	}{
		DeviceID: policy.DeviceID,
		MAC:      policy.MAC,
		Static:   policy.Static,
		Speed:    policy.Speed,
		Access:   policy.Access,
	}
	raw, _ := json.Marshal(stable)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func attachDevicePolicyTransaction(response *models.DevicePolicyResponse, transaction *models.TaskTransaction) *models.DevicePolicyResponse {
	if response != nil && response.Result != nil {
		response.Result.Transaction = transaction
	}
	return response
}

func devicePolicyFailure(code, message string) *models.DevicePolicyResponse {
	return &models.DevicePolicyResponse{
		JSONResponse: models.JSONResponse{Success: -1, Scope: "device_policy", Error: message},
		Result:       &models.DevicePolicyResult{Error: &models.DevicePolicyError{Code: code, Message: message}},
	}
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"sync"

	"github.com/istoreos/quickstart/backend/models"
)

const devicePolicyIdempotencyLimit = 512

type devicePolicyBackup interface{}

type devicePolicyStore interface {
	Get(context.Context, string) (*models.DevicePolicy, error)
	ListRules(context.Context) (*models.DevicePolicyRulesResult, error)
	Backup(context.Context, string) (devicePolicyBackup, error)
	Apply(context.Context, *models.DevicePolicyApplyRequest, *models.DevicePolicy) error
	Restore(context.Context, devicePolicyBackup) error
}

type devicePolicyIdempotencyRecord struct {
	Fingerprint string
}

type DevicePolicyModule struct {
	mu          sync.Mutex
	store       devicePolicyStore
	idempotency map[string]devicePolicyIdempotencyRecord
	order       []string
}

type PolicyError struct {
	Code    string
	Message string
}

func (err *PolicyError) Error() string { return err.Message }

func NewDevicePolicyModule(inventory *DeviceInventoryModule) *DevicePolicyModule {
	return &DevicePolicyModule{store: newSystemDevicePolicyStore(inventory), idempotency: map[string]devicePolicyIdempotencyRecord{}}
}

func newDevicePolicyModuleForTest(store devicePolicyStore) *DevicePolicyModule {
	return &DevicePolicyModule{store: store, idempotency: map[string]devicePolicyIdempotencyRecord{}}
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

func (module *DevicePolicyModule) Apply(ctx context.Context, request *models.DevicePolicyApplyRequest) (*models.DevicePolicyResponse, error) {
	if module == nil || module.store == nil {
		return nil, errors.New("device policy module is unavailable")
	}
	module.mu.Lock()
	defer module.mu.Unlock()
	if policyErr := validateDevicePolicyRequest(request); policyErr != nil {
		return devicePolicyFailure(policyErr.Code, policyErr.Message), nil
	}
	fingerprint, _ := json.Marshal(request)
	if request.IdempotencyKey != "" {
		if previous, ok := module.idempotency[request.IdempotencyKey]; ok {
			if previous.Fingerprint != string(fingerprint) {
				return devicePolicyFailure("conflict", "idempotency key was already used for a different change"), nil
			}
			policy, err := module.store.Get(ctx, request.DeviceID)
			if err != nil {
				return devicePolicyFailure("apply_failed", err.Error()), nil
			}
			return devicePolicySuccess(policy, false), nil
		}
	}

	current, err := module.store.Get(ctx, request.DeviceID)
	if err != nil {
		return devicePolicyFailure("validation_failed", err.Error()), nil
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
		return devicePolicyFailure(code, reason), nil
	}
	if conflict := module.validateConflict(ctx, request, current); conflict != nil {
		return devicePolicyFailure(conflict.Code, conflict.Message), nil
	}
	if devicePolicyNoOp(request, current) {
		module.rememberIdempotency(request.IdempotencyKey, string(fingerprint))
		return devicePolicySuccess(current, false), nil
	}

	backup, err := module.store.Backup(ctx, request.Kind)
	if err != nil {
		return devicePolicyFailure("apply_failed", "could not create a rollback snapshot"), nil
	}
	if err := module.store.Apply(ctx, request, current); err != nil {
		restoreErr := module.store.Restore(ctx, backup)
		message := err.Error()
		if restoreErr != nil {
			message = fmt.Sprintf("%s; rollback failed: %v", message, restoreErr)
		}
		return devicePolicyFailure("apply_failed", message), nil
	}
	updated, err := module.store.Get(ctx, request.DeviceID)
	if err != nil {
		restoreErr := module.store.Restore(ctx, backup)
		message := "could not verify the applied policy"
		if restoreErr != nil {
			message = fmt.Sprintf("%s; rollback failed: %v", message, restoreErr)
		}
		return devicePolicyFailure("apply_failed", message), nil
	}
	module.rememberIdempotency(request.IdempotencyKey, string(fingerprint))
	return devicePolicySuccess(updated, true), nil
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

func (module *DevicePolicyModule) rememberIdempotency(key, fingerprint string) {
	if key == "" {
		return
	}
	if _, exists := module.idempotency[key]; !exists {
		module.order = append(module.order, key)
	}
	module.idempotency[key] = devicePolicyIdempotencyRecord{Fingerprint: fingerprint}
	for len(module.order) > devicePolicyIdempotencyLimit {
		delete(module.idempotency, module.order[0])
		module.order = module.order[1:]
	}
}

func devicePolicySuccess(policy *models.DevicePolicy, changed bool) *models.DevicePolicyResponse {
	return &models.DevicePolicyResponse{Result: &models.DevicePolicyResult{Policy: policy, Changed: changed}}
}

func devicePolicyFailure(code, message string) *models.DevicePolicyResponse {
	return &models.DevicePolicyResponse{
		JSONResponse: models.JSONResponse{Success: -1, Scope: "device_policy", Error: message},
		Result:       &models.DevicePolicyResult{Error: &models.DevicePolicyError{Code: code, Message: message}},
	}
}

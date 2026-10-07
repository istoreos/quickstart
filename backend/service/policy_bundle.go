package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type PolicyBundleModule struct {
	groups  *DeviceGroupModule
	traffic *TrafficInsightsModule
	audit   *NetworkAuditModule
	now     func() time.Time
}

func NewPolicyBundleModule(groups *DeviceGroupModule, traffic *TrafficInsightsModule, audit *NetworkAuditModule) *PolicyBundleModule {
	return &PolicyBundleModule{groups: groups, traffic: traffic, audit: audit, now: time.Now}
}

func (module *AdvancedNetworkModule) Export(ctx context.Context) (*models.PolicyBundleResponse, error) {
	return module.bundles.Export(ctx)
}

func (module *AdvancedNetworkModule) ImportPlan(ctx context.Context, request *models.PolicyImportRequest) (*models.PolicyBundleResponse, error) {
	return module.bundles.ImportPlan(ctx, request)
}

func (module *AdvancedNetworkModule) ImportApply(ctx context.Context, request *models.PolicyImportRequest) (*models.PolicyBundleResponse, error) {
	return module.bundles.ImportApply(ctx, request)
}

func (module *PolicyBundleModule) Export(ctx context.Context) (*models.PolicyBundleResponse, error) {
	state, err := module.groups.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	module.traffic.mu.Lock()
	defer module.traffic.mu.Unlock()
	if err := module.traffic.load(); err != nil {
		return nil, err
	}
	quotas := map[string]*models.TrafficQuota{}
	for id, value := range module.traffic.document.Quotas {
		copyValue := *value
		quotas[id] = &copyValue
	}
	bundle := &models.PolicyBundle{SchemaVersion: 1, Scope: "device_groups_and_quotas", ExportedAt: module.now().UTC().Format(time.RFC3339), GlobalPolicy: state.GlobalPolicy, Groups: state.Groups, DevicePolicies: state.DevicePolicies, Quotas: quotas}
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Bundle: bundle}}, nil
}

func (module *PolicyBundleModule) ImportPlan(ctx context.Context, request *models.PolicyImportRequest) (*models.PolicyBundleResponse, error) {
	plan, policyErr := module.planImport(ctx, request)
	if policyErr != nil {
		return policyBundleFailure(policyErr.Code, policyErr.Message), nil
	}
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Plan: plan}}, nil
}

func (module *PolicyBundleModule) ImportApply(ctx context.Context, request *models.PolicyImportRequest) (*models.PolicyBundleResponse, error) {
	plan, policyErr := module.planImport(ctx, request)
	if policyErr != nil {
		return policyBundleFailure(policyErr.Code, policyErr.Message), nil
	}
	if request == nil || !request.Confirmed {
		return policyBundleFailure("confirmation_required", "apply requires a confirmed dry-run bundle"), nil
	}
	if request.PlanChecksum == "" || request.PlanChecksum != plan.Checksum {
		return policyBundleFailure("conflict", "bundle changed after dry-run; preview it again before applying"), nil
	}
	if request.ExpectedGroupVersion == "" || request.ExpectedGroupVersion != plan.GroupVersion {
		return policyBundleFailure("conflict", "device groups changed after dry-run; preview them again before applying"), nil
	}
	groupStore, ok := module.groups.store.(*jsonDeviceGroupStore)
	if !ok {
		return policyBundleFailure("unavailable", "policy import store is unavailable"), nil
	}
	module.groups.mu.Lock()
	defer module.groups.mu.Unlock()
	module.traffic.mu.Lock()
	defer module.traffic.mu.Unlock()
	oldGroup, oldGroupRaw, err := groupStore.readDocument()
	if err != nil {
		return nil, err
	}
	if err := module.traffic.load(); err != nil {
		return nil, err
	}
	oldTraffic := cloneTrafficInsightsDocument(module.traffic.document)
	nextGroup := deviceGroupDocument{SchemaVersion: 1, GlobalPolicy: request.Bundle.GlobalPolicy, Groups: request.Bundle.Groups, DevicePolicies: request.Bundle.DevicePolicies}
	nextTraffic := cloneTrafficInsightsDocument(module.traffic.document)
	nextTraffic.Quotas = map[string]*models.TrafficQuota{}
	nextTraffic.QuotaRuntime = map[string]trafficQuotaRuntime{}
	for id, value := range request.Bundle.Quotas {
		copyValue := *value
		nextTraffic.Quotas[id] = &copyValue
	}
	if reflect.DeepEqual(oldGroup, nextGroup) && reflect.DeepEqual(oldTraffic.Quotas, nextTraffic.Quotas) {
		return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Plan: plan, Changed: false}}, nil
	}
	updatedGroup, err := groupStore.Replace(nextGroup, request.ExpectedGroupVersion)
	if err != nil {
		return policyBundleFailure("apply_failed", err.Error()), nil
	}
	trafficSize, err := module.traffic.store.Save(&nextTraffic)
	if err != nil {
		_, rollbackErr := groupStore.Replace(oldGroup, "")
		if rollbackErr != nil {
			return policyBundleFailure("rollback_failed", err.Error()+"; rollback failed: "+rollbackErr.Error()), nil
		}
		module.traffic.document = oldTraffic
		return policyBundleFailure("apply_failed", err.Error()), nil
	}
	module.traffic.document = nextTraffic
	module.traffic.storageSize = trafficSize
	module.traffic.dirty = false
	previousGroup := documentDeviceGroupState(oldGroup, oldGroupRaw)
	module.groups.reconcileState(ctx, updatedGroup, &previousGroup, module.now())
	if module.audit != nil {
		module.audit.Record("", "policy_bundle", "policy_changed", "success", plan.Checksum)
	}
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Plan: plan, Changed: true}}, nil
}

func (module *PolicyBundleModule) planImport(ctx context.Context, request *models.PolicyImportRequest) (*models.PolicyImportPlan, *models.DevicePolicyError) {
	if request == nil || request.Bundle == nil {
		return nil, &models.DevicePolicyError{Code: "validation_failed", Message: "bundle is required"}
	}
	bundle := request.Bundle
	if bundle.SchemaVersion != 1 || bundle.Scope != "device_groups_and_quotas" {
		return nil, &models.DevicePolicyError{Code: "incompatible_version", Message: "unsupported policy bundle version or scope"}
	}
	if bundle.Groups == nil || bundle.DevicePolicies == nil || bundle.Quotas == nil {
		return nil, &models.DevicePolicyError{Code: "validation_failed", Message: "bundle collections are required"}
	}
	document := deviceGroupDocument{SchemaVersion: 1, GlobalPolicy: bundle.GlobalPolicy, Groups: bundle.Groups, DevicePolicies: bundle.DevicePolicies}
	if err := validateDeviceGroupDocument(document); err != nil {
		return nil, &models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}
	}
	if len(bundle.Quotas) > trafficInsightsQuotaLimit {
		return nil, &models.DevicePolicyError{Code: "capacity_exceeded", Message: "traffic quota limit reached"}
	}
	for id, quota := range bundle.Quotas {
		if quota == nil || id != quota.DeviceID {
			return nil, &models.DevicePolicyError{Code: "validation_failed", Message: "invalid quota record"}
		}
		if err := validateTrafficQuotaRequest(&models.TrafficQuotaRequest{DeviceID: quota.DeviceID, Enabled: quota.Enabled, Period: quota.Period, LimitBytes: quota.LimitBytes, Action: quota.Action}); err != nil {
			return nil, &models.DevicePolicyError{Code: "validation_failed", Message: err.Error()}
		}
	}
	current, err := module.groups.store.Read(ctx)
	if err != nil {
		return nil, &models.DevicePolicyError{Code: "read_failed", Message: err.Error()}
	}
	if request.ExpectedGroupVersion != "" && request.ExpectedGroupVersion != current.Version {
		return nil, &models.DevicePolicyError{Code: "conflict", Message: "device groups changed; export again before importing"}
	}
	raw, _ := json.Marshal(bundle)
	sum := sha256.Sum256(raw)
	warnings := []string{"Import replaces device groups and traffic quotas only; address, route, speed and firewall rules are unchanged."}
	sort.Strings(warnings)
	return &models.PolicyImportPlan{CanApply: true, GroupVersion: current.Version, GroupCount: len(bundle.Groups), DevicePolicyCount: len(bundle.DevicePolicies), QuotaCount: len(bundle.Quotas), Warnings: warnings, Checksum: hex.EncodeToString(sum[:])}, nil
}

func cloneTrafficInsightsDocument(source trafficInsightsDocument) trafficInsightsDocument {
	raw, _ := json.Marshal(source)
	var result trafficInsightsDocument
	_ = json.Unmarshal(raw, &result)
	normalizeTrafficInsightsDocument(&result)
	return result
}

func policyBundleFailure(code, message string) *models.PolicyBundleResponse {
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Error: &models.DevicePolicyError{Code: code, Message: message}}}
}

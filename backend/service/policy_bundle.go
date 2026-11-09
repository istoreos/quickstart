package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
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
	module = module.scoped(ctx)
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
	schema := 1
	if selectedLAN(ctx) != defaultLAN {
		schema = 2
	}
	bundle := &models.PolicyBundle{SchemaVersion: schema, NetworkID: selectedLAN(ctx), Scope: "device_groups_and_quotas", ExportedAt: module.now().UTC().Format(time.RFC3339), GlobalPolicy: state.GlobalPolicy, Groups: state.Groups, DevicePolicies: state.DevicePolicies, Quotas: quotas}
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
	locked, release, err := lockLANWrite(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	ctx = locked
	module = module.scoped(ctx)
	plan, policyErr := module.planImport(ctx, request)
	if policyErr != nil {
		return policyBundleFailure(policyErr.Code, policyErr.Message), nil
	}
	if !request.Confirmed {
		return policyBundleFailure("confirmation_required", "apply requires a confirmed dry-run bundle"), nil
	}
	if request.PlanChecksum == "" || request.PlanChecksum != plan.Checksum || request.ExpectedGroupVersion == "" || request.ExpectedGroupVersion != plan.GroupVersion {
		return policyBundleFailure("conflict", "network or policy changed after preview"), nil
	}
	store, ok := module.groups.store.(*jsonDeviceGroupStore)
	if !ok {
		return policyBundleFailure("unavailable", "policy import store is unavailable"), nil
	}
	store = store.forContext(ctx)
	if len(request.Bundle.Quotas) > 0 {
		if err := module.traffic.prepareQuotaSample(ctx); err != nil {
			return nil, err
		}
	}
	module.groups.mu.Lock()
	defer module.groups.mu.Unlock()
	oldGroup, raw, err := store.readDocument()
	if err != nil {
		return nil, err
	}
	module.traffic.mu.Lock()
	err = module.traffic.load()
	oldTraffic := cloneTrafficInsightsDocument(module.traffic.document)
	module.traffic.mu.Unlock()
	if err != nil {
		return nil, err
	}
	nextGroup := deviceGroupDocument{SchemaVersion: 1, GlobalPolicy: request.Bundle.GlobalPolicy, Groups: request.Bundle.Groups, DevicePolicies: request.Bundle.DevicePolicies}
	if reflect.DeepEqual(oldGroup, nextGroup) && reflect.DeepEqual(oldTraffic.Quotas, request.Bundle.Quotas) {
		return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{Plan: plan}}, nil
	}
	paths := []string{filepath.Join(deviceRestrictionConfigDir(), "dhcp"), filepath.Join(deviceRestrictionConfigDir(), "eqos"), filepath.Join(deviceRestrictionConfigDir(), "firewall"), immediateAccessIncludePath(deviceRestrictionConfigDir())}
	snapshots, err := networkRulesTakeSnapshots(paths)
	if err != nil {
		return nil, err
	}
	rollback := func(cause error) (*models.PolicyBundleResponse, error) {
		_, groupErr := store.Replace(oldGroup, "")
		quotaErr := module.traffic.RestoreQuotaBatch(ctx, oldTraffic)
		filesErr := networkRulesRestoreSnapshots(ctx, snapshots)
		if groupErr != nil || quotaErr != nil || filesErr != nil {
			return policyBundleFailure("recovery_required", "import and recovery failed; review the saved snapshot"), nil
		}
		return policyBundleFailure("rolled_back", cause.Error()), nil
	}
	updated, err := store.Replace(nextGroup, versionDeviceGroups(raw))
	if err != nil {
		return policyBundleFailure("conflict", err.Error()), nil
	}
	quotaRequests := make([]*models.TrafficQuotaRequest, 0, len(oldTraffic.Quotas)+len(request.Bundle.Quotas))
	for id := range oldTraffic.Quotas {
		if request.Bundle.Quotas[id] == nil {
			quotaRequests = append(quotaRequests, &models.TrafficQuotaRequest{DeviceID: id, Enabled: false})
		}
	}
	for _, quota := range request.Bundle.Quotas {
		quotaRequests = append(quotaRequests, &models.TrafficQuotaRequest{DeviceID: quota.DeviceID, Enabled: quota.Enabled, Period: quota.Period, LimitBytes: quota.LimitBytes, Action: quota.Action})
	}
	if _, err := module.traffic.SetQuotaBatch(ctx, quotaRequests); err != nil {
		return rollback(err)
	}
	previous := documentDeviceGroupState(oldGroup, raw)
	module.groups.reconcileState(ctx, updated, &previous, module.now())
	module.groups.batchMu.Lock()
	batch := module.groups.lastBatch
	module.groups.batchMu.Unlock()
	if batch != nil && batch.Failed > 0 {
		return rollback(errors.New("imported group policies could not be applied"))
	}
	if deviceGroupStateHasWork(updated) {
		module.groups.startScheduler()
	}
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
	if (bundle.SchemaVersion != 1 && bundle.SchemaVersion != 2) || bundle.Scope != "device_groups_and_quotas" {
		return nil, &models.DevicePolicyError{Code: "incompatible_version", Message: "unsupported policy bundle version or scope"}
	}
	source := bundle.NetworkID
	if source == "" {
		source = defaultLAN
	}
	if source != selectedLAN(ctx) || (bundle.SchemaVersion == 1 && source != defaultLAN) {
		return nil, &models.DevicePolicyError{Code: "bundle_network_mismatch", Message: "此备份不属于当前网络，请切换到原网络后导入"}
	}
	if bundle.SchemaVersion == 2 && bundle.NetworkID == "" {
		return nil, &models.DevicePolicyError{Code: "bundle_network_required", Message: "备份缺少网络标识"}
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
	if request.ExpectedGroupVersion != "" && request.ExpectedGroupVersion != bindLANVersion(ctx, current.Version) {
		return nil, &models.DevicePolicyError{Code: "conflict", Message: "device groups changed; export again before importing"}
	}
	for _, group := range bundle.Groups {
		if policyErr := module.groups.validateScopeMutation(ctx, &models.DeviceGroupMutationRequest{Action: "upsert_group", Group: group}, current); policyErr != nil {
			return nil, policyErr
		}
	}
	for id, policy := range bundle.DevicePolicies {
		if policyErr := module.groups.validateScopeMutation(ctx, &models.DeviceGroupMutationRequest{Action: "set_device_policy", DeviceID: id, DevicePolicy: policy}, current); policyErr != nil {
			return nil, policyErr
		}
	}
	if policyErr := module.groups.validateScopeMutation(ctx, &models.DeviceGroupMutationRequest{Action: "set_global_policy", GlobalPolicy: bundle.GlobalPolicy}, current); policyErr != nil {
		return nil, policyErr
	}
	if selectedLANScope(ctx) != nil && module.groups.inventory != nil {
		inventory, err := module.groups.inventory.Snapshot(ctx)
		if err != nil {
			return nil, &models.DevicePolicyError{Code: "inventory_unavailable", Message: err.Error()}
		}
		known := map[string]bool{}
		for _, device := range inventory.Result.Devices {
			if device != nil {
				known[device.DeviceID] = true
			}
		}
		for id := range bundle.Quotas {
			if !known[id] {
				return nil, &models.DevicePolicyError{Code: "device_network_mismatch", Message: "配额设备不属于当前网络"}
			}
		}
	}
	raw, _ := json.Marshal(bundle)
	sum := sha256.Sum256(raw)
	warnings := []string{"导入将替换当前网络的分组和配额，并按分组策略重新执行；其他网络保持不变。"}
	sort.Strings(warnings)
	return &models.PolicyImportPlan{CanApply: true, NetworkID: selectedLAN(ctx), GroupVersion: bindLANVersion(ctx, current.Version), GroupCount: len(bundle.Groups), DevicePolicyCount: len(bundle.DevicePolicies), QuotaCount: len(bundle.Quotas), Warnings: warnings, Checksum: bindLANVersion(ctx, hex.EncodeToString(sum[:]))}, nil
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

func (module *PolicyBundleModule) scoped(ctx context.Context) *PolicyBundleModule {
	result := *module
	result.groups = module.groups.forContext(ctx)
	result.traffic = module.traffic.forContext(ctx)
	if module.audit != nil {
		result.audit = module.audit.forContext(ctx)
	}
	return &result
}

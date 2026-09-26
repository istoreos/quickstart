package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/istoreos/quickstart/backend/models"
	lancontrolspeedstats "github.com/istoreos/quickstart/backend/modules/lancontrol/speedstats"
)

func (backend *ServiceBackend) GetSpeedsForAllDevice(ctx context.Context, r *http.Request) (*models.DeviceSpeedStatsResponse, error) {
	lstat := backend.lstats
	hosts := lstat.reqSnapshotContext(ctx, "", true).hosts
	return lancontrolspeedstats.BuildAllDeviceResponse(lanSpeedHosts(hosts)), nil
}

func (backend *ServiceBackend) GetSpeedsForOneDevice(ctx context.Context, r *http.Request) (*models.NetworkStatisticsResponse, error) {
	var req models.SpeedsForOneDeviceRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if req.IP == "" {
		return nil, errors.New("IP is required")
	}
	ip := req.IP
	lstat := backend.lstats
	hosts := lstat.reqSnapshotContext(ctx, ip, false).hosts
	if len(hosts) == 0 || len(hosts[0].items) == 0 {
		return lancontrolspeedstats.BuildHistoryResponse(nil, int64(slots)), nil
	}
	return lancontrolspeedstats.BuildHistoryResponse(lanSpeedSamples(hosts[0].items), int64(slots)), nil
}

func lanSpeedHosts(hosts []*LanHostRet) []lancontrolspeedstats.Host {
	result := make([]lancontrolspeedstats.Host, 0, len(hosts))
	for _, host := range hosts {
		result = append(result, lancontrolspeedstats.Host{
			IP:      host.ip,
			Samples: lanSpeedSamples(host.items),
		})
	}
	return result
}

func lanSpeedSamples(items []*NetworkStatisticsItem) []lancontrolspeedstats.Sample {
	samples := make([]lancontrolspeedstats.Sample, 0, len(items))
	for _, item := range items {
		samples = append(samples, lancontrolspeedstats.Sample{
			StartTime:     item.startTime,
			EndTime:       item.endTime,
			UploadSpeed:   item.txAvg,
			DownloadSpeed: item.rxAvg,
		})
	}
	return samples
}

func (backend *ServiceBackend) PostLanDhcpTagsConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlDhcpTagConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	if err := DhcpTagsConfig(ctx, DhcpTagConfigInput{
		Action:     req.Action,
		TagName:    req.TagName,
		TagTitle:   req.TagTitle,
		DhcpOption: req.DhcpOption,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanDhcpGatewayConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlDhcpGatewayConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	if err := DhcpGatewayConfig(ctx, DhcpGatewayInput{
		DhcpEnabled: req.DhcpEnabled,
		DhcpGateway: req.DhcpGateway,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanSpeedLimitConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlSpeedLimitItem
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if req.Action == "" {
		return nil, errors.New("action is required")
	}
	if req.Action != "delete" && req.Mac == "" {
		return nil, errors.New("mac is required")
	}
	req.Mac = strings.ToUpper(req.Mac)
	if err := newLanSpeedLimitWriteService().UpsertSpeedLimitRule(ctx, SpeedLimitWriteInput{
		Action:        req.Action,
		IP:            req.IP,
		MAC:           req.Mac,
		NetworkAccess: req.NetworkAccess,
		UploadSpeed:   req.UploadSpeed,
		DownloadSpeed: req.DownloadSpeed,
		Comment:       req.Comment,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanEnableSpeedLimit(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlSpeedLimitModule
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if req.DownloadSpeed == 0 {
		req.DownloadSpeed = 2000
	}
	if req.UploadSpeed == 0 {
		req.UploadSpeed = 200
	}
	if err := newLanSpeedLimitWriteService().SetSpeedLimitModule(ctx, SpeedLimitModuleInput{
		Enabled:       req.Enabled,
		UploadSpeed:   req.UploadSpeed,
		DownloadSpeed: req.DownloadSpeed,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanEnableFloatGateway(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANCtrlFloatGatewayModule
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if err := newLanFloatGatewayWriteService().SetFloatGateway(ctx, FloatGatewayWriteInput{
		Enabled:         req.Enabled,
		Role:            req.Role,
		SetIP:           req.SetIP,
		CheckIP:         req.CheckIP,
		CheckURL:        req.CheckURL,
		CheckURLTimeout: req.CheckURLTimeout,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) PostLanStaticDeviceConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	var req models.LANStaticAssigned
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	if err := newLanStaticAssignmentWriteService().ApplyStaticAssignment(ctx, StaticAssignmentWriteInput{
		Action:      req.Action,
		AssignedMAC: req.AssignedMac,
		AssignedIP:  req.AssignedIP,
		BindIP:      req.BindIP,
		Hostname:    req.Hostname,
		TagName:     req.TagName,
		TagTitle:    req.TagTitle,
	}); err != nil {
		return nil, err
	}
	return &models.JSONResponse{}, nil
}

func (backend *ServiceBackend) GetLanGlobalConfigs(ctx context.Context) (*models.LANCtrlGlobalConfigResponse, error) {
	return newLanGlobalConfigService().GetGlobalConfigs(ctx)
}

func (backend *ServiceBackend) GetLanListDevices(ctx context.Context) (*models.LANDeviceResponse, error) {
	return newLanDeviceListService().GetListDevices(ctx, backend)
}

func (backend *ServiceBackend) GetDeviceInventoryV2(ctx context.Context) (*models.DeviceInventoryResponse, error) {
	backend.mu.Lock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	inventory := backend.deviceInventory
	backend.mu.Unlock()
	response, err := inventory.Snapshot(ctx)
	if err == nil {
		backend.auditModule().ObserveInventory(response)
	}
	return response, err
}

func (backend *ServiceBackend) PostDeviceInventoryV2(ctx context.Context, r *http.Request) (*models.DeviceInventoryResponse, error) {
	var request models.DeviceInventoryAddRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, errors.New("invalid manual device request")
	}
	backend.mu.Lock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	inventory := backend.deviceInventory
	backend.mu.Unlock()
	return inventory.AddManual(ctx, &request)
}

func (backend *ServiceBackend) GetDeviceClassificationV2(ctx context.Context, r *http.Request) (*models.DeviceClassificationResponse, error) {
	return backend.deviceClassificationModule().Get(ctx, r.URL.Query().Get("deviceId"))
}

func (backend *ServiceBackend) PostDeviceClassificationV2(ctx context.Context, r *http.Request) (*models.DeviceClassificationResponse, error) {
	var request models.DeviceClassificationApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return classificationFailure("", "validation_failed", "invalid classification request"), nil
	}
	return backend.deviceClassificationModule().Apply(ctx, &request)
}

func (backend *ServiceBackend) GetDeviceProfileV2(ctx context.Context, r *http.Request) (*models.DeviceProfileResponse, error) {
	return backend.deviceProfileModule().Get(ctx, r.URL.Query().Get("deviceId"))
}

func (backend *ServiceBackend) PostDeviceProfileV2(ctx context.Context, r *http.Request) (*models.DeviceProfileResponse, error) {
	var request models.DeviceProfileApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return deviceProfileFailure("validation_failed", "invalid device profile request"), nil
	}
	return backend.deviceProfileModule().Apply(ctx, &request)
}

func (backend *ServiceBackend) deviceProfileModule() *DeviceProfileModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.deviceProfile == nil {
		backend.deviceProfile = NewDeviceProfileModule(backend.deviceInventory)
		backend.deviceProfile.transactions = backend.taskTransactionJournalLocked()
	}
	return backend.deviceProfile
}

func (backend *ServiceBackend) deviceClassificationModule() *DeviceClassificationModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.deviceClassification == nil {
		backend.deviceClassification = NewDeviceClassificationModule(backend.deviceInventory)
	}
	return backend.deviceClassification
}

func (backend *ServiceBackend) GetDeviceTrafficV2(ctx context.Context) (*models.DeviceTrafficResponse, error) {
	backend.mu.Lock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.deviceTraffic == nil {
		backend.deviceTraffic = NewDeviceTrafficModule(backend.deviceInventory, backend.lstats)
	}
	traffic := backend.deviceTraffic
	backend.mu.Unlock()
	response, err := traffic.Snapshot(ctx)
	if err == nil && response != nil {
		backend.mu.Lock()
		if backend.trafficInsights == nil {
			backend.trafficInsights = NewDefaultTrafficInsightsModule(backend.devicePolicy)
			if backend.deviceGroups != nil {
				backend.deviceGroups.AttachTrafficInsights(backend.trafficInsights)
			}
			backend.trafficInsights.AttachCollector(traffic.Snapshot)
		}
		insights := backend.trafficInsights
		backend.mu.Unlock()
		_ = insights.Record(ctx, response)
	}
	return response, err
}

func (backend *ServiceBackend) GetTrafficInsightsV2(ctx context.Context, r *http.Request) (*models.TrafficInsightsResponse, error) {
	return backend.trafficInsightsModule().Get(ctx, r.URL.Query().Get("deviceId"), r.URL.Query().Get("range"))
}

func (backend *ServiceBackend) PostTrafficQuotaV2(ctx context.Context, r *http.Request) (*models.TrafficInsightsResponse, error) {
	var request models.TrafficQuotaRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return trafficInsightsFailure("validation_failed", "invalid traffic quota request"), nil
	}
	response, err := backend.trafficInsightsModule().SetQuota(ctx, &request)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil {
		backend.auditModule().Record(request.DeviceID, "traffic_quota", "policy_changed", "success", "")
	}
	return response, err
}

func (backend *ServiceBackend) trafficInsightsModule() *TrafficInsightsModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.devicePolicy == nil {
		backend.devicePolicy = NewDevicePolicyModule(backend.deviceInventory)
		backend.devicePolicy.transactions = backend.taskTransactionJournalLocked()
	}
	if backend.trafficInsights == nil {
		backend.trafficInsights = NewDefaultTrafficInsightsModule(backend.devicePolicy)
		if backend.deviceGroups != nil {
			backend.deviceGroups.AttachTrafficInsights(backend.trafficInsights)
		}
	}
	if backend.deviceTraffic != nil {
		backend.trafficInsights.AttachCollector(backend.deviceTraffic.Snapshot)
	}
	if backend.networkAudit != nil {
		backend.trafficInsights.audit = backend.networkAudit
	}
	return backend.trafficInsights
}

func (backend *ServiceBackend) GetAdvancedNetworkV2(ctx context.Context, r *http.Request) (*models.AdvancedNetworkResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return backend.advancedNetworkModule().Status(r.URL.Query().Get("deviceId"))
}

func (backend *ServiceBackend) PostManagementProbeV2(ctx context.Context, r *http.Request) (*models.ManagementProbeResponse, error) {
	var request models.ManagementProbeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return managementProbeFailure("validation_failed", "invalid management probe request"), nil
	}
	return backend.advancedNetworkModule().Probe(ctx, &request)
}

func (backend *ServiceBackend) PostNetworkWebhookV2(ctx context.Context, r *http.Request) (*models.AdvancedNetworkResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var request models.WebhookConfig
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return advancedNetworkFailure("validation_failed", "invalid webhook config"), nil
	}
	return backend.auditModule().SetWebhook(&request)
}

func (backend *ServiceBackend) GetPolicyBundleV2(ctx context.Context) (*models.PolicyBundleResponse, error) {
	return backend.advancedNetworkModule().Export(ctx)
}
func (backend *ServiceBackend) PostPolicyImportPlanV2(ctx context.Context, r *http.Request) (*models.PolicyBundleResponse, error) {
	var request models.PolicyImportRequest
	if err := decodeBoundedJSON(r, &request, 4<<20); err != nil {
		return policyBundleFailure("validation_failed", "invalid policy import request"), nil
	}
	return backend.advancedNetworkModule().ImportPlan(ctx, &request)
}
func (backend *ServiceBackend) PostPolicyImportApplyV2(ctx context.Context, r *http.Request) (*models.PolicyBundleResponse, error) {
	var request models.PolicyImportRequest
	if err := decodeBoundedJSON(r, &request, 4<<20); err != nil {
		return policyBundleFailure("validation_failed", "invalid policy import request"), nil
	}
	return backend.advancedNetworkModule().ImportApply(ctx, &request)
}

func decodeBoundedJSON(r *http.Request, target any, limit int64) error {
	if r == nil || r.Body == nil || limit <= 0 {
		return errors.New("invalid JSON request")
	}
	decoder := json.NewDecoder(io.LimitReader(r.Body, limit+1))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}

func (backend *ServiceBackend) auditModule() *NetworkAuditModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.networkAudit == nil {
		backend.networkAudit = NewDefaultNetworkAuditModule()
	}
	if backend.trafficInsights != nil {
		backend.trafficInsights.audit = backend.networkAudit
	}
	return backend.networkAudit
}
func (backend *ServiceBackend) advancedNetworkModule() *AdvancedNetworkModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.devicePolicy == nil {
		backend.devicePolicy = NewDevicePolicyModule(backend.deviceInventory)
		backend.devicePolicy.transactions = backend.taskTransactionJournalLocked()
	}
	if backend.deviceNetworkPolicy == nil {
		if backend.gatewayPolicy == nil {
			backend.gatewayPolicy = NewDefaultGatewayPolicyModule(backend.deviceInventory)
		}
		backend.deviceNetworkPolicy = NewDefaultDeviceNetworkPolicyModule(backend.deviceInventory, backend.devicePolicy, backend.gatewayPolicy)
		backend.deviceNetworkPolicy.transactions = backend.taskTransactionJournalLocked()
	}
	if backend.deviceGroups == nil {
		backend.deviceGroups = NewDefaultDeviceGroupModule(backend.devicePolicy, backend.deviceNetworkPolicy, backend.trafficInsights)
	}
	if backend.trafficInsights == nil {
		backend.trafficInsights = NewDefaultTrafficInsightsModule(backend.devicePolicy)
		backend.deviceGroups.AttachTrafficInsights(backend.trafficInsights)
	}
	if backend.networkAudit == nil {
		backend.networkAudit = NewDefaultNetworkAuditModule()
	}
	if backend.advancedNetwork == nil {
		backend.advancedNetwork = NewAdvancedNetworkModule(backend.deviceInventory, backend.deviceGroups, backend.trafficInsights, backend.networkAudit)
	}
	return backend.advancedNetwork
}

func (backend *ServiceBackend) GetDeviceRuntimeDiagnosticsV2(ctx context.Context) (*models.DeviceRuntimeDiagnosticsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	backend.mu.Lock()
	inventory, traffic, sampler := backend.deviceInventory, backend.deviceTraffic, backend.lstats
	backend.mu.Unlock()
	oui := GomanufDiagnostics()
	return &models.DeviceRuntimeDiagnosticsResponse{Result: &models.DeviceRuntimeDiagnostics{
		Inventory: inventory.diagnostics(),
		Traffic:   traffic.diagnostics(),
		Sampler:   sampler.diagnostics(),
		OUI: &models.DeviceOUIRuntimeDiagnostics{
			Entries: int64(oui.Entries), SourceBytes: oui.SourceBytes,
			LoadDurationMS: oui.LoadDuration.Milliseconds(), HeapAllocBytes: int64(oui.HeapAllocDelta),
		},
	}}, nil
}

func (backend *ServiceBackend) GetGatewayTargetsV2(ctx context.Context) (*models.GatewayTargetListResponse, error) {
	return backend.gatewayPolicyModule().ListTargets(ctx)
}

func (backend *ServiceBackend) PostGatewayAssignmentPlanV2(ctx context.Context, r *http.Request) (*models.GatewayAssignmentPlanResponse, error) {
	var request models.GatewayAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return &models.GatewayAssignmentPlanResponse{Result: &models.GatewayAssignmentPlan{
			CanApply: false, Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid gateway assignment request"},
		}}, nil
	}
	return backend.gatewayPolicyModule().PlanAssignment(ctx, &request)
}

func (backend *ServiceBackend) PostGatewayAssignmentApplyV2(ctx context.Context, r *http.Request) (*models.GatewayAssignmentApplyResponse, error) {
	var request models.GatewayAssignmentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return &models.GatewayAssignmentApplyResponse{Result: &models.GatewayAssignmentApplyResult{
			Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid gateway assignment request"},
		}}, nil
	}
	response, err := backend.gatewayPolicyModule().ApplyAssignment(ctx, &request)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil {
		backend.auditModule().Record(request.DeviceID, "gateway_assignment", "policy_changed", "success", request.Action)
	}
	return response, err
}

func (backend *ServiceBackend) GetGatewayReferencesV2(ctx context.Context, r *http.Request) (*models.GatewayReferencesResponse, error) {
	return backend.gatewayPolicyModule().References(ctx, r.URL.Query().Get("targetId"))
}

func (backend *ServiceBackend) PostGatewayTargetPlanV2(ctx context.Context, r *http.Request) (*models.GatewayTargetMutationPlanResponse, error) {
	var request models.GatewayTargetMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return &models.GatewayTargetMutationPlanResponse{Result: &models.GatewayTargetMutationPlan{
			CanApply: false, ReferenceSummary: &models.GatewayReferenceSummary{},
			Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid gateway target request"},
		}}, nil
	}
	return backend.gatewayPolicyModule().PlanTargetMutation(ctx, &request)
}

func (backend *ServiceBackend) PostGatewayTargetApplyV2(ctx context.Context, r *http.Request) (*models.GatewayTargetMutationApplyResponse, error) {
	var request models.GatewayTargetMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return &models.GatewayTargetMutationApplyResponse{Result: &models.GatewayTargetMutationApplyResult{
			Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid gateway target request"},
		}}, nil
	}
	response, err := backend.gatewayPolicyModule().ApplyTargetMutation(ctx, &request)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil && response.Result.Changed {
		backend.auditModule().Record("", "gateway_target", "policy_changed", "success", request.Action)
	}
	return response, err
}

func (backend *ServiceBackend) GetDeviceNetworkPolicyV2(ctx context.Context, r *http.Request) (*models.DeviceNetworkPolicyResponse, error) {
	return backend.deviceNetworkPolicyModule().Get(ctx, r.URL.Query().Get("deviceId"))
}

func (backend *ServiceBackend) PostDeviceNetworkPolicyV2(ctx context.Context, r *http.Request) (*models.DeviceNetworkPolicyResponse, error) {
	var request models.DeviceNetworkPolicyApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return deviceNetworkPolicyFailure(&models.DevicePolicyError{Code: "validation_failed", Message: "invalid device network policy request"}), nil
	}
	response, err := backend.deviceNetworkPolicyModule().Apply(ctx, &request)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil && response.Result.Changed {
		backend.auditModule().Record(request.DeviceID, "device_network_policy", "policy_changed", "success", "")
	} else if err == nil && response != nil && response.Result != nil && isAddressConflict(response.Result.Error) {
		backend.auditModule().Record(request.DeviceID, "address_reservation", "address_conflict", "rejected", "")
	}
	return response, err
}

func (backend *ServiceBackend) deviceNetworkPolicyModule() *DeviceNetworkPolicyModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.devicePolicy == nil {
		backend.devicePolicy = NewDevicePolicyModule(backend.deviceInventory)
		backend.devicePolicy.transactions = backend.taskTransactionJournalLocked()
	}
	if backend.gatewayPolicy == nil {
		backend.gatewayPolicy = NewDefaultGatewayPolicyModule(backend.deviceInventory)
	}
	if backend.deviceNetworkPolicy == nil {
		backend.deviceNetworkPolicy = NewDefaultDeviceNetworkPolicyModule(backend.deviceInventory, backend.devicePolicy, backend.gatewayPolicy)
		backend.deviceNetworkPolicy.transactions = backend.taskTransactionJournalLocked()
	}
	return backend.deviceNetworkPolicy
}

func (backend *ServiceBackend) GetFloatingGatewayV2(ctx context.Context) (*models.FloatingGatewayResponse, error) {
	response, err := backend.floatingGatewayModule().Get(ctx)
	if err == nil {
		backend.auditModule().ObserveFloatingGateway(response)
	}
	return response, err
}

func (backend *ServiceBackend) PostFloatingGatewayPlanV2(ctx context.Context, r *http.Request) (*models.FloatingGatewayResponse, error) {
	var request models.FloatingGatewayApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return &models.FloatingGatewayResponse{Result: &models.FloatingGatewayResult{Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid floating gateway request"}}}, nil
	}
	return backend.floatingGatewayModule().Plan(ctx, &request)
}

func (backend *ServiceBackend) PostFloatingGatewayApplyV2(ctx context.Context, r *http.Request) (*models.FloatingGatewayResponse, error) {
	var request models.FloatingGatewayApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return &models.FloatingGatewayResponse{Result: &models.FloatingGatewayResult{Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid floating gateway request"}}}, nil
	}
	response, err := backend.floatingGatewayModule().Apply(ctx, &request)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil && response.Result.Changed {
		backend.auditModule().Record("", "floating_gateway", "policy_changed", "success", "")
		backend.auditModule().ObserveFloatingGateway(response)
	}
	return response, err
}

func (backend *ServiceBackend) GetFloatingGatewayDrillPlanV2(ctx context.Context) (*models.FloatingGatewayDrillResponse, error) {
	return backend.floatingGatewayModule().DrillPlan(ctx)
}

func (backend *ServiceBackend) floatingGatewayModule() *FloatingGatewayModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.gatewayPolicy == nil {
		backend.gatewayPolicy = NewDefaultGatewayPolicyModule(backend.deviceInventory)
	}
	if backend.floatingGateway == nil {
		backend.floatingGateway = NewDefaultFloatingGatewayModule(backend.gatewayPolicy)
	}
	return backend.floatingGateway
}

func (backend *ServiceBackend) GetNetworkRulesV2(ctx context.Context) (*models.NetworkRulesResponse, error) {
	return backend.networkRulesModule().List(ctx)
}

func (backend *ServiceBackend) PostNetworkRulesPlanV2(ctx context.Context, r *http.Request) (*models.NetworkRulesBulkResponse, error) {
	var request models.NetworkRulesBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return &models.NetworkRulesBulkResponse{Result: &models.NetworkRulesBulkResult{Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid network rules request"}}}, nil
	}
	return backend.networkRulesModule().Plan(ctx, &request)
}

func (backend *ServiceBackend) PostNetworkRulesApplyV2(ctx context.Context, r *http.Request) (*models.NetworkRulesBulkResponse, error) {
	var request models.NetworkRulesBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return &models.NetworkRulesBulkResponse{Result: &models.NetworkRulesBulkResult{Error: &models.DevicePolicyError{Code: "validation_failed", Message: "invalid network rules request"}}}, nil
	}
	response, err := backend.networkRulesModule().Apply(ctx, &request)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil {
		backend.auditModule().Record("", "network_rules", "policy_changed", "success", "")
	}
	return response, err
}

func (backend *ServiceBackend) networkRulesModule() *NetworkRulesModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.devicePolicy == nil {
		backend.devicePolicy = NewDevicePolicyModule(backend.deviceInventory)
		backend.devicePolicy.transactions = backend.taskTransactionJournalLocked()
	}
	if backend.gatewayPolicy == nil {
		backend.gatewayPolicy = NewDefaultGatewayPolicyModule(backend.deviceInventory)
	}
	if backend.networkRules == nil {
		backend.networkRules = NewDefaultNetworkRulesModule(backend.deviceInventory, backend.devicePolicy, backend.gatewayPolicy)
	}
	return backend.networkRules
}

func (backend *ServiceBackend) GetLanDeviceMigrationPlanV2(ctx context.Context) (*models.LanDeviceMigrationResponse, error) {
	return backend.lanDeviceMigrationModule().Plan(ctx)
}

func (backend *ServiceBackend) PostLanDeviceMigrationApplyV2(ctx context.Context, r *http.Request) (*models.LanDeviceMigrationResponse, error) {
	var request models.LanDeviceMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return migrationResponse(nil, false, &models.DevicePolicyError{Code: "validation_failed", Message: "invalid migration request"}), nil
	}
	return backend.lanDeviceMigrationModule().Apply(ctx, &request)
}

func (backend *ServiceBackend) lanDeviceMigrationModule() *LanDeviceMigrationModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.devicePolicy == nil {
		backend.devicePolicy = NewDevicePolicyModule(backend.deviceInventory)
		backend.devicePolicy.transactions = backend.taskTransactionJournalLocked()
	}
	if backend.gatewayPolicy == nil {
		backend.gatewayPolicy = NewDefaultGatewayPolicyModule(backend.deviceInventory)
	}
	if backend.networkRules == nil {
		backend.networkRules = NewDefaultNetworkRulesModule(backend.deviceInventory, backend.devicePolicy, backend.gatewayPolicy)
	}
	if backend.lanDeviceMigration == nil {
		backend.lanDeviceMigration = NewDefaultLanDeviceMigrationModule(backend.networkRules)
	}
	return backend.lanDeviceMigration
}

func (backend *ServiceBackend) PostCapabilityActionPlanV2(ctx context.Context, r *http.Request) (*models.CapabilityActionResponse, error) {
	var request models.CapabilityActionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return capabilityActionResponse(nil, false, false, nil, &models.DevicePolicyError{Code: "validation_failed", Message: "invalid capability action request"}), nil
	}
	return backend.capabilityActionModule().Plan(ctx, &request)
}

func (backend *ServiceBackend) PostCapabilityActionApplyV2(ctx context.Context, r *http.Request) (*models.CapabilityActionResponse, error) {
	var request models.CapabilityActionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return capabilityActionResponse(nil, false, false, nil, &models.DevicePolicyError{Code: "validation_failed", Message: "invalid capability action request"}), nil
	}
	return backend.capabilityActionModule().Apply(ctx, &request)
}

func (backend *ServiceBackend) capabilityActionModule() *CapabilityActionModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.capabilityActions == nil {
		backend.capabilityActions = NewDefaultCapabilityActionModule()
	}
	return backend.capabilityActions
}

func (backend *ServiceBackend) GetDeviceGroupsV2(ctx context.Context) (*models.DeviceGroupsResponse, error) {
	return backend.deviceGroupsModule().List(ctx)
}

func (backend *ServiceBackend) PostDeviceGroupsV2(ctx context.Context, r *http.Request) (*models.DeviceGroupsResponse, error) {
	var request models.DeviceGroupMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return deviceGroupsFailure(&models.DevicePolicyError{Code: "validation_failed", Message: "invalid device group request"}), nil
	}
	response, err := backend.deviceGroupsModule().Mutate(ctx, &request)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil {
		backend.auditModule().Record(request.DeviceID, "device_groups", "policy_changed", "success", request.Action)
	}
	return response, err
}

func (backend *ServiceBackend) deviceGroupsModule() *DeviceGroupModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.devicePolicy == nil {
		backend.devicePolicy = NewDevicePolicyModule(backend.deviceInventory)
		backend.devicePolicy.transactions = backend.taskTransactionJournalLocked()
	}
	if backend.gatewayPolicy == nil {
		backend.gatewayPolicy = NewDefaultGatewayPolicyModule(backend.deviceInventory)
	}
	if backend.deviceNetworkPolicy == nil {
		backend.deviceNetworkPolicy = NewDefaultDeviceNetworkPolicyModule(backend.deviceInventory, backend.devicePolicy, backend.gatewayPolicy)
		backend.deviceNetworkPolicy.transactions = backend.taskTransactionJournalLocked()
	}
	if backend.deviceGroups == nil {
		backend.deviceGroups = NewDefaultDeviceGroupModule(backend.devicePolicy, backend.deviceNetworkPolicy, backend.trafficInsights)
	}
	return backend.deviceGroups
}

func (backend *ServiceBackend) gatewayPolicyModule() *GatewayPolicyModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.gatewayPolicy == nil {
		backend.gatewayPolicy = NewDefaultGatewayPolicyModule(backend.deviceInventory)
	}
	return backend.gatewayPolicy
}

func (backend *ServiceBackend) GetDevicePolicyV2(ctx context.Context, r *http.Request) (*models.DevicePolicyResponse, error) {
	return backend.devicePolicyModule().Get(ctx, r.URL.Query().Get("deviceId"))
}

func (backend *ServiceBackend) PostDevicePolicyV2(ctx context.Context, r *http.Request) (*models.DevicePolicyResponse, error) {
	var request models.DevicePolicyApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return devicePolicyFailure("validation_failed", "invalid policy request"), nil
	}
	response, err := backend.devicePolicyModule().Apply(ctx, &request)
	if err == nil && response != nil && response.Result != nil && response.Result.Error == nil && response.Result.Changed {
		backend.auditModule().Record(request.DeviceID, "device_policy", "policy_changed", "success", request.Kind)
	} else if err == nil && response != nil && response.Result != nil && isAddressConflict(response.Result.Error) {
		backend.auditModule().Record(request.DeviceID, "address_reservation", "address_conflict", "rejected", "")
	}
	return response, err
}

func isAddressConflict(policyErr *models.DevicePolicyError) bool {
	return policyErr != nil && policyErr.Code == "conflict" && strings.Contains(strings.ToLower(policyErr.Message), "ipv4 address")
}

func (backend *ServiceBackend) GetDevicePolicyRulesV2(ctx context.Context) (*models.DevicePolicyRulesResponse, error) {
	return backend.devicePolicyModule().ListRules(ctx)
}

func (backend *ServiceBackend) devicePolicyModule() *DevicePolicyModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.deviceInventory == nil {
		backend.deviceInventory = NewDeviceInventoryModule()
	}
	if backend.devicePolicy == nil {
		backend.devicePolicy = NewDevicePolicyModule(backend.deviceInventory)
		backend.devicePolicy.transactions = backend.taskTransactionJournalLocked()
	}
	return backend.devicePolicy
}

func (backend *ServiceBackend) taskTransactionJournalLocked() *TaskTransactionJournal {
	if backend.taskTransactions == nil {
		backend.taskTransactions = NewDefaultTaskTransactionJournal()
	}
	return backend.taskTransactions
}

func (backend *ServiceBackend) GetLanListStaticDevices(ctx context.Context) (*models.LANCtrlStaticAssignedResponse, error) {
	return newLanStaticDeviceListService().GetListStaticDevices(ctx)
}

func (backend *ServiceBackend) GetLanListSpeedLimitedDevices(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error) {
	return newLanSpeedLimitedDeviceListService().GetListSpeedLimitedDevices(ctx)
}

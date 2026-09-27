package lancontrol

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/istoreos/quickstart/backend/internal/httpapi"
	"github.com/istoreos/quickstart/backend/models"
	"github.com/julienschmidt/httprouter"
)

type fakeLanControlBackend struct {
	err error

	calls        []string
	requestPaths []string
}

func (backend *fakeLanControlBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeLanControlBackend) recordRequest(call string, r *http.Request) {
	backend.record(call)
	backend.requestPaths = append(backend.requestPaths, r.URL.Path)
}

func (backend *fakeLanControlBackend) GetSpeedsForAllDevice(ctx context.Context, r *http.Request) (*models.DeviceSpeedStatsResponse, error) {
	backend.recordRequest("speedsForDevices", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.DeviceSpeedStatsResponse{
		Result: []*models.DeviceSpeedStat{{IP: "192.168.1.2", DownloadSpeed: 1024}},
	}, nil
}

func (backend *fakeLanControlBackend) GetSpeedsForOneDevice(ctx context.Context, r *http.Request) (*models.NetworkStatisticsResponse, error) {
	backend.recordRequest("speedsForOneDevice", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkStatisticsResponse{
		Result: &models.NetworkStatisticsResponseResult{
			Items: []*models.NetworkStatisticsItem{{DownloadSpeed: 2048}},
		},
	}, nil
}

func (backend *fakeLanControlBackend) PostLanDhcpTagsConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("dhcpTagsConfig", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanDhcpGatewayConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("dhcpGatewayConfig", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanSpeedLimitConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("speedLimitConfig", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanEnableSpeedLimit(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("enableSpeedLimit", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostRateLimitSettingsPlanV2(ctx context.Context, r *http.Request) (*models.RateLimitSettingsResponse, error) {
	backend.recordRequest("rateLimitSettingsPlanV2", r)
	return &models.RateLimitSettingsResponse{Result: &models.RateLimitSettingsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostRateLimitSettingsApplyV2(ctx context.Context, r *http.Request) (*models.RateLimitSettingsResponse, error) {
	backend.recordRequest("rateLimitSettingsApplyV2", r)
	return &models.RateLimitSettingsResponse{Result: &models.RateLimitSettingsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostLanEnableFloatGateway(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("enableFloatGateway", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanStaticDeviceConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("staticDeviceConfig", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) GetLanGlobalConfigs(ctx context.Context) (*models.LANCtrlGlobalConfigResponse, error) {
	backend.record("globalConfigs")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.LANCtrlGlobalConfigResponse{Result: &models.LANCtrlGlobalConfig{}}, nil
}

func (backend *fakeLanControlBackend) GetLanListDevices(ctx context.Context) (*models.LANDeviceResponse, error) {
	backend.record("listDevices")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.LANDeviceResponse{
		Result: &models.LANDeviceResponseResult{Devices: models.LANDevices{{IP: "192.168.1.2"}}},
	}, nil
}

func (backend *fakeLanControlBackend) GetDeviceInventoryV2(ctx context.Context) (*models.DeviceInventoryResponse, error) {
	backend.record("deviceInventoryV2")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.DeviceInventoryResponse{Result: &models.DeviceInventoryResult{
		Devices: []*models.DeviceInventoryItem{{DeviceID: "mac:aa:bb:cc:dd:ee:ff"}},
	}}, nil
}

func (backend *fakeLanControlBackend) PostDeviceInventoryV2(ctx context.Context, r *http.Request) (*models.DeviceInventoryResponse, error) {
	backend.recordRequest("postDeviceInventoryV2", r)
	return &models.DeviceInventoryResponse{Result: &models.DeviceInventoryResult{Devices: []*models.DeviceInventoryItem{}}}, backend.err
}

func (backend *fakeLanControlBackend) GetDeviceClassificationV2(ctx context.Context, r *http.Request) (*models.DeviceClassificationResponse, error) {
	backend.recordRequest("getDeviceClassificationV2", r)
	return &models.DeviceClassificationResponse{Result: &models.DeviceClassificationResult{
		DeviceID: r.URL.Query().Get("deviceId"),
	}}, nil
}

func (backend *fakeLanControlBackend) PostDeviceClassificationV2(ctx context.Context, r *http.Request) (*models.DeviceClassificationResponse, error) {
	backend.recordRequest("postDeviceClassificationV2", r)
	return &models.DeviceClassificationResponse{Result: &models.DeviceClassificationResult{Changed: true}}, nil
}

func (backend *fakeLanControlBackend) GetDeviceProfileV2(ctx context.Context, r *http.Request) (*models.DeviceProfileResponse, error) {
	backend.recordRequest("getDeviceProfileV2", r)
	return &models.DeviceProfileResponse{Result: &models.DeviceProfileResult{Profile: &models.DeviceProfile{DeviceID: r.URL.Query().Get("deviceId")}}}, nil
}

func (backend *fakeLanControlBackend) PostDeviceProfileV2(ctx context.Context, r *http.Request) (*models.DeviceProfileResponse, error) {
	backend.recordRequest("postDeviceProfileV2", r)
	return &models.DeviceProfileResponse{Result: &models.DeviceProfileResult{Changed: true}}, nil
}

func (backend *fakeLanControlBackend) GetDeviceTrafficV2(ctx context.Context) (*models.DeviceTrafficResponse, error) {
	backend.record("deviceTrafficV2")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{
		Items: []*models.DeviceTrafficItem{{DeviceID: "mac:aa:bb:cc:dd:ee:ff", State: "ready"}},
	}}, nil
}

func (backend *fakeLanControlBackend) GetDeviceRuntimeDiagnosticsV2(ctx context.Context) (*models.DeviceRuntimeDiagnosticsResponse, error) {
	backend.record("deviceRuntimeDiagnosticsV2")
	return &models.DeviceRuntimeDiagnosticsResponse{Result: &models.DeviceRuntimeDiagnostics{}}, backend.err
}

func (backend *fakeLanControlBackend) GetGatewayTargetsV2(ctx context.Context) (*models.GatewayTargetListResponse, error) {
	backend.record("gatewayTargetsV2")
	return &models.GatewayTargetListResponse{Result: &models.GatewayTargetListResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostGatewayAssignmentPlanV2(ctx context.Context, r *http.Request) (*models.GatewayAssignmentPlanResponse, error) {
	backend.recordRequest("gatewayAssignmentPlanV2", r)
	return &models.GatewayAssignmentPlanResponse{Result: &models.GatewayAssignmentPlan{}}, backend.err
}

func (backend *fakeLanControlBackend) PostGatewayAssignmentApplyV2(ctx context.Context, r *http.Request) (*models.GatewayAssignmentApplyResponse, error) {
	backend.recordRequest("gatewayAssignmentApplyV2", r)
	return &models.GatewayAssignmentApplyResponse{Result: &models.GatewayAssignmentApplyResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetGatewayReferencesV2(ctx context.Context, r *http.Request) (*models.GatewayReferencesResponse, error) {
	backend.recordRequest("gatewayReferencesV2", r)
	return &models.GatewayReferencesResponse{Result: &models.GatewayReferencesResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostGatewayTargetPlanV2(ctx context.Context, r *http.Request) (*models.GatewayTargetMutationPlanResponse, error) {
	backend.recordRequest("gatewayTargetPlanV2", r)
	return &models.GatewayTargetMutationPlanResponse{Result: &models.GatewayTargetMutationPlan{}}, backend.err
}

func (backend *fakeLanControlBackend) PostGatewayTargetApplyV2(ctx context.Context, r *http.Request) (*models.GatewayTargetMutationApplyResponse, error) {
	backend.recordRequest("gatewayTargetApplyV2", r)
	return &models.GatewayTargetMutationApplyResponse{Result: &models.GatewayTargetMutationApplyResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetDeviceNetworkPolicyV2(ctx context.Context, r *http.Request) (*models.DeviceNetworkPolicyResponse, error) {
	backend.recordRequest("getDeviceNetworkPolicyV2", r)
	return &models.DeviceNetworkPolicyResponse{Result: &models.DeviceNetworkPolicyResult{Policy: &models.DeviceNetworkPolicy{DeviceID: r.URL.Query().Get("deviceId")}}}, backend.err
}

func (backend *fakeLanControlBackend) PostDeviceNetworkPolicyV2(ctx context.Context, r *http.Request) (*models.DeviceNetworkPolicyResponse, error) {
	backend.recordRequest("postDeviceNetworkPolicyV2", r)
	return &models.DeviceNetworkPolicyResponse{Result: &models.DeviceNetworkPolicyResult{Changed: true}}, backend.err
}

func (backend *fakeLanControlBackend) PostDeviceNetworkPolicyPlanV2(ctx context.Context, r *http.Request) (*models.DeviceNetworkPolicyPlanResponse, error) {
	backend.recordRequest("planDeviceNetworkPolicyV2", r)
	return &models.DeviceNetworkPolicyPlanResponse{Result: &models.DeviceNetworkPolicyPlanResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostDeviceNetworkPolicyApplyV2(ctx context.Context, r *http.Request) (*models.DeviceNetworkPolicyResponse, error) {
	backend.recordRequest("applyDeviceNetworkPolicyV2", r)
	return &models.DeviceNetworkPolicyResponse{Result: &models.DeviceNetworkPolicyResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostDeviceRestrictionsPlanV2(ctx context.Context, r *http.Request) (*models.DeviceRestrictionPlanResponse, error) {
	backend.recordRequest("planDeviceRestrictionsV2", r)
	return &models.DeviceRestrictionPlanResponse{Result: &models.DeviceRestrictionPlanResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostDeviceRestrictionsApplyV2(ctx context.Context, r *http.Request) (*models.DevicePolicyResponse, error) {
	backend.recordRequest("applyDeviceRestrictionsV2", r)
	return &models.DevicePolicyResponse{Result: &models.DevicePolicyResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetFloatingGatewayV2(ctx context.Context) (*models.FloatingGatewayResponse, error) {
	backend.record("getFloatingGatewayV2")
	return &models.FloatingGatewayResponse{Result: &models.FloatingGatewayResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostFloatingGatewayPlanV2(ctx context.Context, r *http.Request) (*models.FloatingGatewayResponse, error) {
	backend.recordRequest("postFloatingGatewayPlanV2", r)
	return &models.FloatingGatewayResponse{Result: &models.FloatingGatewayResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostFloatingGatewayApplyV2(ctx context.Context, r *http.Request) (*models.FloatingGatewayResponse, error) {
	backend.recordRequest("postFloatingGatewayApplyV2", r)
	return &models.FloatingGatewayResponse{Result: &models.FloatingGatewayResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetFloatingGatewayDrillPlanV2(ctx context.Context) (*models.FloatingGatewayDrillResponse, error) {
	backend.record("getFloatingGatewayDrillPlanV2")
	return &models.FloatingGatewayDrillResponse{Result: &models.FloatingGatewayDrillPlan{}}, backend.err
}

func (backend *fakeLanControlBackend) GetNetworkRulesV2(ctx context.Context) (*models.NetworkRulesResponse, error) {
	backend.record("getNetworkRulesV2")
	return &models.NetworkRulesResponse{Result: &models.NetworkRulesResult{}}, backend.err
}
func (backend *fakeLanControlBackend) PostNetworkRulesPlanV2(ctx context.Context, r *http.Request) (*models.NetworkRulesBulkResponse, error) {
	backend.recordRequest("postNetworkRulesPlanV2", r)
	return &models.NetworkRulesBulkResponse{Result: &models.NetworkRulesBulkResult{}}, backend.err
}
func (backend *fakeLanControlBackend) PostNetworkRulesApplyV2(ctx context.Context, r *http.Request) (*models.NetworkRulesBulkResponse, error) {
	backend.recordRequest("postNetworkRulesApplyV2", r)
	return &models.NetworkRulesBulkResponse{Result: &models.NetworkRulesBulkResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetLanDeviceMigrationPlanV2(ctx context.Context) (*models.LanDeviceMigrationResponse, error) {
	backend.record("getLanDeviceMigrationPlanV2")
	return &models.LanDeviceMigrationResponse{Result: &models.LanDeviceMigrationResult{Plan: &models.LanDeviceMigrationPlan{}}}, backend.err
}

func (backend *fakeLanControlBackend) PostLanDeviceMigrationApplyV2(ctx context.Context, r *http.Request) (*models.LanDeviceMigrationResponse, error) {
	backend.recordRequest("postLanDeviceMigrationApplyV2", r)
	return &models.LanDeviceMigrationResponse{Result: &models.LanDeviceMigrationResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetRateLimitMigrationPlanV2(context.Context) (*models.RateLimitMigrationResponse, error) {
	backend.record("getRateLimitMigrationPlanV2")
	return &models.RateLimitMigrationResponse{Result: &models.RateLimitMigrationResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostRateLimitMigrationApplyV2(_ context.Context, r *http.Request) (*models.RateLimitMigrationResponse, error) {
	backend.recordRequest("postRateLimitMigrationApplyV2", r)
	return &models.RateLimitMigrationResponse{Result: &models.RateLimitMigrationResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostRateLimitMigrationRollbackV2(_ context.Context, r *http.Request) (*models.RateLimitMigrationResponse, error) {
	backend.recordRequest("postRateLimitMigrationRollbackV2", r)
	return &models.RateLimitMigrationResponse{Result: &models.RateLimitMigrationResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostCapabilityActionPlanV2(ctx context.Context, r *http.Request) (*models.CapabilityActionResponse, error) {
	backend.recordRequest("postCapabilityActionPlanV2", r)
	return &models.CapabilityActionResponse{Result: &models.CapabilityActionResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostCapabilityActionApplyV2(ctx context.Context, r *http.Request) (*models.CapabilityActionResponse, error) {
	backend.recordRequest("postCapabilityActionApplyV2", r)
	return &models.CapabilityActionResponse{Result: &models.CapabilityActionResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetRouterContextV2(ctx context.Context, r *http.Request) (*models.RouterContextResponse, error) {
	backend.recordRequest("getRouterContextV2", r)
	return &models.RouterContextResponse{Result: &models.RouterContext{LAN: r.URL.Query().Get("lan")}}, backend.err
}

func (backend *fakeLanControlBackend) GetLanDHCPSettingsV2(ctx context.Context) (*models.LanDHCPSettingsResponse, error) {
	backend.record("getLanDHCPSettingsV2")
	return &models.LanDHCPSettingsResponse{Result: &models.LanDHCPSettingsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostLanDHCPSettingsPlanV2(ctx context.Context, r *http.Request) (*models.LanDHCPSettingsResponse, error) {
	backend.recordRequest("planLanDHCPSettingsV2", r)
	return &models.LanDHCPSettingsResponse{Result: &models.LanDHCPSettingsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostLanDHCPSettingsApplyV2(ctx context.Context, r *http.Request) (*models.LanDHCPSettingsResponse, error) {
	backend.recordRequest("applyLanDHCPSettingsV2", r)
	return &models.LanDHCPSettingsResponse{Result: &models.LanDHCPSettingsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetDeviceGroupsV2(ctx context.Context) (*models.DeviceGroupsResponse, error) {
	backend.record("getDeviceGroupsV2")
	return &models.DeviceGroupsResponse{Result: &models.DeviceGroupsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostDeviceGroupsV2(ctx context.Context, r *http.Request) (*models.DeviceGroupsResponse, error) {
	backend.recordRequest("postDeviceGroupsV2", r)
	return &models.DeviceGroupsResponse{Result: &models.DeviceGroupsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetTrafficInsightsV2(ctx context.Context, r *http.Request) (*models.TrafficInsightsResponse, error) {
	backend.recordRequest("getTrafficInsightsV2", r)
	return &models.TrafficInsightsResponse{Result: &models.TrafficInsightsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) PostTrafficQuotaV2(ctx context.Context, r *http.Request) (*models.TrafficInsightsResponse, error) {
	backend.recordRequest("postTrafficQuotaV2", r)
	return &models.TrafficInsightsResponse{Result: &models.TrafficInsightsResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetAdvancedNetworkV2(ctx context.Context, r *http.Request) (*models.AdvancedNetworkResponse, error) {
	backend.recordRequest("getAdvancedNetworkV2", r)
	return &models.AdvancedNetworkResponse{Result: &models.AdvancedNetworkResult{}}, backend.err
}
func (backend *fakeLanControlBackend) PostManagementProbeV2(ctx context.Context, r *http.Request) (*models.ManagementProbeResponse, error) {
	backend.recordRequest("postManagementProbeV2", r)
	return &models.ManagementProbeResponse{Result: &models.ManagementProbeResult{}}, backend.err
}
func (backend *fakeLanControlBackend) PostNetworkWebhookV2(ctx context.Context, r *http.Request) (*models.AdvancedNetworkResponse, error) {
	backend.recordRequest("postNetworkWebhookV2", r)
	return &models.AdvancedNetworkResponse{Result: &models.AdvancedNetworkResult{}}, backend.err
}
func (backend *fakeLanControlBackend) GetPolicyBundleV2(ctx context.Context) (*models.PolicyBundleResponse, error) {
	backend.record("getPolicyBundleV2")
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{}}, backend.err
}
func (backend *fakeLanControlBackend) PostPolicyImportPlanV2(ctx context.Context, r *http.Request) (*models.PolicyBundleResponse, error) {
	backend.recordRequest("postPolicyImportPlanV2", r)
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{}}, backend.err
}
func (backend *fakeLanControlBackend) PostPolicyImportApplyV2(ctx context.Context, r *http.Request) (*models.PolicyBundleResponse, error) {
	backend.recordRequest("postPolicyImportApplyV2", r)
	return &models.PolicyBundleResponse{Result: &models.PolicyBundleResult{}}, backend.err
}

func (backend *fakeLanControlBackend) GetDevicePolicyV2(ctx context.Context, r *http.Request) (*models.DevicePolicyResponse, error) {
	backend.recordRequest("getDevicePolicyV2", r)
	return &models.DevicePolicyResponse{Result: &models.DevicePolicyResult{Policy: &models.DevicePolicy{DeviceID: r.URL.Query().Get("deviceId")}}}, nil
}

func (backend *fakeLanControlBackend) PostDevicePolicyV2(ctx context.Context, r *http.Request) (*models.DevicePolicyResponse, error) {
	backend.recordRequest("postDevicePolicyV2", r)
	return &models.DevicePolicyResponse{Result: &models.DevicePolicyResult{Changed: true}}, nil
}

func (backend *fakeLanControlBackend) GetDevicePolicyRulesV2(ctx context.Context) (*models.DevicePolicyRulesResponse, error) {
	backend.record("devicePolicyRulesV2")
	return &models.DevicePolicyRulesResponse{Result: &models.DevicePolicyRulesResult{}}, nil
}

func (backend *fakeLanControlBackend) GetLanListStaticDevices(ctx context.Context) (*models.LANCtrlStaticAssignedResponse, error) {
	backend.record("listStaticDevices")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.LANCtrlStaticAssignedResponse{
		Result: []*models.LANStaticAssigned{{AssignedIP: "192.168.1.10"}},
	}, nil
}

func (backend *fakeLanControlBackend) GetLanListSpeedLimitedDevices(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error) {
	backend.record("listSpeedLimitedDevices")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.LANCtrlSpeedLimitResponse{
		Result: []*models.LANCtrlSpeedLimitItem{{IP: "192.168.1.20"}},
	}, nil
}

func (backend *fakeLanControlBackend) normalResponse() (*models.JSONResponse, error) {
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.JSONResponse{}, nil
}

func TestRegisterLanControlRoutesMapsRoutesToBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		wantCall string
	}{
		{
			name:     "speeds for devices",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/speedsForDevices/",
			wantCall: "speedsForDevices",
		},
		{
			name:     "device inventory v2",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/devices/",
			wantCall: "deviceInventoryV2",
		},
		{
			name:     "add manual device v2",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/devices/",
			body:     `{"mac":"AA:BB:CC:DD:EE:FF","alias":"Office PC"}`,
			wantCall: "postDeviceInventoryV2",
		},
		{
			name:     "get device classification v2",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-classification/?deviceId=mac%3Aaa",
			wantCall: "getDeviceClassificationV2",
		},
		{
			name:     "post device classification v2",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-classification/",
			body:     `{"deviceId":"mac:aa","action":"set","category":"computer"}`,
			wantCall: "postDeviceClassificationV2",
		},
		{
			name:     "get device profile v2",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-profile/?deviceId=mac%3Aaa",
			wantCall: "getDeviceProfileV2",
		},
		{
			name:     "post device profile v2",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-profile/",
			body:     `{"deviceId":"mac:aa","action":"patch","alias":"客厅电视"}`,
			wantCall: "postDeviceProfileV2",
		},
		{
			name:     "device traffic v2",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-traffic/",
			wantCall: "deviceTrafficV2",
		},
		{
			name:     "device runtime diagnostics v2",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-diagnostics/",
			wantCall: "deviceRuntimeDiagnosticsV2",
		},
		{name: "gateway targets v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/gateway-targets/", wantCall: "gatewayTargetsV2"},
		{name: "gateway assignment plan v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/gateway-assignment/plan/", body: `{"action":"assign","deviceId":"mac:aa","targetId":"self"}`, wantCall: "gatewayAssignmentPlanV2"},
		{name: "gateway assignment apply v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/gateway-assignment/apply/", body: `{"action":"assign","deviceId":"mac:aa","targetId":"self"}`, wantCall: "gatewayAssignmentApplyV2"},
		{name: "gateway target plan v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/gateway-target/plan/", body: `{"action":"create","name":"旁路由","kind":"bypass","gateway":"192.168.1.2"}`, wantCall: "gatewayTargetPlanV2"},
		{name: "gateway target apply v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/gateway-target/apply/", body: `{"action":"delete","targetId":"custom:1","replacementTargetId":"self"}`, wantCall: "gatewayTargetApplyV2"},
		{name: "device network plan v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/plan/", body: `{"deviceId":"mac:aa","targetId":"self","static":{"enabled":false}}`, wantCall: "planDeviceNetworkPolicyV2"},
		{name: "device network apply v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/apply/", body: `{"deviceId":"mac:aa","targetId":"self","static":{"enabled":false}}`, wantCall: "applyDeviceNetworkPolicyV2"},
		{name: "device restrictions plan v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/device-restrictions/plan/", body: `{"deviceId":"mac:aa","kind":"access","access":{"networkAccess":false}}`, wantCall: "planDeviceRestrictionsV2"},
		{name: "device restrictions apply v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/device-restrictions/apply/", body: `{"deviceId":"mac:aa","kind":"access","access":{"networkAccess":false}}`, wantCall: "applyDeviceRestrictionsV2"},
		{name: "gateway references v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/gateway-target-references/?targetId=self", wantCall: "gatewayReferencesV2"},
		{name: "get device network policy v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/?deviceId=mac%3Aaa", wantCall: "getDeviceNetworkPolicyV2"},
		{name: "post device network policy v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/", body: `{"deviceId":"mac:aa","targetId":"default","static":{"enabled":false}}`, wantCall: "postDeviceNetworkPolicyV2"},
		{name: "get floating gateway v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/", wantCall: "getFloatingGatewayV2"},
		{name: "plan floating gateway v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/plan/", body: `{"config":{"enabled":false}}`, wantCall: "postFloatingGatewayPlanV2"},
		{name: "apply floating gateway v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/apply/", body: `{"config":{"enabled":false}}`, wantCall: "postFloatingGatewayApplyV2"},
		{name: "floating gateway drill plan v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/drill-plan/", wantCall: "getFloatingGatewayDrillPlanV2"},
		{name: "network rules v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/network-rules/", wantCall: "getNetworkRulesV2"},
		{name: "plan network rules v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/network-rules/plan/", body: `{"action":"delete","ruleIds":["static:1"]}`, wantCall: "postNetworkRulesPlanV2"},
		{name: "apply network rules v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/network-rules/apply/", body: `{"action":"delete","ruleIds":["static:1"]}`, wantCall: "postNetworkRulesApplyV2"},
		{name: "plan LAN device migration v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/migration/plan/", wantCall: "getLanDeviceMigrationPlanV2"},
		{name: "apply LAN device migration v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/migration/apply/", body: `{"expectedVersion":"v1"}`, wantCall: "postLanDeviceMigrationApplyV2"},
		{name: "plan rate limit migration v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/rate-limit-migration/plan/", wantCall: "getRateLimitMigrationPlanV2"},
		{name: "apply rate limit migration v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/rate-limit-migration/apply/", body: `{"expectedVersion":"v1"}`, wantCall: "postRateLimitMigrationApplyV2"},
		{name: "rollback rate limit migration v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/rate-limit-migration/rollback/", body: `{"migrationId":"m1"}`, wantCall: "postRateLimitMigrationRollbackV2"},
		{name: "plan capability install v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/capability-action/plan/", body: `{"capabilityKey":"device_speed_limit","action":"install","draftToken":"draft-1"}`, wantCall: "postCapabilityActionPlanV2"},
		{name: "apply capability install v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/capability-action/apply/", body: `{"capabilityKey":"device_speed_limit","action":"install","draftToken":"draft-1","confirm":true}`, wantCall: "postCapabilityActionApplyV2"},
		{name: "get router context v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/router-context/?lan=guest_20", wantCall: "getRouterContextV2"},
		{name: "get LAN DHCP settings v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/lan-dhcp-settings/", wantCall: "getLanDHCPSettingsV2"},
		{name: "plan LAN DHCP settings v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/lan-dhcp-settings/plan/", body: `{"settings":{"enabled":true,"poolStart":"192.168.1.100","poolEnd":"192.168.1.200","leaseTime":"12h","defaultTargetId":"self"}}`, wantCall: "planLanDHCPSettingsV2"},
		{name: "apply LAN DHCP settings v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/lan-dhcp-settings/apply/", body: `{"settings":{"enabled":true,"poolStart":"192.168.1.100","poolEnd":"192.168.1.200","leaseTime":"12h","defaultTargetId":"self"}}`, wantCall: "applyLanDHCPSettingsV2"},
		{name: "get device groups v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/device-groups/", wantCall: "getDeviceGroupsV2"},
		{name: "post device groups v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/device-groups/", body: `{"action":"delete_group","groupId":"kids"}`, wantCall: "postDeviceGroupsV2"},
		{name: "get traffic insights v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/traffic-insights/?deviceId=mac%3Aaa&range=month", wantCall: "getTrafficInsightsV2"},
		{name: "post traffic quota v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/traffic-quota/", body: `{"deviceId":"mac:aa","enabled":true,"period":"monthly","limitBytes":1024,"action":"notify"}`, wantCall: "postTrafficQuotaV2"},
		{name: "advanced network v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/advanced-network/?deviceId=mac%3Aaa", wantCall: "getAdvancedNetworkV2"},
		{name: "management probe v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/management-probe/", body: `{"deviceId":"mac:aa","address":"192.168.1.2","scheme":"http","port":80}`, wantCall: "postManagementProbeV2"},
		{name: "network webhook v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/network-webhook/", body: `{"enabled":false}`, wantCall: "postNetworkWebhookV2"},
		{name: "policy bundle v2", method: http.MethodGet, path: "/cgi-bin/luci/istore/lanctrl/v2/policy-bundle/", wantCall: "getPolicyBundleV2"},
		{name: "policy import plan v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/policy-import/plan/", body: `{"bundle":{"schemaVersion":1}}`, wantCall: "postPolicyImportPlanV2"},
		{name: "policy import apply v2", method: http.MethodPost, path: "/cgi-bin/luci/istore/lanctrl/v2/policy-import/apply/", body: `{"bundle":{"schemaVersion":1},"confirmed":true}`, wantCall: "postPolicyImportApplyV2"},
		{
			name:     "get device policy v2",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-policy/?deviceId=mac%3Aaa",
			wantCall: "getDevicePolicyV2",
		},
		{
			name:     "post device policy v2",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-policy/",
			body:     `{"deviceId":"mac:aa","kind":"access","access":{"networkAccess":false}}`,
			wantCall: "postDevicePolicyV2",
		},
		{
			name:     "device policy rules v2",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/device-policy/rules/",
			wantCall: "devicePolicyRulesV2",
		},
		{
			name:     "speeds for one device",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/speedsForOneDevice/",
			body:     `{"ip":"192.168.1.2"}`,
			wantCall: "speedsForOneDevice",
		},
		{
			name:     "dhcp tags config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/dhcpTagsConfig/",
			body:     `{"action":"add","tagName":"guest"}`,
			wantCall: "dhcpTagsConfig",
		},
		{
			name:     "dhcp gateway config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/dhcpGatewayConfig/",
			body:     `{"dhcpEnabled":true,"dhcpGateway":"192.168.1.1"}`,
			wantCall: "dhcpGatewayConfig",
		},
		{
			name:     "speed limit config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/speedLimitConfig/",
			body:     `{"action":"add","mac":"aa:bb:cc:dd:ee:ff"}`,
			wantCall: "speedLimitConfig",
		},
		{
			name:     "enable speed limit",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/enableSpeedLimit/",
			body:     `{"enabled":true}`,
			wantCall: "enableSpeedLimit",
		},
		{
			name:     "plan rate limit settings",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/rate-limit-settings/plan/",
			body:     `{"settings":{"enabled":true,"uploadSpeed":100,"downloadSpeed":1000}}`,
			wantCall: "rateLimitSettingsPlanV2",
		},
		{
			name:     "apply rate limit settings",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/v2/rate-limit-settings/apply/",
			body:     `{"settings":{"enabled":true,"uploadSpeed":100,"downloadSpeed":1000}}`,
			wantCall: "rateLimitSettingsApplyV2",
		},
		{
			name:     "enable float gateway",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/enableFloatGateway/",
			body:     `{"enabled":true}`,
			wantCall: "enableFloatGateway",
		},
		{
			name:     "static device config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/staticDeviceConfig/",
			body:     `{"action":"add","assignedMac":"AA:BB:CC:DD:EE:FF"}`,
			wantCall: "staticDeviceConfig",
		},
		{
			name:     "global configs",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/globalConfigs/",
			wantCall: "globalConfigs",
		},
		{
			name:     "list devices",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/listDevices/",
			wantCall: "listDevices",
		},
		{
			name:     "list static devices",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/listStaticDevices/",
			wantCall: "listStaticDevices",
		},
		{
			name:     "list speed limited devices",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/listSpeedLimitedDevices/",
			wantCall: "listSpeedLimitedDevices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeLanControlBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestLanControlRoute(t, router, tt.method, tt.path, tt.body, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
		})
	}
}

func TestRegisterLanControlRoutesPostPassesOriginalRequestPath(t *testing.T) {
	backend := &fakeLanControlBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	const path = "/cgi-bin/luci/istore/lanctrl/staticDeviceConfig/"
	requestLanControlRoute(t, router, http.MethodPost, path, `{"action":"add"}`, true)

	if len(backend.requestPaths) != 1 || backend.requestPaths[0] != path {
		t.Fatalf("expected request path %q, got %#v", path, backend.requestPaths)
	}
}

func TestRegisterLanControlRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeLanControlBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestLanControlRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/lanctrl/listDevices/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireLanControlEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterLanControlRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeLanControlBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestLanControlRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/lanctrl/listDevices/", "", true)

	if len(backend.calls) != 1 || backend.calls[0] != "listDevices" {
		t.Fatalf("expected listDevices backend call, got %#v", backend.calls)
	}
	requireLanControlEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestLanControlRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if withSID {
		req.Header.Set("X-Forwarded-Sid", "sid-1")
	}
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s expected status 200, got %d", method, path, rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func requireLanControlEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

package lancontrol

import (
	"context"
	"net/http"

	"github.com/istoreos/quickstart/backend/internal/httpapi"
	"github.com/istoreos/quickstart/backend/models"
	"github.com/julienschmidt/httprouter"
)

type Backend interface {
	GetSpeedsForAllDevice(ctx context.Context, r *http.Request) (*models.DeviceSpeedStatsResponse, error)
	GetSpeedsForOneDevice(ctx context.Context, r *http.Request) (*models.NetworkStatisticsResponse, error)
	PostLanDhcpTagsConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error)
	PostLanDhcpGatewayConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error)
	PostLanSpeedLimitConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error)
	PostLanEnableSpeedLimit(ctx context.Context, r *http.Request) (*models.JSONResponse, error)
	PostLanEnableFloatGateway(ctx context.Context, r *http.Request) (*models.JSONResponse, error)
	PostLanStaticDeviceConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error)
	GetLanGlobalConfigs(ctx context.Context) (*models.LANCtrlGlobalConfigResponse, error)
	GetLanListDevices(ctx context.Context) (*models.LANDeviceResponse, error)
	GetDeviceInventoryV2(ctx context.Context) (*models.DeviceInventoryResponse, error)
	PostDeviceInventoryV2(ctx context.Context, r *http.Request) (*models.DeviceInventoryResponse, error)
	GetDeviceClassificationV2(ctx context.Context, r *http.Request) (*models.DeviceClassificationResponse, error)
	PostDeviceClassificationV2(ctx context.Context, r *http.Request) (*models.DeviceClassificationResponse, error)
	GetDeviceProfileV2(ctx context.Context, r *http.Request) (*models.DeviceProfileResponse, error)
	PostDeviceProfileV2(ctx context.Context, r *http.Request) (*models.DeviceProfileResponse, error)
	GetDeviceTrafficV2(ctx context.Context) (*models.DeviceTrafficResponse, error)
	GetDeviceRuntimeDiagnosticsV2(ctx context.Context) (*models.DeviceRuntimeDiagnosticsResponse, error)
	GetGatewayTargetsV2(ctx context.Context) (*models.GatewayTargetListResponse, error)
	PostGatewayAssignmentPlanV2(ctx context.Context, r *http.Request) (*models.GatewayAssignmentPlanResponse, error)
	PostGatewayAssignmentApplyV2(ctx context.Context, r *http.Request) (*models.GatewayAssignmentApplyResponse, error)
	GetGatewayReferencesV2(ctx context.Context, r *http.Request) (*models.GatewayReferencesResponse, error)
	GetDeviceNetworkPolicyV2(ctx context.Context, r *http.Request) (*models.DeviceNetworkPolicyResponse, error)
	PostDeviceNetworkPolicyV2(ctx context.Context, r *http.Request) (*models.DeviceNetworkPolicyResponse, error)
	GetFloatingGatewayV2(ctx context.Context) (*models.FloatingGatewayResponse, error)
	PostFloatingGatewayPlanV2(ctx context.Context, r *http.Request) (*models.FloatingGatewayResponse, error)
	PostFloatingGatewayApplyV2(ctx context.Context, r *http.Request) (*models.FloatingGatewayResponse, error)
	GetFloatingGatewayDrillPlanV2(ctx context.Context) (*models.FloatingGatewayDrillResponse, error)
	GetNetworkRulesV2(ctx context.Context) (*models.NetworkRulesResponse, error)
	PostNetworkRulesPlanV2(ctx context.Context, r *http.Request) (*models.NetworkRulesBulkResponse, error)
	PostNetworkRulesApplyV2(ctx context.Context, r *http.Request) (*models.NetworkRulesBulkResponse, error)
	GetLanDeviceMigrationPlanV2(ctx context.Context) (*models.LanDeviceMigrationResponse, error)
	PostLanDeviceMigrationApplyV2(ctx context.Context, r *http.Request) (*models.LanDeviceMigrationResponse, error)
	PostCapabilityActionPlanV2(ctx context.Context, r *http.Request) (*models.CapabilityActionResponse, error)
	PostCapabilityActionApplyV2(ctx context.Context, r *http.Request) (*models.CapabilityActionResponse, error)
	GetRouterContextV2(ctx context.Context, r *http.Request) (*models.RouterContextResponse, error)
	GetDeviceGroupsV2(ctx context.Context) (*models.DeviceGroupsResponse, error)
	PostDeviceGroupsV2(ctx context.Context, r *http.Request) (*models.DeviceGroupsResponse, error)
	GetTrafficInsightsV2(ctx context.Context, r *http.Request) (*models.TrafficInsightsResponse, error)
	PostTrafficQuotaV2(ctx context.Context, r *http.Request) (*models.TrafficInsightsResponse, error)
	GetAdvancedNetworkV2(ctx context.Context, r *http.Request) (*models.AdvancedNetworkResponse, error)
	PostManagementProbeV2(ctx context.Context, r *http.Request) (*models.ManagementProbeResponse, error)
	PostNetworkWebhookV2(ctx context.Context, r *http.Request) (*models.AdvancedNetworkResponse, error)
	GetPolicyBundleV2(ctx context.Context) (*models.PolicyBundleResponse, error)
	PostPolicyImportPlanV2(ctx context.Context, r *http.Request) (*models.PolicyBundleResponse, error)
	PostPolicyImportApplyV2(ctx context.Context, r *http.Request) (*models.PolicyBundleResponse, error)
	GetDevicePolicyV2(ctx context.Context, r *http.Request) (*models.DevicePolicyResponse, error)
	PostDevicePolicyV2(ctx context.Context, r *http.Request) (*models.DevicePolicyResponse, error)
	GetDevicePolicyRulesV2(ctx context.Context) (*models.DevicePolicyRulesResponse, error)
	GetLanListStaticDevices(ctx context.Context) (*models.LANCtrlStaticAssignedResponse, error)
	GetLanListSpeedLimitedDevices(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/speedsForDevices/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSpeedsForAllDevice(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/speedsForOneDevice/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSpeedsForOneDevice(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/dhcpTagsConfig/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostLanDhcpTagsConfig(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/dhcpGatewayConfig/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostLanDhcpGatewayConfig(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/speedLimitConfig/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostLanSpeedLimitConfig(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/enableSpeedLimit/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostLanEnableSpeedLimit(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/enableFloatGateway/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostLanEnableFloatGateway(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/staticDeviceConfig/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostLanStaticDeviceConfig(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/globalConfigs/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLanGlobalConfigs(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/listDevices/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLanListDevices(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/devices/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceInventoryV2(ctx)
	})
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/devices/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostDeviceInventoryV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-classification/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceClassificationV2(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-classification/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostDeviceClassificationV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-profile/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceProfileV2(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-profile/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostDeviceProfileV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-traffic/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceTrafficV2(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-diagnostics/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceRuntimeDiagnosticsV2(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/gateway-targets/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGatewayTargetsV2(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/gateway-assignment/plan/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGatewayAssignmentPlanV2(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/gateway-assignment/apply/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGatewayAssignmentApplyV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/gateway-target-references/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGatewayReferencesV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceNetworkPolicyV2(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-network-policy/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostDeviceNetworkPolicyV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetFloatingGatewayV2(ctx)
	})
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/plan/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostFloatingGatewayPlanV2(ctx, r)
	})
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/apply/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostFloatingGatewayApplyV2(ctx, r)
	})
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/floating-gateway/drill-plan/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetFloatingGatewayDrillPlanV2(ctx)
	})
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/network-rules/", func(ctx context.Context, r *http.Request) (any, error) { return backend.GetNetworkRulesV2(ctx) })
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/network-rules/plan/", func(ctx context.Context, r *http.Request) (any, error) { return backend.PostNetworkRulesPlanV2(ctx, r) })
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/network-rules/apply/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNetworkRulesApplyV2(ctx, r)
	})
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/migration/plan/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLanDeviceMigrationPlanV2(ctx)
	})
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/migration/apply/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostLanDeviceMigrationApplyV2(ctx, r)
	})
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/capability-action/plan/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostCapabilityActionPlanV2(ctx, r)
	})
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/capability-action/apply/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostCapabilityActionApplyV2(ctx, r)
	})
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/router-context/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetRouterContextV2(ctx, r)
	})
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-groups/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceGroupsV2(ctx)
	})
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-groups/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostDeviceGroupsV2(ctx, r)
	})
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/traffic-insights/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetTrafficInsightsV2(ctx, r)
	})
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/traffic-quota/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostTrafficQuotaV2(ctx, r)
	})
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/advanced-network/", func(ctx context.Context, r *http.Request) (any, error) { return backend.GetAdvancedNetworkV2(ctx, r) })
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/management-probe/", func(ctx context.Context, r *http.Request) (any, error) { return backend.PostManagementProbeV2(ctx, r) })
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/network-webhook/", func(ctx context.Context, r *http.Request) (any, error) { return backend.PostNetworkWebhookV2(ctx, r) })
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/policy-bundle/", func(ctx context.Context, r *http.Request) (any, error) { return backend.GetPolicyBundleV2(ctx) })
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/policy-import/plan/", func(ctx context.Context, r *http.Request) (any, error) { return backend.PostPolicyImportPlanV2(ctx, r) })
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/policy-import/apply/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostPolicyImportApplyV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-policy/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDevicePolicyV2(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-policy/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostDevicePolicyV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-policy/rules/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDevicePolicyRulesV2(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/listStaticDevices/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLanListStaticDevices(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/listSpeedLimitedDevices/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLanListSpeedLimitedDevices(ctx)
	})
}

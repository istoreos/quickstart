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
	GetDeviceClassificationV2(ctx context.Context, r *http.Request) (*models.DeviceClassificationResponse, error)
	PostDeviceClassificationV2(ctx context.Context, r *http.Request) (*models.DeviceClassificationResponse, error)
	GetDeviceTrafficV2(ctx context.Context) (*models.DeviceTrafficResponse, error)
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

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-classification/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceClassificationV2(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-classification/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostDeviceClassificationV2(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/v2/device-traffic/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetDeviceTrafficV2(ctx)
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

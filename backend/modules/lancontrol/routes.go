package lancontrol

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
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

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/listStaticDevices/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLanListStaticDevices(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/lanctrl/listSpeedLimitedDevices/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLanListSpeedLimitedDevices(ctx)
	})
}

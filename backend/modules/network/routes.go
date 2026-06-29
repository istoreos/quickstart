package network

import (
	"context"
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/interfacewrite"
)

type Backend interface {
	GetNetworkStatistic(ctx context.Context) (*models.NetworkStatisticsResponse, error)
	GetNetworkStatus(ctx context.Context, setupFinish bool) (*models.NetworkStatusResponse, error)
	GetNetworkDeviceList(ctx context.Context) (*models.DeviceListResponse, error)
	EnableNetworkHomebox(ctx context.Context) (*models.NetworkHomeBoxEnableResponse, error)
	GetNetworkInterfaceStatus(ctx context.Context) (*models.NetworkInterfaceStatusResponse, error)
	CheckNetworkPublicAddress(ctx context.Context, ipVersion string) (*models.NetworkCheckPublicNetResponse, error)
	GetNetworkPortList(ctx context.Context) (*models.NetworkPortListResponse, error)
	GetNetworkInterfaceConfig(ctx context.Context) (*models.NetworkInterfaceGetConfigResponse, error)
	SetNetworkInterfaceConfig(ctx context.Context, input interfacewrite.Input) (*models.SDKNormalResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.GetJSONAliases(router, []string{
		"/cgi-bin/luci/istore/network/statistics/",
		"/cgi-bin/luci/istore/u/network/statistics/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNetworkStatistic(ctx)
	})

	httpapi.GetJSONAliases(router, []string{
		"/cgi-bin/luci/istore/network/status/",
		"/cgi-bin/luci/istore/u/network/status/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNetworkStatus(ctx, false)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/network/setup/finish/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNetworkStatus(ctx, true)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/network/device/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNetworkDeviceList(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/network/homebox/enable", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.EnableNetworkHomebox(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/network/interface/status/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNetworkInterfaceStatus(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/network/checkPublicNet/", func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeNetworkJSON[models.NetworkCheckPublicNetRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.CheckNetworkPublicAddress(ctx, req.IPVersion)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/network/port/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNetworkPortList(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/network/interface/config/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNetworkInterfaceConfig(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/network/interface/config/", func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeNetworkJSON[models.NetworkInterfaceSetConfigRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.SetNetworkInterfaceConfig(ctx, interfacewrite.Input{
			Configs: req.Configs,
		})
	})
}

func decodeNetworkJSON[T any](r *http.Request) (T, error) {
	req, err := httpapi.DecodeJSON[T](r)
	if err != nil {
		return req, errors.New("请求解析失败")
	}
	return req, nil
}

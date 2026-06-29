package wireless

import (
	"context"
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	WirelessListIfaces(ctx context.Context) (*models.WirelessListIfaceResponse, error)
	WirelessEnableIface(ctx context.Context, req models.WirelessEnableIfaceRequest) error
	WirelessSetDevicePower(ctx context.Context, req models.WirelessSetDevicePowerRequest) error
	WirelessEditIface(ctx context.Context, req models.WirelessIfaceInfo) error
	WirelessQuickSetupIface(ctx context.Context, req models.WirelessQuickSetupRequest) error
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/wireless/list-iface/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.WirelessListIfaces(ctx)
	})

	postWirelessAction[models.WirelessEnableIfaceRequest](router, "/cgi-bin/luci/istore/wireless/enable-iface/", func(ctx context.Context, req models.WirelessEnableIfaceRequest) error {
		return backend.WirelessEnableIface(ctx, req)
	})

	postWirelessAction[models.WirelessSetDevicePowerRequest](router, "/cgi-bin/luci/istore/wireless/set-device-power/", func(ctx context.Context, req models.WirelessSetDevicePowerRequest) error {
		return backend.WirelessSetDevicePower(ctx, req)
	})

	postWirelessAction[models.WirelessIfaceInfo](router, "/cgi-bin/luci/istore/wireless/edit-iface/", func(ctx context.Context, req models.WirelessIfaceInfo) error {
		return backend.WirelessEditIface(ctx, req)
	})

	postWirelessAction[models.WirelessQuickSetupRequest](router, "/cgi-bin/luci/istore/wireless/setup/", func(ctx context.Context, req models.WirelessQuickSetupRequest) error {
		return backend.WirelessQuickSetupIface(ctx, req)
	})
}

func postWirelessAction[T any](router *httprouter.Router, path string, action func(context.Context, T) error) {
	httpapi.PostJSON(router, path, func(ctx context.Context, r *http.Request) (any, error) {
		req, err := httpapi.DecodeJSON[T](r)
		if err != nil {
			return nil, errors.New("Invalid request")
		}
		if err := action(ctx, req); err != nil {
			return nil, err
		}
		success := models.ResponseSuccess(0)
		return &models.SDKNormalResponse{Success: &success}, nil
	})
}

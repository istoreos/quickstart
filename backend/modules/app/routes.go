package app

import (
	"context"
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	CheckApp(ctx context.Context, req models.AppCheckRequest) (*models.AppCheckResponse, error)
	InstallAppPackage(ctx context.Context, req models.AppInstallRequest) (*models.SDKNormalResponse, error)
	ListInstalledApps(ctx context.Context) (models.AppInstalledListResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/app/check/", func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeAppJSON[models.AppCheckRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.CheckApp(ctx, req)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/app/install/", func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeAppJSON[models.AppInstallRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.InstallAppPackage(ctx, req)
	})

	router.GET("/cgi-bin/luci/istore/app/install-list/", httpapi.AuthenticatedJSON(func(ctx context.Context, r *http.Request) (any, error) {
		resp, err := backend.ListInstalledApps(ctx)
		if err != nil {
			return []*models.AppInstalled{}, nil
		}
		return resp, nil
	}))
}

func decodeAppJSON[T any](r *http.Request) (T, error) {
	req, err := httpapi.DecodeJSON[T](r)
	if err != nil {
		return req, errors.New("请求解析失败")
	}
	return req, nil
}

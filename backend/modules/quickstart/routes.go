package quickstart

import (
	"context"
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	GetQuickstartConfig(ctx context.Context, req models.QuickstartGetConfigRequest) (*models.QuickstartConfigResponse, error)
	SetQuickstartConfig(ctx context.Context, req models.QuickstartConfigRequest) (*models.SDKNormalResponse, error)
	DeleteQuickstartConfig(ctx context.Context, req models.QuickstartDeleteConfigRequest) (*models.SDKNormalResponse, error)
}

func decodeQuickstartJSON[T any](r *http.Request) (T, error) {
	req, err := httpapi.DecodeJSON[T](r)
	if err != nil {
		return req, errors.New("请求解析失败")
	}
	return req, nil
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.PostJSONAliases(router, []string{
		"/cgi-bin/luci/istore/quickstart/get/",
		"/cgi-bin/luci/istore/u/quickstart/get/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeQuickstartJSON[models.QuickstartGetConfigRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.GetQuickstartConfig(ctx, req)
	})

	httpapi.PostJSONAliases(router, []string{
		"/cgi-bin/luci/istore/quickstart/set/",
		"/cgi-bin/luci/istore/u/quickstart/set/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeQuickstartJSON[models.QuickstartConfigRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.SetQuickstartConfig(ctx, req)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/quickstart/delete/", func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeQuickstartJSON[models.QuickstartDeleteConfigRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.DeleteQuickstartConfig(ctx, req)
	})
}

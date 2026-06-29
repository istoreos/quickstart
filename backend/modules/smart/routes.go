package smart

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	GetSmartList(ctx context.Context) (*models.SmartListResponse, error)
	GetSmartLog(ctx context.Context) (*models.SmartLogResponse, error)
	GetSmartConfig(ctx context.Context) (*models.SmartConfigResponse, error)
	PostSmartConfig(ctx context.Context, req models.SmartConfigRequest) (*models.SmartConfigResponse, error)
	PostSmartTest(ctx context.Context, req models.SmartTestRequest) (*models.SmartTestResponse, error)
	PostSmartTestResult(ctx context.Context, req models.SmartTestResultRequest) (*models.SmartTestResultResponse, error)
	PostSmartAttributeResult(ctx context.Context, req models.SmartAttributeResultRequest) (*models.SmartAttributeResultResponse, error)
	PostSmartExtendResult(ctx context.Context, req models.SmartExtendResultRequest) (*models.SmartExtendResultResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.GetJSONAliases(router, []string{
		"/cgi-bin/luci/istore/smart/list/",
		"/cgi-bin/luci/istore/u/smart/list/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSmartList(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/smart/log/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSmartLog(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/smart/config/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSmartConfig(ctx)
	})

	postJSONDecoded[models.SmartConfigRequest](router, "/cgi-bin/luci/istore/smart/config/", func(ctx context.Context, req models.SmartConfigRequest) (any, error) {
		return backend.PostSmartConfig(ctx, req)
	})

	postJSONDecodedAliases[models.SmartTestRequest](router, []string{
		"/cgi-bin/luci/istore/smart/test/",
		"/cgi-bin/luci/istore/u/smart/test/",
	}, func(ctx context.Context, req models.SmartTestRequest) (any, error) {
		return backend.PostSmartTest(ctx, req)
	})

	postJSONDecoded[models.SmartTestResultRequest](router, "/cgi-bin/luci/istore/smart/test/result/", func(ctx context.Context, req models.SmartTestResultRequest) (any, error) {
		return backend.PostSmartTestResult(ctx, req)
	})

	postJSONDecoded[models.SmartAttributeResultRequest](router, "/cgi-bin/luci/istore/smart/attribute/result/", func(ctx context.Context, req models.SmartAttributeResultRequest) (any, error) {
		return backend.PostSmartAttributeResult(ctx, req)
	})

	postJSONDecoded[models.SmartExtendResultRequest](router, "/cgi-bin/luci/istore/smart/extend/result/", func(ctx context.Context, req models.SmartExtendResultRequest) (any, error) {
		return backend.PostSmartExtendResult(ctx, req)
	})
}

type decodedJSONHandler[T any] func(context.Context, T) (any, error)

func postJSONDecoded[T any](router *httprouter.Router, path string, fn decodedJSONHandler[T]) {
	httpapi.PostJSON(router, path, func(ctx context.Context, r *http.Request) (any, error) {
		req, err := httpapi.DecodeJSON[T](r)
		if err != nil {
			return nil, err
		}
		return fn(ctx, req)
	})
}

func postJSONDecodedAliases[T any](router *httprouter.Router, paths []string, fn decodedJSONHandler[T]) {
	for _, path := range paths {
		postJSONDecoded[T](router, path, fn)
	}
}

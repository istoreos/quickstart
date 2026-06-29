package share

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	GetShareUserList(ctx context.Context) (*models.ShareUserListResponse, error)
	PostShareUserCreate(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error)
	PostShareUserUpdate(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error)
	PostShareUserDelete(ctx context.Context, req models.ShareUserDeleteRequest) (*models.SDKNormalResponse, error)
	GetShareServiceList(ctx context.Context) (*models.ShareServiceListResponse, error)
	PostShareServiceCreate(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error)
	PostShareServiceUpdate(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error)
	PostShareServiceDelete(ctx context.Context, req models.ShareServicDeleteRequest) (*models.SDKNormalResponse, error)
	GetShareWebdavConfig(ctx context.Context) (*models.ShareProtocolWebdavResponse, error)
	PostShareWebdavConfig(ctx context.Context, req models.ShareProtocolWebdavConfig) (*models.SDKNormalResponse, error)
	GetShareSambaConfig(ctx context.Context) (*models.ShareProtocolSambaResponse, error)
	PostShareSambaConfig(ctx context.Context, req models.ShareProtocolSambaConfig) (*models.SDKNormalResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/share/user/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetShareUserList(ctx)
	})

	postDecodedJSON[models.ShareUserCreateRequest](router, "/cgi-bin/luci/istore/share/user/create/", func(ctx context.Context, req models.ShareUserCreateRequest) (any, error) {
		return backend.PostShareUserCreate(ctx, req)
	})

	postDecodedJSON[models.ShareUserCreateRequest](router, "/cgi-bin/luci/istore/share/user/update/", func(ctx context.Context, req models.ShareUserCreateRequest) (any, error) {
		return backend.PostShareUserUpdate(ctx, req)
	})

	postDecodedJSON[models.ShareUserDeleteRequest](router, "/cgi-bin/luci/istore/share/user/delete/", func(ctx context.Context, req models.ShareUserDeleteRequest) (any, error) {
		return backend.PostShareUserDelete(ctx, req)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/share/service/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetShareServiceList(ctx)
	})

	postDecodedJSON[models.ShareServiceCreateRequest](router, "/cgi-bin/luci/istore/share/service/create/", func(ctx context.Context, req models.ShareServiceCreateRequest) (any, error) {
		return backend.PostShareServiceCreate(ctx, req)
	})

	postDecodedJSON[models.ShareServiceCreateRequest](router, "/cgi-bin/luci/istore/share/service/update/", func(ctx context.Context, req models.ShareServiceCreateRequest) (any, error) {
		return backend.PostShareServiceUpdate(ctx, req)
	})

	postDecodedJSON[models.ShareServicDeleteRequest](router, "/cgi-bin/luci/istore/share/service/delete/", func(ctx context.Context, req models.ShareServicDeleteRequest) (any, error) {
		return backend.PostShareServiceDelete(ctx, req)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/share/protocol/webdav/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetShareWebdavConfig(ctx)
	})

	postDecodedJSON[models.ShareProtocolWebdavConfig](router, "/cgi-bin/luci/istore/share/protocol/webdav/", func(ctx context.Context, req models.ShareProtocolWebdavConfig) (any, error) {
		return backend.PostShareWebdavConfig(ctx, req)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/share/protocol/samba/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetShareSambaConfig(ctx)
	})

	postDecodedJSON[models.ShareProtocolSambaConfig](router, "/cgi-bin/luci/istore/share/protocol/samba/", func(ctx context.Context, req models.ShareProtocolSambaConfig) (any, error) {
		return backend.PostShareSambaConfig(ctx, req)
	})
}

func postDecodedJSON[T any](router *httprouter.Router, path string, fn func(context.Context, T) (any, error)) {
	httpapi.PostJSON(router, path, func(ctx context.Context, r *http.Request) (any, error) {
		req, err := httpapi.DecodeJSON[T](r)
		if err != nil {
			return nil, err
		}
		return fn(ctx, req)
	})
}

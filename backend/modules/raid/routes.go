package raid

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	PostRaidCreate(ctx context.Context, r *http.Request) (*models.NasDiskPartitionFormatResponse, error)
	PostRaidDelete(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostRaidAdd(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostRaidRemove(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostRaidDetail(ctx context.Context, r *http.Request) (*models.RaidDetailResponse, error)
	PostRaidRecover(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	GetRaidList(ctx context.Context) (*models.RaidListResponse, error)
	PostRaidAutoFix(ctx context.Context) (*models.SDKNormalResponse, error)
	GetRaidCreateList(ctx context.Context) (*models.RaidCreateListResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/raid/create/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostRaidCreate(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/raid/delete/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostRaidDelete(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/raid/add/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostRaidAdd(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/raid/remove/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostRaidRemove(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/raid/detail/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostRaidDetail(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/raid/recover/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostRaidRecover(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/raid/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetRaidList(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/raid/autofix/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostRaidAutoFix(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/raid/create/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetRaidCreateList(ctx)
	})
}

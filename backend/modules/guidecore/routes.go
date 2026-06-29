package guidecore

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	GuideNeedSetup(ctx context.Context, r *http.Request) (*models.GuideNeedSetupResponse, error)
	GuideFinishSetup(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostGuidePppoe(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	GetGuidePppoe(ctx context.Context) (*models.GuidePppoeStatusResponse, error)
	PostGuideLan(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	GetGuideLan(ctx context.Context) (*models.GuideLanSettingResponse, error)
	PostGuideClientMode(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	GetGuideClientMode(ctx context.Context) (*models.GuideClientModeResponse, error)
	PostGuideGatewayRouter(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostGuideDnsConfig(ctx context.Context, r *http.Request) (*models.GuideDNSConfigResponse, error)
	GetGuideDnsConfig(ctx context.Context) (*models.GuideDNSConfigResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/dns-config/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideDnsConfig(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/dns-config/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideDnsConfig(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/client-mode/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideClientMode(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/client-mode/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideClientMode(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/gateway-router/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideGatewayRouter(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/need/setup/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GuideNeedSetup(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/finish/setup/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GuideFinishSetup(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/pppoe/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuidePppoe(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/pppoe/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuidePppoe(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/lan/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideLan(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/lan/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideLan(ctx)
	})
}

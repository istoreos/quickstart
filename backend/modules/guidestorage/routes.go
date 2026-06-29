package guidestorage

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	PostGuideAria2Init(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostGuideQbittorrentInit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostGuideTransmissionInit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	GetGuideDownloadServiceStatus(ctx context.Context) (*models.GuideDownloadServiceResponse, error)
	GetGuideDownloadPartList(ctx context.Context) (*models.GuideDownloadPartitionListResponse, error)
	GetGuideDockerPartList(ctx context.Context) (*models.GuideDockerPartitionListResponse, error)
	GetGuideDockerStatus(ctx context.Context) (*models.GuideDockerStatusResponse, error)
	PostGuideDockerTransfer(ctx context.Context, r *http.Request) (*models.GuideDockerTransferResponse, error)
	PostGuideDockerSwitch(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	GetGuideSoftSource(ctx context.Context) (*models.GuideSoftSourceResponse, error)
	PostGuideSoftSource(ctx context.Context, r *http.Request) (*models.GuideSoftSourceResponse, error)
	GetGuideSoftSourceList(ctx context.Context) (*models.GuideSoftSourceListResponse, error)
	GetGlobalFolders(ctx context.Context) (*models.GlobalFoldersResponse, error)
	PostGlobalFolders(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/aria2/init/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideAria2Init(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/qbittorrent/init/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideQbittorrentInit(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/transmission/init/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideTransmissionInit(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/download-service/status/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideDownloadServiceStatus(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/download/partition/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideDownloadPartList(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/docker/partition/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideDockerPartList(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/docker/status/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideDockerStatus(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/docker/transfer/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideDockerTransfer(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/docker/switch/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideDockerSwitch(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/soft-source/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideSoftSource(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/soft-source/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideSoftSource(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/soft-source/list/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideSoftSourceList(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/guide/global-folders/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGlobalFolders(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/guide/global-folders/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGlobalFolders(ctx, r)
	})
}

package nas

import (
	"context"
	"net/http"

	"github.com/istoreos/quickstart/backend/internal/httpapi"
	"github.com/istoreos/quickstart/backend/models"
	"github.com/julienschmidt/httprouter"
)

type Backend interface {
	GetNasDiskStatus(ctx context.Context) (*models.NasDiskStatusResponse, error)
	GetNasServiceStatus(ctx context.Context) (*models.NasServiceResponse, error)
	PostNasDiskInit(ctx context.Context, r *http.Request) (*models.NasDiskInitDiskResponse, error)
	PostNasDiskMountPoint(ctx context.Context, r *http.Request) (*models.NasDiskMountPointResponse, error)
	PostNasDiskInitRest(ctx context.Context, r *http.Request) (*models.NasDiskInitDiskResponse, error)
	PostNasDiskPartFormat(ctx context.Context, r *http.Request) (*models.NasDiskPartitionFormatResponse, error)
	PostNasSanboxFormat(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostNasSanboxCommit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostNasSanboxReset(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostNasSanboxExit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	GetNasSanboxDisks(ctx context.Context) (*models.NasSandboxDisksResponse, error)
	GetNasSanboxStatus(ctx context.Context) (*models.NasSandboxStatusResponse, error)
	PostNasDiskPartMount(ctx context.Context, r *http.Request) (*models.NasDiskPartitionMountResponse, error)
	PostNasDiskLinkeaseEnable(ctx context.Context, r *http.Request) (*models.NasLinkeaseEnableResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.GetJSONAliases(router, []string{
		"/cgi-bin/luci/istore/nas/service/status/",
		"/cgi-bin/luci/istore/u/nas/service/status/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNasServiceStatus(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/nas/disk/status/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNasDiskStatus(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/nas/disk/init/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasDiskInit(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/nas/disk/mountpoint/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasDiskMountPoint(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/nas/disk/initrest/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasDiskInitRest(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/nas/disk/partition/format", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasDiskPartFormat(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/nas/disk/partition/mount", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasDiskPartMount(ctx, r)
	})

	httpapi.PostJSONAliases(router, []string{
		"/cgi-bin/luci/istore/nas/linkease/enable",
		"/cgi-bin/luci/istore/u/nas/linkease/enable",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasDiskLinkeaseEnable(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/nas/sandbox/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasSanboxFormat(ctx, r)
	})

	httpapi.PostJSONAliases(router, []string{
		"/cgi-bin/luci/istore/nas/sandbox/commit/",
		"/cgi-bin/luci/istore/u/nas/sandbox/commit/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasSanboxCommit(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/nas/sandbox/exit/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasSanboxExit(ctx, r)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/nas/sandbox/reset/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostNasSanboxReset(ctx, r)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/nas/sandbox/disks/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNasSanboxDisks(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/nas/sandbox/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetNasSanboxStatus(ctx)
	})
}

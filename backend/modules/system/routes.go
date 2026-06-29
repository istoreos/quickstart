package system

import (
	"context"
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	GetSystemVersion(ctx context.Context) (*models.SystemVersionResponse, error)
	GetSystemCheckUpdate(ctx context.Context) (*models.SystemCheckUpdateResponse, error)
	PostSystemAutoCheckUpdate(ctx context.Context, req models.SystemAutoCheckUpdateRequest) (*models.SDKNormalResponse, error)
	PostSystemSetPassword(ctx context.Context, req models.NasSystemSetPasswordRequest) (*models.SDKNormalResponse, error)
	GetSystemGetSession(ctx context.Context) (*models.SystemCsrfTokenResponse, error)
	GetSystemTime(ctx context.Context) (*models.SystemTimeResponse, error)
	GetSystemCpuStatus(ctx context.Context) (*models.SystemCPUStatusResponse, error)
	GetSystemCpuTemperature(ctx context.Context) (*models.SystemCPUTemperatureResponse, error)
	GetSystemMemoryStatus(ctx context.Context) (*models.SystemMemeryStatusResponse, error)
	GetSystemStatus(ctx context.Context) (*models.SystemStatusResponse, error)
	PostSystemReboot(ctx context.Context) (*models.SDKNormalResponse, error)
	PostSystemPowerOff(ctx context.Context) (*models.SDKNormalResponse, error)
	GetSystemModuleSettings(ctx context.Context) (*models.SystemModuleSettingsResponse, error)
	PostSystemModuleSettings(ctx context.Context, req models.SystemModuleSettingsRequest) (*models.SDKNormalResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	httpapi.GetJSONAliases(router, []string{
		"/cgi-bin/luci/istore/system/version/",
		"/cgi-bin/luci/istore/u/system/version/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSystemVersion(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/system/check-update/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSystemCheckUpdate(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/system/auto-check-update/", func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeSystemJSON[models.SystemAutoCheckUpdateRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.PostSystemAutoCheckUpdate(ctx, req)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/system/setPassword/", func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeSystemJSON[models.NasSystemSetPasswordRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.PostSystemSetPassword(ctx, req)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/system/getToken/", func(ctx context.Context, r *http.Request) (any, error) {
		ctx = context.WithValue(ctx, systemSessionIDContextKey, systemSessionIDFromRequest(r))
		return backend.GetSystemGetSession(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/system/time/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSystemTime(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/system/cpu/status/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSystemCpuStatus(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/system/cpu/temperature/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSystemCpuTemperature(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/system/memery/status/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSystemMemoryStatus(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/system/status/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSystemStatus(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/system/reboot/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostSystemReboot(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/system/poweroff/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostSystemPowerOff(ctx)
	})

	httpapi.GetJSON(router, "/cgi-bin/luci/istore/system/module-settings/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetSystemModuleSettings(ctx)
	})

	httpapi.PostJSON(router, "/cgi-bin/luci/istore/system/module-settings/", func(ctx context.Context, r *http.Request) (any, error) {
		req, err := decodeSystemJSON[models.SystemModuleSettingsRequest](r)
		if err != nil {
			return nil, err
		}
		return backend.PostSystemModuleSettings(ctx, req)
	})
}

func decodeSystemJSON[T any](r *http.Request) (T, error) {
	req, err := httpapi.DecodeJSON[T](r)
	if err != nil {
		return req, errors.New("请求解析失败")
	}
	return req, nil
}

const systemSessionIDContextKey = "github.com/linkease/quick-start/istore-backend/system/session-id"

func systemSessionIDFromRequest(r *http.Request) string {
	for _, name := range []string{"sysauth", "sysauth_http", "sysauth_https"} {
		if c, err := r.Cookie(name); err == nil {
			return c.Value
		}
	}
	return ""
}

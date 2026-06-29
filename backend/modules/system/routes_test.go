package system

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSystemBackend struct {
	err error

	calls              []string
	autoCheckUpdateReq *models.SystemAutoCheckUpdateRequest
	setPasswordReq     *models.NasSystemSetPasswordRequest
	moduleSettingsReq  *models.SystemModuleSettingsRequest
	sessionID          string
}

func (backend *fakeSystemBackend) record(call string) error {
	backend.calls = append(backend.calls, call)
	return backend.err
}

func (backend *fakeSystemBackend) GetSystemVersion(ctx context.Context) (*models.SystemVersionResponse, error) {
	if err := backend.record("version"); err != nil {
		return nil, err
	}
	return &models.SystemVersionResponse{
		Result: &models.SystemVersionResponseResult{Quickstart: "1.0.0"},
	}, nil
}

func (backend *fakeSystemBackend) GetSystemCheckUpdate(ctx context.Context) (*models.SystemCheckUpdateResponse, error) {
	if err := backend.record("checkUpdate"); err != nil {
		return nil, err
	}
	return &models.SystemCheckUpdateResponse{
		Result: &models.SystemCheckUpdateResponseResult{Msg: "current"},
	}, nil
}

func (backend *fakeSystemBackend) PostSystemAutoCheckUpdate(ctx context.Context, req models.SystemAutoCheckUpdateRequest) (*models.SDKNormalResponse, error) {
	backend.autoCheckUpdateReq = &req
	if err := backend.record("autoCheckUpdate"); err != nil {
		return nil, err
	}
	return normalSystemResponse(), nil
}

func (backend *fakeSystemBackend) PostSystemSetPassword(ctx context.Context, req models.NasSystemSetPasswordRequest) (*models.SDKNormalResponse, error) {
	backend.setPasswordReq = &req
	if err := backend.record("setPassword"); err != nil {
		return nil, err
	}
	return normalSystemResponse(), nil
}

func (backend *fakeSystemBackend) GetSystemGetSession(ctx context.Context) (*models.SystemCsrfTokenResponse, error) {
	backend.sessionID, _ = ctx.Value(systemSessionIDContextKey).(string)
	if err := backend.record("getToken"); err != nil {
		return nil, err
	}
	return &models.SystemCsrfTokenResponse{
		Result: &models.SystemCsrfTokenResponseResult{Token: "token-1"},
	}, nil
}

func (backend *fakeSystemBackend) GetSystemTime(ctx context.Context) (*models.SystemTimeResponse, error) {
	if err := backend.record("time"); err != nil {
		return nil, err
	}
	return &models.SystemTimeResponse{
		Result: &models.SystemTimeResponseResult{Localtime: "2026-06-25 00:00:00"},
	}, nil
}

func (backend *fakeSystemBackend) GetSystemCpuStatus(ctx context.Context) (*models.SystemCPUStatusResponse, error) {
	if err := backend.record("cpuStatus"); err != nil {
		return nil, err
	}
	return &models.SystemCPUStatusResponse{
		Result: &models.SystemCPUStatusResponseResult{Usage: 12},
	}, nil
}

func (backend *fakeSystemBackend) GetSystemCpuTemperature(ctx context.Context) (*models.SystemCPUTemperatureResponse, error) {
	if err := backend.record("cpuTemperature"); err != nil {
		return nil, err
	}
	return &models.SystemCPUTemperatureResponse{
		Result: &models.SystemCPUTemperatureResponseResult{Temperature: 47},
	}, nil
}

func (backend *fakeSystemBackend) GetSystemMemoryStatus(ctx context.Context) (*models.SystemMemeryStatusResponse, error) {
	if err := backend.record("memoryStatus"); err != nil {
		return nil, err
	}
	return &models.SystemMemeryStatusResponse{
		Result: &models.SystemMemeryStatusResponseResult{Available: "128 MB"},
	}, nil
}

func (backend *fakeSystemBackend) GetSystemStatus(ctx context.Context) (*models.SystemStatusResponse, error) {
	if err := backend.record("status"); err != nil {
		return nil, err
	}
	return &models.SystemStatusResponse{
		Result: &models.SystemStatusResponseResult{CPUUsage: 33},
	}, nil
}

func (backend *fakeSystemBackend) PostSystemReboot(ctx context.Context) (*models.SDKNormalResponse, error) {
	if err := backend.record("reboot"); err != nil {
		return nil, err
	}
	return normalSystemResponse(), nil
}

func (backend *fakeSystemBackend) PostSystemPowerOff(ctx context.Context) (*models.SDKNormalResponse, error) {
	if err := backend.record("poweroff"); err != nil {
		return nil, err
	}
	return normalSystemResponse(), nil
}

func (backend *fakeSystemBackend) GetSystemModuleSettings(ctx context.Context) (*models.SystemModuleSettingsResponse, error) {
	if err := backend.record("moduleSettingsGet"); err != nil {
		return nil, err
	}
	return &models.SystemModuleSettingsResponse{
		Result: &models.SystemModuleSettingsResponseResult{DiableDisplay: []string{"lcd"}},
	}, nil
}

func (backend *fakeSystemBackend) PostSystemModuleSettings(ctx context.Context, req models.SystemModuleSettingsRequest) (*models.SDKNormalResponse, error) {
	backend.moduleSettingsReq = &req
	if err := backend.record("moduleSettingsPost"); err != nil {
		return nil, err
	}
	return normalSystemResponse(), nil
}

func TestRegisterSystemRoutesVersionAliasUsesSameBackendMethod(t *testing.T) {
	backend := &fakeSystemBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	requestSystemRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/system/version/", "", true)
	requestSystemRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/u/system/version/", "", true)

	if len(backend.calls) != 2 || backend.calls[0] != "version" || backend.calls[1] != "version" {
		t.Fatalf("expected both version paths to call version backend method, got %#v", backend.calls)
	}
}

func TestRegisterSystemRoutesPostDecodesRequestBody(t *testing.T) {
	backend := &fakeSystemBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	const path = "/cgi-bin/luci/istore/system/setPassword/"
	requestSystemRoute(t, router, http.MethodPost, path, `{"password":"secret"}`, true)

	if len(backend.calls) != 1 || backend.calls[0] != "setPassword" {
		t.Fatalf("expected setPassword backend call, got %#v", backend.calls)
	}
	if backend.setPasswordReq == nil || backend.setPasswordReq.Password != "secret" {
		t.Fatalf("expected decoded password request, got %#v", backend.setPasswordReq)
	}
}

func TestRegisterSystemRoutesGetTokenPassesSessionIDInContext(t *testing.T) {
	backend := &fakeSystemBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	req := httptest.NewRequest(http.MethodGet, "/cgi-bin/luci/istore/system/getToken/", nil)
	req.Header.Set("X-Forwarded-Sid", "sid-1")
	req.AddCookie(&http.Cookie{Name: "sysauth_https", Value: "https-session"})
	req.AddCookie(&http.Cookie{Name: "sysauth_http", Value: "http-session"})
	req.AddCookie(&http.Cookie{Name: "sysauth", Value: "main-session"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if backend.sessionID != "main-session" {
		t.Fatalf("sessionID = %q, want main-session", backend.sessionID)
	}
}

func TestRegisterSystemRoutesRejectsInvalidPostJSONBeforeBackend(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "auto check update malformed",
			path: "/cgi-bin/luci/istore/system/auto-check-update/",
			body: `{"enable":`,
		},
		{
			name: "auto check update trailing",
			path: "/cgi-bin/luci/istore/system/auto-check-update/",
			body: `{"enable":true} trailing`,
		},
		{
			name: "set password malformed",
			path: "/cgi-bin/luci/istore/system/setPassword/",
			body: `{"password":`,
		},
		{
			name: "set password trailing",
			path: "/cgi-bin/luci/istore/system/setPassword/",
			body: `{"password":"secret"} trailing`,
		},
		{
			name: "module settings malformed",
			path: "/cgi-bin/luci/istore/system/module-settings/",
			body: `{"diableDisplay":`,
		},
		{
			name: "module settings trailing",
			path: "/cgi-bin/luci/istore/system/module-settings/",
			body: `{"diableDisplay":["lcd"]} trailing`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeSystemBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestSystemRoute(t, router, http.MethodPost, tt.path, tt.body, true)

			if len(backend.calls) != 0 {
				t.Fatalf("expected backend not to be called, got %#v", backend.calls)
			}
			requireEnvelopeCode(t, resp, httpapi.GeneralError)
		})
	}
}

func TestRegisterSystemRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeSystemBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestSystemRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/system/version/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterSystemRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeSystemBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestSystemRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/system/version/", "", true)

	if len(backend.calls) != 1 || backend.calls[0] != "version" {
		t.Fatalf("expected version backend call, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.GeneralError)
}

func TestRegisterSystemRoutesRouteToMethodMapping(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCall string
	}{
		{
			name:     "version",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/version/",
			wantCall: "version",
		},
		{
			name:     "version user alias",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/u/system/version/",
			wantCall: "version",
		},
		{
			name:     "check update",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/check-update/",
			wantCall: "checkUpdate",
		},
		{
			name:     "auto check update",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/system/auto-check-update/",
			wantCall: "autoCheckUpdate",
		},
		{
			name:     "set password",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/system/setPassword/",
			wantCall: "setPassword",
		},
		{
			name:     "get token",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/getToken/",
			wantCall: "getToken",
		},
		{
			name:     "time",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/time/",
			wantCall: "time",
		},
		{
			name:     "cpu status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/cpu/status/",
			wantCall: "cpuStatus",
		},
		{
			name:     "cpu temperature",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/cpu/temperature/",
			wantCall: "cpuTemperature",
		},
		{
			name:     "memory status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/memery/status/",
			wantCall: "memoryStatus",
		},
		{
			name:     "status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/status/",
			wantCall: "status",
		},
		{
			name:     "reboot",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/system/reboot/",
			wantCall: "reboot",
		},
		{
			name:     "poweroff",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/system/poweroff/",
			wantCall: "poweroff",
		},
		{
			name:     "module settings get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/system/module-settings/",
			wantCall: "moduleSettingsGet",
		},
		{
			name:     "module settings post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/system/module-settings/",
			wantCall: "moduleSettingsPost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeSystemBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestSystemRoute(t, router, tt.method, tt.path, validSystemRouteBody(tt.method, tt.path), true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
		})
	}
}

func validSystemRouteBody(method, path string) string {
	if method != http.MethodPost {
		return ""
	}
	switch path {
	case "/cgi-bin/luci/istore/system/auto-check-update/":
		return `{"enable":true}`
	case "/cgi-bin/luci/istore/system/setPassword/":
		return `{"password":"secret"}`
	case "/cgi-bin/luci/istore/system/module-settings/":
		return `{"diableDisplay":["lcd"]}`
	default:
		return ""
	}
}

func normalSystemResponse() *models.SDKNormalResponse {
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}
}

func requestSystemRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if withSID {
		req.Header.Set("X-Forwarded-Sid", "sid-1")
	}
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s expected status 200, got %d", method, path, rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func requireEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

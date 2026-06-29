package guidecore

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

type fakeGuideCoreBackend struct {
	err error

	calls        []string
	requestPaths []string
}

func (backend *fakeGuideCoreBackend) record(call string, r *http.Request) {
	backend.calls = append(backend.calls, call)
	if r != nil {
		backend.requestPaths = append(backend.requestPaths, r.URL.Path)
	}
}

func (backend *fakeGuideCoreBackend) GuideNeedSetup(ctx context.Context, r *http.Request) (*models.GuideNeedSetupResponse, error) {
	backend.record("needSetup", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideNeedSetupResponse{Result: &models.GuideNeedSetupInfo{Need: true}}, nil
}

func (backend *fakeGuideCoreBackend) GuideFinishSetup(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("finishSetup", r)
	return backend.normalResponse()
}

func (backend *fakeGuideCoreBackend) PostGuidePppoe(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("postPppoe", r)
	return backend.normalResponse()
}

func (backend *fakeGuideCoreBackend) GetGuidePppoe(ctx context.Context) (*models.GuidePppoeStatusResponse, error) {
	backend.record("getPppoe", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuidePppoeStatusResponse{Result: &models.GuidePppoeStatusResponseResult{Account: "user"}}, nil
}

func (backend *fakeGuideCoreBackend) PostGuideLan(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("postLan", r)
	return backend.normalResponse()
}

func (backend *fakeGuideCoreBackend) GetGuideLan(ctx context.Context) (*models.GuideLanSettingResponse, error) {
	backend.record("getLan", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideLanSettingResponse{Result: &models.GuideLanSettingResponseResult{LanIP: "192.168.1.1"}}, nil
}

func (backend *fakeGuideCoreBackend) PostGuideClientMode(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("postClientMode", r)
	return backend.normalResponse()
}

func (backend *fakeGuideCoreBackend) GetGuideClientMode(ctx context.Context) (*models.GuideClientModeResponse, error) {
	backend.record("getClientMode", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideClientModeResponse{Result: &models.GuideClientModeResponseResult{WanProto: "dhcp"}}, nil
}

func (backend *fakeGuideCoreBackend) PostGuideGatewayRouter(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("postGatewayRouter", r)
	return backend.normalResponse()
}

func (backend *fakeGuideCoreBackend) PostGuideDnsConfig(ctx context.Context, r *http.Request) (*models.GuideDNSConfigResponse, error) {
	backend.record("postDnsConfig", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDNSConfigResponse{Result: &models.GuideDNSConfigResponseResult{DNSProto: "manual"}}, nil
}

func (backend *fakeGuideCoreBackend) GetGuideDnsConfig(ctx context.Context) (*models.GuideDNSConfigResponse, error) {
	backend.record("getDnsConfig", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDNSConfigResponse{Result: &models.GuideDNSConfigResponseResult{DNSProto: "auto"}}, nil
}

func (backend *fakeGuideCoreBackend) normalResponse() (*models.SDKNormalResponse, error) {
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func TestRegisterGuideCoreRoutesMapsRoutesToBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCall string
	}{
		{
			name:     "get dns config",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/dns-config/",
			wantCall: "getDnsConfig",
		},
		{
			name:     "post dns config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/dns-config/",
			wantCall: "postDnsConfig",
		},
		{
			name:     "get client mode",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/client-mode/",
			wantCall: "getClientMode",
		},
		{
			name:     "post client mode",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/client-mode/",
			wantCall: "postClientMode",
		},
		{
			name:     "post gateway router",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/gateway-router/",
			wantCall: "postGatewayRouter",
		},
		{
			name:     "get need setup",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/need/setup/",
			wantCall: "needSetup",
		},
		{
			name:     "post finish setup",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/finish/setup/",
			wantCall: "finishSetup",
		},
		{
			name:     "get pppoe",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/pppoe/",
			wantCall: "getPppoe",
		},
		{
			name:     "post pppoe",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/pppoe/",
			wantCall: "postPppoe",
		},
		{
			name:     "post lan",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/lan/",
			wantCall: "postLan",
		},
		{
			name:     "get lan",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/lan/",
			wantCall: "getLan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeGuideCoreBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestGuideCoreRoute(t, router, tt.method, tt.path, `{}`, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
			requireGuideCoreSuccessResponse(t, resp)
		})
	}
}

func TestRegisterGuideCoreRoutesPostPassesOriginalRequestPath(t *testing.T) {
	backend := &fakeGuideCoreBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	path := "/cgi-bin/luci/istore/guide/gateway-router/"
	requestGuideCoreRoute(t, router, http.MethodPost, path, `{}`, true)

	if len(backend.calls) != 1 || backend.calls[0] != "postGatewayRouter" {
		t.Fatalf("expected postGatewayRouter backend call, got %#v", backend.calls)
	}
	if len(backend.requestPaths) != 1 || backend.requestPaths[0] != path {
		t.Fatalf("expected original request path %q, got %#v", path, backend.requestPaths)
	}
}

func TestRegisterGuideCoreRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeGuideCoreBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestGuideCoreRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/guide/dns-config/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireGuideCoreEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterGuideCoreRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeGuideCoreBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestGuideCoreRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/guide/client-mode/", `{}`, true)

	if len(backend.calls) != 1 || backend.calls[0] != "postClientMode" {
		t.Fatalf("expected postClientMode backend call, got %#v", backend.calls)
	}
	requireGuideCoreEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestGuideCoreRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func requireGuideCoreSuccessResponse(t *testing.T, resp map[string]any) {
	t.Helper()

	if _, ok := resp["result"]; ok {
		return
	}
	requireGuideCoreEnvelopeCode(t, resp, 0)
}

func requireGuideCoreEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

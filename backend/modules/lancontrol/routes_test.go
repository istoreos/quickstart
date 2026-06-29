package lancontrol

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

type fakeLanControlBackend struct {
	err error

	calls        []string
	requestPaths []string
}

func (backend *fakeLanControlBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeLanControlBackend) recordRequest(call string, r *http.Request) {
	backend.record(call)
	backend.requestPaths = append(backend.requestPaths, r.URL.Path)
}

func (backend *fakeLanControlBackend) GetSpeedsForAllDevice(ctx context.Context, r *http.Request) (*models.DeviceSpeedStatsResponse, error) {
	backend.recordRequest("speedsForDevices", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.DeviceSpeedStatsResponse{
		Result: []*models.DeviceSpeedStat{{IP: "192.168.1.2", DownloadSpeed: 1024}},
	}, nil
}

func (backend *fakeLanControlBackend) GetSpeedsForOneDevice(ctx context.Context, r *http.Request) (*models.NetworkStatisticsResponse, error) {
	backend.recordRequest("speedsForOneDevice", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkStatisticsResponse{
		Result: &models.NetworkStatisticsResponseResult{
			Items: []*models.NetworkStatisticsItem{{DownloadSpeed: 2048}},
		},
	}, nil
}

func (backend *fakeLanControlBackend) PostLanDhcpTagsConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("dhcpTagsConfig", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanDhcpGatewayConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("dhcpGatewayConfig", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanSpeedLimitConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("speedLimitConfig", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanEnableSpeedLimit(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("enableSpeedLimit", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanEnableFloatGateway(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("enableFloatGateway", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) PostLanStaticDeviceConfig(ctx context.Context, r *http.Request) (*models.JSONResponse, error) {
	backend.recordRequest("staticDeviceConfig", r)
	return backend.normalResponse()
}

func (backend *fakeLanControlBackend) GetLanGlobalConfigs(ctx context.Context) (*models.LANCtrlGlobalConfigResponse, error) {
	backend.record("globalConfigs")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.LANCtrlGlobalConfigResponse{Result: &models.LANCtrlGlobalConfig{}}, nil
}

func (backend *fakeLanControlBackend) GetLanListDevices(ctx context.Context) (*models.LANDeviceResponse, error) {
	backend.record("listDevices")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.LANDeviceResponse{
		Result: &models.LANDeviceResponseResult{Devices: models.LANDevices{{IP: "192.168.1.2"}}},
	}, nil
}

func (backend *fakeLanControlBackend) GetLanListStaticDevices(ctx context.Context) (*models.LANCtrlStaticAssignedResponse, error) {
	backend.record("listStaticDevices")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.LANCtrlStaticAssignedResponse{
		Result: []*models.LANStaticAssigned{{AssignedIP: "192.168.1.10"}},
	}, nil
}

func (backend *fakeLanControlBackend) GetLanListSpeedLimitedDevices(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error) {
	backend.record("listSpeedLimitedDevices")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.LANCtrlSpeedLimitResponse{
		Result: []*models.LANCtrlSpeedLimitItem{{IP: "192.168.1.20"}},
	}, nil
}

func (backend *fakeLanControlBackend) normalResponse() (*models.JSONResponse, error) {
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.JSONResponse{}, nil
}

func TestRegisterLanControlRoutesMapsRoutesToBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		wantCall string
	}{
		{
			name:     "speeds for devices",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/speedsForDevices/",
			wantCall: "speedsForDevices",
		},
		{
			name:     "speeds for one device",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/speedsForOneDevice/",
			body:     `{"ip":"192.168.1.2"}`,
			wantCall: "speedsForOneDevice",
		},
		{
			name:     "dhcp tags config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/dhcpTagsConfig/",
			body:     `{"action":"add","tagName":"guest"}`,
			wantCall: "dhcpTagsConfig",
		},
		{
			name:     "dhcp gateway config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/dhcpGatewayConfig/",
			body:     `{"dhcpEnabled":true,"dhcpGateway":"192.168.1.1"}`,
			wantCall: "dhcpGatewayConfig",
		},
		{
			name:     "speed limit config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/speedLimitConfig/",
			body:     `{"action":"add","mac":"aa:bb:cc:dd:ee:ff"}`,
			wantCall: "speedLimitConfig",
		},
		{
			name:     "enable speed limit",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/enableSpeedLimit/",
			body:     `{"enabled":true}`,
			wantCall: "enableSpeedLimit",
		},
		{
			name:     "enable float gateway",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/enableFloatGateway/",
			body:     `{"enabled":true}`,
			wantCall: "enableFloatGateway",
		},
		{
			name:     "static device config",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/lanctrl/staticDeviceConfig/",
			body:     `{"action":"add","assignedMac":"AA:BB:CC:DD:EE:FF"}`,
			wantCall: "staticDeviceConfig",
		},
		{
			name:     "global configs",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/globalConfigs/",
			wantCall: "globalConfigs",
		},
		{
			name:     "list devices",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/listDevices/",
			wantCall: "listDevices",
		},
		{
			name:     "list static devices",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/listStaticDevices/",
			wantCall: "listStaticDevices",
		},
		{
			name:     "list speed limited devices",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/lanctrl/listSpeedLimitedDevices/",
			wantCall: "listSpeedLimitedDevices",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeLanControlBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestLanControlRoute(t, router, tt.method, tt.path, tt.body, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
		})
	}
}

func TestRegisterLanControlRoutesPostPassesOriginalRequestPath(t *testing.T) {
	backend := &fakeLanControlBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	const path = "/cgi-bin/luci/istore/lanctrl/staticDeviceConfig/"
	requestLanControlRoute(t, router, http.MethodPost, path, `{"action":"add"}`, true)

	if len(backend.requestPaths) != 1 || backend.requestPaths[0] != path {
		t.Fatalf("expected request path %q, got %#v", path, backend.requestPaths)
	}
}

func TestRegisterLanControlRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeLanControlBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestLanControlRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/lanctrl/listDevices/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireLanControlEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterLanControlRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeLanControlBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestLanControlRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/lanctrl/listDevices/", "", true)

	if len(backend.calls) != 1 || backend.calls[0] != "listDevices" {
		t.Fatalf("expected listDevices backend call, got %#v", backend.calls)
	}
	requireLanControlEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestLanControlRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func requireLanControlEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

package network

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
	"github.com/linkease/quick-start/istore-backend/modules/network/interfacewrite"
)

type fakeNetworkBackend struct {
	err error

	calls           []string
	setupFinish     []bool
	ipVersions      []string
	interfaceInputs []interfacewrite.Input
}

func (backend *fakeNetworkBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeNetworkBackend) GetNetworkStatistic(ctx context.Context) (*models.NetworkStatisticsResponse, error) {
	backend.record("statistics")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkStatisticsResponse{
		Result: &models.NetworkStatisticsResponseResult{Slots: 3},
	}, nil
}

func (backend *fakeNetworkBackend) GetNetworkStatus(ctx context.Context, setupFinish bool) (*models.NetworkStatusResponse, error) {
	backend.record("status")
	backend.setupFinish = append(backend.setupFinish, setupFinish)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkStatusResponse{
		Result: &models.NetworkStatusResponseResult{NetworkInfo: "netSuccess"},
	}, nil
}

func (backend *fakeNetworkBackend) GetNetworkDeviceList(ctx context.Context) (*models.DeviceListResponse, error) {
	backend.record("deviceList")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.DeviceListResponse{
		Result: &models.DeviceListResponseResult{
			Devices: []*models.DeviceInfo{{Name: "phone"}},
		},
	}, nil
}

func (backend *fakeNetworkBackend) EnableNetworkHomebox(ctx context.Context) (*models.NetworkHomeBoxEnableResponse, error) {
	backend.record("homeboxEnable")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkHomeBoxEnableResponse{
		Result: &models.NetworkHomeBoxEnableResponseResult{Port: "8897"},
	}, nil
}

func (backend *fakeNetworkBackend) GetNetworkInterfaceStatus(ctx context.Context) (*models.NetworkInterfaceStatusResponse, error) {
	backend.record("interfaceStatus")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkInterfaceStatusResponse{
		Result: &models.NetworkInterfaceStatusResponseResult{
			Interfaces: []*models.NetworkInterfaceInfo{{Name: "lan"}},
		},
	}, nil
}

func (backend *fakeNetworkBackend) CheckNetworkPublicAddress(ctx context.Context, ipVersion string) (*models.NetworkCheckPublicNetResponse, error) {
	backend.record("checkPublicNet")
	backend.ipVersions = append(backend.ipVersions, ipVersion)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkCheckPublicNetResponse{
		Result: &models.NetworkCheckPublicNetResponseResult{Address: "203.0.113.10"},
	}, nil
}

func (backend *fakeNetworkBackend) GetNetworkPortList(ctx context.Context) (*models.NetworkPortListResponse, error) {
	backend.record("portList")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkPortListResponse{
		Result: &models.NetworkPortListResponseResult{
			Ports: []*models.NetworkPortInfo{{Name: "eth0"}},
		},
	}, nil
}

func (backend *fakeNetworkBackend) GetNetworkInterfaceConfig(ctx context.Context) (*models.NetworkInterfaceGetConfigResponse, error) {
	backend.record("interfaceConfigGet")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NetworkInterfaceGetConfigResponse{
		Result: &models.NetworkInterfaceGetConfigResponseResult{
			Interfaces: []*models.NetworkInterfaceInfo{{Name: "wan"}},
			Devices:    []*models.NetworkPortInfo{{Name: "eth1"}},
		},
	}, nil
}

func (backend *fakeNetworkBackend) SetNetworkInterfaceConfig(ctx context.Context, input interfacewrite.Input) (*models.SDKNormalResponse, error) {
	backend.record("interfaceConfigPost")
	backend.interfaceInputs = append(backend.interfaceInputs, input)
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func TestRegisterNetworkRoutesSuccessRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantCall   string
		assertions func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any)
	}{
		{
			name:     "statistics",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/network/statistics/",
			wantCall: "statistics",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireNestedNumber(t, resp, []string{"result", "slots"}, 3)
			},
		},
		{
			name:     "statistics user alias",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/u/network/statistics/",
			wantCall: "statistics",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireNestedNumber(t, resp, []string{"result", "slots"}, 3)
			},
		},
		{
			name:     "status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/network/status/",
			wantCall: "status",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireSetupFinish(t, backend, false)
				requireNestedString(t, resp, []string{"result", "networkInfo"}, "netSuccess")
			},
		},
		{
			name:     "status user alias",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/u/network/status/",
			wantCall: "status",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireSetupFinish(t, backend, false)
				requireNestedString(t, resp, []string{"result", "networkInfo"}, "netSuccess")
			},
		},
		{
			name:     "setup finish status",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/network/setup/finish/",
			wantCall: "status",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireSetupFinish(t, backend, true)
			},
		},
		{
			name:     "device list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/network/device/list/",
			wantCall: "deviceList",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireFirstNestedString(t, resp, []string{"result", "devices"}, "name", "phone")
			},
		},
		{
			name:     "homebox enable",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/network/homebox/enable",
			wantCall: "homeboxEnable",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireNestedString(t, resp, []string{"result", "port"}, "8897")
			},
		},
		{
			name:     "interface status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/network/interface/status/",
			wantCall: "interfaceStatus",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireFirstNestedString(t, resp, []string{"result", "interfaces"}, "name", "lan")
			},
		},
		{
			name:     "check public net",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/network/checkPublicNet/",
			body:     `{"ipVersion":"ipv4"}`,
			wantCall: "checkPublicNet",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				if len(backend.ipVersions) != 1 || backend.ipVersions[0] != "ipv4" {
					t.Fatalf("expected ipv4 check, got %#v", backend.ipVersions)
				}
				requireNestedString(t, resp, []string{"result", "address"}, "203.0.113.10")
			},
		},
		{
			name:     "port list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/network/port/list/",
			wantCall: "portList",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireFirstNestedString(t, resp, []string{"result", "ports"}, "name", "eth0")
			},
		},
		{
			name:     "interface config get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/network/interface/config/",
			wantCall: "interfaceConfigGet",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				requireFirstNestedString(t, resp, []string{"result", "interfaces"}, "name", "wan")
				requireFirstNestedString(t, resp, []string{"result", "devices"}, "name", "eth1")
			},
		},
		{
			name:     "interface config post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/network/interface/config/",
			body:     `{"configs":[{"name":"lan","proto":"dhcp","devices":["eth0"],"firewallType":"lan"}]}`,
			wantCall: "interfaceConfigPost",
			assertions: func(t *testing.T, backend *fakeNetworkBackend, resp map[string]any) {
				if len(backend.interfaceInputs) != 1 {
					t.Fatalf("expected one typed interface config input, got %#v", backend.interfaceInputs)
				}
				configs := backend.interfaceInputs[0].Configs
				if len(configs) != 1 {
					t.Fatalf("expected one interface config, got %#v", configs)
				}
				if configs[0].Name != "lan" || configs[0].Proto != "dhcp" || len(configs[0].Devices) != 1 || configs[0].Devices[0] != "eth0" || configs[0].FirewallType != "lan" {
					t.Fatalf("unexpected decoded config: %#v", configs[0])
				}
				requireEnvelopeCode(t, resp, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeNetworkBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestNetworkRoute(t, router, tt.method, tt.path, tt.body, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
			tt.assertions(t, backend, resp)
		})
	}
}

func TestRegisterNetworkRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeNetworkBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestNetworkRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/network/statistics/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterNetworkRoutesMalformedCheckPublicNetJSONReturnsError(t *testing.T) {
	backend := &fakeNetworkBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestNetworkRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/network/checkPublicNet/", "{", true)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.GeneralError)
}

func TestRegisterNetworkRoutesMalformedInterfaceConfigJSONReturnsError(t *testing.T) {
	backend := &fakeNetworkBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestNetworkRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/network/interface/config/", "{", true)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.GeneralError)
}

func TestRegisterNetworkRoutesCheckPublicNetRejectsTrailingGarbage(t *testing.T) {
	backend := &fakeNetworkBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestNetworkRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/network/checkPublicNet/", `{"ipVersion":"ipv4"} trailing`, true)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.GeneralError)
}

func TestRegisterNetworkRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeNetworkBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestNetworkRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/network/statistics/", "", true)

	if len(backend.calls) != 1 || backend.calls[0] != "statistics" {
		t.Fatalf("expected statistics backend call, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestNetworkRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func requireSetupFinish(t *testing.T, backend *fakeNetworkBackend, want bool) {
	t.Helper()

	if len(backend.setupFinish) != 1 || backend.setupFinish[0] != want {
		t.Fatalf("expected setupFinish %v, got %#v", want, backend.setupFinish)
	}
}

func requireNestedString(t *testing.T, resp map[string]any, path []string, want string) {
	t.Helper()

	value := nestedValue(t, resp, path)
	got, ok := value.(string)
	if !ok || got != want {
		t.Fatalf("expected %s to be %q, got %#v", strings.Join(path, "."), want, value)
	}
}

func requireNestedNumber(t *testing.T, resp map[string]any, path []string, want float64) {
	t.Helper()

	value := nestedValue(t, resp, path)
	got, ok := value.(float64)
	if !ok || got != want {
		t.Fatalf("expected %s to be %v, got %#v", strings.Join(path, "."), want, value)
	}
}

func requireFirstNestedString(t *testing.T, resp map[string]any, path []string, key, want string) {
	t.Helper()

	value := nestedValue(t, resp, path)
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected non-empty array at %s, got %#v", strings.Join(path, "."), value)
	}
	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected object at %s[0], got %#v", strings.Join(path, "."), items[0])
	}
	got, ok := first[key].(string)
	if !ok || got != want {
		t.Fatalf("expected %s[0].%s to be %q, got %#v", strings.Join(path, "."), key, want, first[key])
	}
}

func nestedValue(t *testing.T, resp map[string]any, path []string) any {
	t.Helper()

	var current any = resp
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected object before %q in %s, got %#v", key, strings.Join(path, "."), current)
		}
		current, ok = obj[key]
		if !ok {
			t.Fatalf("missing %q in %#v", key, obj)
		}
	}
	return current
}

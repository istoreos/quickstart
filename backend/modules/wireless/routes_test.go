package wireless

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

type fakeWirelessBackend struct {
	err error

	calls          []string
	enableIfaceReq *models.WirelessEnableIfaceRequest
}

func (backend *fakeWirelessBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeWirelessBackend) recordAction(call string) error {
	backend.record(call)
	return backend.err
}

func (backend *fakeWirelessBackend) WirelessListIfaces(ctx context.Context) (*models.WirelessListIfaceResponse, error) {
	backend.record("listIface")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.WirelessListIfaceResponse{
		Result: &models.WirelessListIfaceResponseResult{
			Ifaces: []*models.WirelessIfaceInfo{{IfaceName: "wifi2g", Ssid: "test-wifi"}},
		},
	}, nil
}

func (backend *fakeWirelessBackend) WirelessEnableIface(ctx context.Context, req models.WirelessEnableIfaceRequest) error {
	backend.enableIfaceReq = &req
	return backend.recordAction("enableIface")
}

func (backend *fakeWirelessBackend) WirelessSetDevicePower(ctx context.Context, req models.WirelessSetDevicePowerRequest) error {
	return backend.recordAction("setDevicePower")
}

func (backend *fakeWirelessBackend) WirelessEditIface(ctx context.Context, req models.WirelessIfaceInfo) error {
	return backend.recordAction("editIface")
}

func (backend *fakeWirelessBackend) WirelessQuickSetupIface(ctx context.Context, req models.WirelessQuickSetupRequest) error {
	return backend.recordAction("setup")
}

func TestRegisterWirelessRoutesRouteToMethodMapping(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		wantCall string
	}{
		{
			name:     "list iface",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/wireless/list-iface/",
			wantCall: "listIface",
		},
		{
			name:     "enable iface",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/wireless/enable-iface/",
			body:     `{"ifaceName":"wifi2g","enable":true}`,
			wantCall: "enableIface",
		},
		{
			name:     "set device power",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/wireless/set-device-power/",
			body:     `{"device":"radio0","txpower":20}`,
			wantCall: "setDevicePower",
		},
		{
			name:     "edit iface",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/wireless/edit-iface/",
			body:     `{"ifaceName":"wifi2g","ssid":"test-wifi"}`,
			wantCall: "editIface",
		},
		{
			name:     "setup",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/wireless/setup/",
			body:     `{"wifi2g":{"ssid":"test-2g"},"wifi5g":{"ssid":"test-5g"}}`,
			wantCall: "setup",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeWirelessBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestWirelessRoute(t, router, tt.method, tt.path, tt.body, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
		})
	}
}

func TestRegisterWirelessRoutesPassesDecodedRequestToBackend(t *testing.T) {
	backend := &fakeWirelessBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	requestWirelessRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/wireless/enable-iface/", `{"ifaceName":"wifi2g","enable":true}`, true)

	if backend.enableIfaceReq == nil {
		t.Fatal("expected enable iface request to be recorded")
	}
	if backend.enableIfaceReq.IfaceName != "wifi2g" || !backend.enableIfaceReq.Enable {
		t.Fatalf("unexpected enable iface request: %#v", backend.enableIfaceReq)
	}
}

func TestRegisterWirelessRoutesListIfaceReturnsBackendResponse(t *testing.T) {
	backend := &fakeWirelessBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestWirelessRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/wireless/list-iface/", "", true)

	requireWirelessFirstNestedString(t, resp, []string{"result", "ifaces"}, "ifaceName", "wifi2g")
	requireWirelessFirstNestedString(t, resp, []string{"result", "ifaces"}, "ssid", "test-wifi")
}

func TestRegisterWirelessRoutesActionSuccessResponses(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "enable iface", path: "/cgi-bin/luci/istore/wireless/enable-iface/", body: `{"ifaceName":"wifi2g","enable":true}`},
		{name: "set device power", path: "/cgi-bin/luci/istore/wireless/set-device-power/", body: `{"device":"radio0","txpower":20}`},
		{name: "edit iface", path: "/cgi-bin/luci/istore/wireless/edit-iface/", body: `{"ifaceName":"wifi2g","ssid":"test-wifi"}`},
		{name: "setup", path: "/cgi-bin/luci/istore/wireless/setup/", body: `{"wifi2g":{"ssid":"test-2g"},"wifi5g":{"ssid":"test-5g"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeWirelessBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestWirelessRoute(t, router, http.MethodPost, tt.path, tt.body, true)

			requireWirelessEnvelopeCode(t, resp, 0)
		})
	}
}

func TestRegisterWirelessRoutesActionBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeWirelessBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestWirelessRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/wireless/enable-iface/", `{"ifaceName":"wifi2g","enable":true}`, true)

	if len(backend.calls) != 1 || backend.calls[0] != "enableIface" {
		t.Fatalf("expected enableIface backend call, got %#v", backend.calls)
	}
	requireWirelessEnvelopeCode(t, resp, httpapi.GeneralError)
}

func TestRegisterWirelessRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeWirelessBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestWirelessRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/wireless/list-iface/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireWirelessEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func requestWirelessRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func TestRegisterWirelessRoutesRejectMalformedPostJSONWithoutCallingBackend(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "enable iface", path: "/cgi-bin/luci/istore/wireless/enable-iface/"},
		{name: "set device power", path: "/cgi-bin/luci/istore/wireless/set-device-power/"},
		{name: "edit iface", path: "/cgi-bin/luci/istore/wireless/edit-iface/"},
		{name: "setup", path: "/cgi-bin/luci/istore/wireless/setup/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeWirelessBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestWirelessRoute(t, router, http.MethodPost, tt.path, `{"broken":`, true)

			if len(backend.calls) != 0 {
				t.Fatalf("expected backend not to be called, got %#v", backend.calls)
			}
			requireWirelessEnvelopeNotCode(t, resp, 0)
			requireWirelessEnvelopeError(t, resp, "Invalid request")
		})
	}
}

func TestRegisterWirelessRoutesRejectTrailingPostJSONWithoutCallingBackend(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "enable iface", path: "/cgi-bin/luci/istore/wireless/enable-iface/"},
		{name: "set device power", path: "/cgi-bin/luci/istore/wireless/set-device-power/"},
		{name: "edit iface", path: "/cgi-bin/luci/istore/wireless/edit-iface/"},
		{name: "setup", path: "/cgi-bin/luci/istore/wireless/setup/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeWirelessBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestWirelessRoute(t, router, http.MethodPost, tt.path, `{"ok":true} {"extra":true}`, true)

			if len(backend.calls) != 0 {
				t.Fatalf("expected backend not to be called, got %#v", backend.calls)
			}
			requireWirelessEnvelopeNotCode(t, resp, 0)
			requireWirelessEnvelopeError(t, resp, "Invalid request")
		})
	}
}

func requireWirelessEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

func requireWirelessEnvelopeNotCode(t *testing.T, resp map[string]any, notWant int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) == notWant {
		t.Fatalf("expected success code not to be %d in %#v", notWant, resp)
	}
}

func requireWirelessEnvelopeError(t *testing.T, resp map[string]any, want string) {
	t.Helper()

	got, ok := resp["error"].(string)
	if !ok {
		t.Fatalf("expected error string in response, got %#v", resp)
	}
	if got != want {
		t.Fatalf("expected error %q, got %q in %#v", want, got, resp)
	}
}

func requireWirelessFirstNestedString(t *testing.T, resp map[string]any, path []string, key, want string) {
	t.Helper()

	value := wirelessNestedValue(t, resp, path)
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

func wirelessNestedValue(t *testing.T, resp map[string]any, path []string) any {
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

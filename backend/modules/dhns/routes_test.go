package dhns

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/models"
	dhnsevents "github.com/linkease/quick-start/istore-backend/modules/dhns/events"
)

type fakeBackend struct {
	disabled bool
	calls    []string
	changes  []models.DHNSChangeRequest
	dhcps    []models.DHNSDhcpValidRequest
}

func (backend *fakeBackend) record(call string, w http.ResponseWriter) {
	backend.calls = append(backend.calls, call)
	w.WriteHeader(http.StatusAccepted)
}

func (backend *fakeBackend) DhnsDisabled() bool {
	return backend.disabled
}

func (backend *fakeBackend) DhnsConnect(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	backend.record("connect", w)
}

func (backend *fakeBackend) DhnsProxy(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	backend.record("proxy", w)
}

func (backend *fakeBackend) DhnsForward(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	backend.record("forward", w)
}

func (backend *fakeBackend) HandleDhnsChange(evt models.DHNSChangeRequest) bool {
	backend.changes = append(backend.changes, evt)
	return dhnsevents.ShouldTriggerIfaceEvent(evt)
}

func (backend *fakeBackend) HandleDhcpValid(info models.DHNSDhcpValidRequest) {
	backend.dhcps = append(backend.dhcps, info)
}

func TestRegisterRoutesEnabledMapsHijackEndpointsToBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCall string
	}{
		{name: "connect", method: http.MethodGet, path: "/api/dhns/connect/", wantCall: "connect"},
		{name: "proxy", method: http.MethodGet, path: "/api/dhns/proxy/", wantCall: "proxy"},
		{name: "forward", method: http.MethodGet, path: "/api/dhns/forward/", wantCall: "forward"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			rec := requestRoute(router, tt.method, tt.path)

			if rec.Code != http.StatusAccepted {
				t.Fatalf("expected status %d, got %d", http.StatusAccepted, rec.Code)
			}
			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
		})
	}
}

func TestRegisterRoutesDhnsChangeDecodesJSONAndWritesOK(t *testing.T) {
	tests := []struct {
		name string
		body string
		want models.DHNSChangeRequest
	}{
		{
			name: "iface event",
			body: `{"action":"ifaceEvent","params":["up","wan"]}`,
			want: models.DHNSChangeRequest{Action: "ifaceEvent", Params: []string{"up", "wan"}},
		},
		{
			name: "uci change",
			body: `{"action":"uciChange"}`,
			want: models.DHNSChangeRequest{Action: "uciChange"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			rec := requestRouteWithBody(router, http.MethodPost, "/api/dhns/dhnsChange/", tt.body)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
			}
			if rec.Body.String() != "OK" {
				t.Fatalf("expected OK body, got %q", rec.Body.String())
			}
			requireJSONEqual(t, backend.changes, []models.DHNSChangeRequest{tt.want})
		})
	}
}

func TestRegisterRoutesDhnsChangeRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed json", body: `{`, want: "unexpected EOF\n"},
		{name: "invalid event", body: `{"action":"ifaceEvent","params":["sideways"]}`, want: "error event\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			rec := requestRouteWithBody(router, http.MethodPost, "/api/dhns/dhnsChange/", tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
			if rec.Body.String() != tt.want {
				t.Fatalf("expected body %q, got %q", tt.want, rec.Body.String())
			}
		})
	}
}

func TestRegisterRoutesDhcpValidDecodesJSONAndWritesSuccess(t *testing.T) {
	backend := &fakeBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	rec := requestRouteWithBody(router, http.MethodPost, "/api/dhns/dhcpValid/", `{"ip":"192.168.1.10","gateway":"192.168.1.1","subnet":"255.255.255.0","dns":"1.1.1.1"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if rec.Body.String() != "Found DHCP Server" {
		t.Fatalf("expected DHCP success body, got %q", rec.Body.String())
	}
	requireJSONEqual(t, backend.dhcps, []models.DHNSDhcpValidRequest{{
		Ip:      "192.168.1.10",
		Gateway: "192.168.1.1",
		Subnet:  "255.255.255.0",
		Dns:     "1.1.1.1",
	}})
}

func TestRegisterRoutesDhcpValidRejectsInvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "malformed json", body: `{`, want: "unexpected EOF\n"},
		{name: "missing required field", body: `{"ip":"192.168.1.10","subnet":"255.255.255.0"}`, want: "empty ip\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			rec := requestRouteWithBody(router, http.MethodPost, "/api/dhns/dhcpValid/", tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
			}
			if rec.Body.String() != tt.want {
				t.Fatalf("expected body %q, got %q", tt.want, rec.Body.String())
			}
			if len(backend.dhcps) != 0 {
				t.Fatalf("expected no DHCP backend calls, got %#v", backend.dhcps)
			}
		})
	}
}

func TestRegisterRoutesDisabledReturnsNotImplementedWithoutCallingBackend(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "connect", method: http.MethodGet, path: "/api/dhns/connect/"},
		{name: "proxy", method: http.MethodGet, path: "/api/dhns/proxy/"},
		{name: "forward", method: http.MethodGet, path: "/api/dhns/forward/"},
		{name: "change", method: http.MethodPost, path: "/api/dhns/dhnsChange/"},
		{name: "dhcp valid", method: http.MethodPost, path: "/api/dhns/dhcpValid/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeBackend{disabled: true}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			rec := requestRoute(router, tt.method, tt.path)

			if rec.Code != http.StatusNotImplemented {
				t.Fatalf("expected status %d, got %d", http.StatusNotImplemented, rec.Code)
			}
			if rec.Body.String() != "ServiceDisabled\n" {
				t.Fatalf("expected ServiceDisabled body, got %q", rec.Body.String())
			}
			if len(backend.calls) != 0 {
				t.Fatalf("expected no backend calls, got %#v", backend.calls)
			}
		})
	}
}

func requestRoute(router *httprouter.Router, method, path string) *httptest.ResponseRecorder {
	return requestRouteWithBody(router, method, path, "")
}

func requestRouteWithBody(router *httprouter.Router, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	return rec
}

func requireJSONEqual[T any](t *testing.T, got, want T) {
	t.Helper()

	gotJSON, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal got: %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal want: %v", err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("expected %s, got %s", wantJSON, gotJSON)
	}
}

package app

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

type fakeAppBackend struct {
	err error

	calls       []string
	checkReqs   []models.AppCheckRequest
	installReqs []models.AppInstallRequest
}

func (backend *fakeAppBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeAppBackend) CheckApp(ctx context.Context, req models.AppCheckRequest) (*models.AppCheckResponse, error) {
	backend.record("check")
	backend.checkReqs = append(backend.checkReqs, req)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.AppCheckResponse{
		Result: &models.AppCheckResponseResult{Name: req.Name, Status: "installed"},
	}, nil
}

func (backend *fakeAppBackend) InstallAppPackage(ctx context.Context, req models.AppInstallRequest) (*models.SDKNormalResponse, error) {
	backend.record("install")
	backend.installReqs = append(backend.installReqs, req)
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeAppBackend) ListInstalledApps(ctx context.Context) (models.AppInstalledListResponse, error) {
	backend.record("installList")
	if backend.err != nil {
		return nil, backend.err
	}
	return models.AppInstalledListResponse{
		{Name: "installed-a", Title: "Installed A"},
	}, nil
}

func TestRegisterAppRoutesMapsEndpointsToBackendMethods(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantCall   string
		assertions func(t *testing.T, body []byte, backend *fakeAppBackend)
	}{
		{
			name:     "check",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/app/check/",
			body:     `{"name":"app-a","checkRunning":true}`,
			wantCall: "check",
			assertions: func(t *testing.T, body []byte, backend *fakeAppBackend) {
				if len(backend.checkReqs) != 1 {
					t.Fatalf("expected one check request, got %#v", backend.checkReqs)
				}
				if backend.checkReqs[0].Name != "app-a" || !backend.checkReqs[0].CheckRunning {
					t.Fatalf("expected check request for running app-a, got %#v", backend.checkReqs[0])
				}
				resp := decodeAppObject(t, body)
				requireNestedString(t, resp, []string{"result", "name"}, "app-a")
				requireNestedString(t, resp, []string{"result", "status"}, "installed")
			},
		},
		{
			name:     "install",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/app/install/",
			body:     `{"name":"app-a"}`,
			wantCall: "install",
			assertions: func(t *testing.T, body []byte, backend *fakeAppBackend) {
				if len(backend.installReqs) != 1 {
					t.Fatalf("expected one install request, got %#v", backend.installReqs)
				}
				if backend.installReqs[0].Name != "app-a" {
					t.Fatalf("expected install request for app-a, got %#v", backend.installReqs[0])
				}
				resp := decodeAppObject(t, body)
				requireEnvelopeCode(t, resp, 0)
			},
		},
		{
			name:     "install list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/app/install-list/",
			wantCall: "installList",
			assertions: func(t *testing.T, body []byte, backend *fakeAppBackend) {
				items := decodeAppList(t, body)
				if len(items) != 1 || items[0]["name"] != "installed-a" {
					t.Fatalf("expected installed app list, got %#v", items)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeAppBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			body := requestAppRoute(t, router, tt.method, tt.path, tt.body, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
			tt.assertions(t, body, backend)
		})
	}
}

func TestRegisterAppRoutesInstallListErrorReturnsEmptyList(t *testing.T) {
	backend := &fakeAppBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	body := requestAppRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/app/install-list/", "", true)

	if len(backend.calls) != 1 || backend.calls[0] != "installList" {
		t.Fatalf("expected installList backend call, got %#v", backend.calls)
	}
	items := decodeAppList(t, body)
	if len(items) != 0 {
		t.Fatalf("expected empty installed app list, got %#v", items)
	}
}

func TestRegisterAppRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeAppBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	body := requestAppRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/app/check/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	resp := decodeAppSDKNormalResponse(t, body)
	if resp.Success == nil || int64(*resp.Success) != httpapi.ForbiddenError {
		t.Fatalf("expected forbidden success code %d, got %#v", httpapi.ForbiddenError, resp.Success)
	}
}

func TestRegisterAppRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeAppBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	body := requestAppRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/app/check/", `{"name":"app-a"}`, true)

	if len(backend.calls) != 1 || backend.calls[0] != "check" {
		t.Fatalf("expected check backend call, got %#v", backend.calls)
	}
	resp := decodeAppSDKNormalResponse(t, body)
	if resp.Success == nil || int64(*resp.Success) != httpapi.GeneralError {
		t.Fatalf("expected general error success code %d, got %#v", httpapi.GeneralError, resp.Success)
	}
}

func TestRegisterAppRoutesRejectsMalformedJSON(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "check malformed",
			path: "/cgi-bin/luci/istore/app/check/",
			body: "{",
		},
		{
			name: "check trailing data",
			path: "/cgi-bin/luci/istore/app/check/",
			body: `{"name":"app-a"} trailing`,
		},
		{
			name: "install malformed",
			path: "/cgi-bin/luci/istore/app/install/",
			body: "{",
		},
		{
			name: "install trailing data",
			path: "/cgi-bin/luci/istore/app/install/",
			body: `{"name":"app-a"} trailing`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeAppBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			body := requestAppRoute(t, router, http.MethodPost, tt.path, tt.body, true)

			if len(backend.calls) != 0 {
				t.Fatalf("expected backend not to be called, got %#v", backend.calls)
			}
			resp := decodeAppSDKNormalResponse(t, body)
			if resp.Success == nil || int64(*resp.Success) != httpapi.GeneralError {
				t.Fatalf("expected general error success code %d, got %#v", httpapi.GeneralError, resp.Success)
			}
		})
	}
}

func requestAppRoute(t *testing.T, router *httprouter.Router, method, path string, body string, withSID bool) []byte {
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
	return rec.Body.Bytes()
}

func decodeAppObject(t *testing.T, body []byte) map[string]any {
	t.Helper()

	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response object: %v", err)
	}
	return resp
}

func decodeAppSDKNormalResponse(t *testing.T, body []byte) models.SDKNormalResponse {
	t.Helper()

	var resp models.SDKNormalResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode SDK normal response: %v", err)
	}
	return resp
}

func decodeAppList(t *testing.T, body []byte) []map[string]any {
	t.Helper()

	var resp []map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("decode response list: %v; body=%s", err, body)
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

func requireNestedString(t *testing.T, resp map[string]any, path []string, want string) {
	t.Helper()

	var current any = resp
	for _, key := range path {
		obj, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("expected object before %q, got %#v", key, current)
		}
		current = obj[key]
	}
	got, ok := current.(string)
	if !ok || got != want {
		t.Fatalf("expected %v to be %q, got %#v", path, want, current)
	}
}

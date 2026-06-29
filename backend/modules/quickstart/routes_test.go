package quickstart

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

type fakeQuickstartBackend struct {
	err error

	calls      []string
	getReqs    []models.QuickstartGetConfigRequest
	setReqs    []models.QuickstartConfigRequest
	deleteReqs []models.QuickstartDeleteConfigRequest
}

func (backend *fakeQuickstartBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeQuickstartBackend) GetQuickstartConfig(ctx context.Context, req models.QuickstartGetConfigRequest) (*models.QuickstartConfigResponse, error) {
	backend.record("get")
	backend.getReqs = append(backend.getReqs, req)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.QuickstartConfigResponse{
		Result: &models.QuickstartConfigResponseResult{Key: "dockerdir", Type: "option", Values: []string{"/mnt/data"}},
	}, nil
}

func (backend *fakeQuickstartBackend) SetQuickstartConfig(ctx context.Context, req models.QuickstartConfigRequest) (*models.SDKNormalResponse, error) {
	backend.record("set")
	backend.setReqs = append(backend.setReqs, req)
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeQuickstartBackend) DeleteQuickstartConfig(ctx context.Context, req models.QuickstartDeleteConfigRequest) (*models.SDKNormalResponse, error) {
	backend.record("delete")
	backend.deleteReqs = append(backend.deleteReqs, req)
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func TestRegisterQuickstartRoutesGetAndSetAliasesCallSameBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		body     string
		wantCall string
	}{
		{
			name:     "get legacy",
			path:     "/cgi-bin/luci/istore/quickstart/get/",
			body:     `{"key":"dockerdir"}`,
			wantCall: "get",
		},
		{
			name:     "get user alias",
			path:     "/cgi-bin/luci/istore/u/quickstart/get/",
			body:     `{"key":"dockerdir"}`,
			wantCall: "get",
		},
		{
			name:     "set legacy",
			path:     "/cgi-bin/luci/istore/quickstart/set/",
			body:     `{"key":"dockerdir","type":"list","values":["/mnt/data"]}`,
			wantCall: "set",
		},
		{
			name:     "set user alias",
			path:     "/cgi-bin/luci/istore/u/quickstart/set/",
			body:     `{"key":"dockerdir","type":"list","values":["/mnt/data"]}`,
			wantCall: "set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeQuickstartBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestQuickstartRoute(t, router, http.MethodPost, tt.path, tt.body, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
			switch tt.wantCall {
			case "get":
				if len(backend.getReqs) != 1 || backend.getReqs[0].Key != "dockerdir" {
					t.Fatalf("unexpected get request: %#v", backend.getReqs)
				}
			case "set":
				if len(backend.setReqs) != 1 {
					t.Fatalf("unexpected set request: %#v", backend.setReqs)
				}
				req := backend.setReqs[0]
				if req.Key != "dockerdir" || req.Type != "list" || len(req.Values) != 1 || req.Values[0] != "/mnt/data" {
					t.Fatalf("unexpected set request: %#v", backend.setReqs)
				}
			}
			requireQuickstartSuccessResponse(t, resp)
		})
	}
}

func TestRegisterQuickstartRoutesDeleteDecodesTypedRequest(t *testing.T) {
	backend := &fakeQuickstartBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestQuickstartRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/quickstart/delete/", `{"key":"dockerdir"}`, true)

	if len(backend.calls) != 1 || backend.calls[0] != "delete" {
		t.Fatalf("expected delete backend call, got %#v", backend.calls)
	}
	if len(backend.deleteReqs) != 1 || backend.deleteReqs[0].Key != "dockerdir" {
		t.Fatalf("unexpected delete request: %#v", backend.deleteReqs)
	}
	requireEnvelopeCode(t, resp, 0)
}

func TestRegisterQuickstartRoutesMalformedJSONReturnsErrorWithoutBackendCall(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "get incomplete",
			path: "/cgi-bin/luci/istore/quickstart/get/",
			body: `{`,
		},
		{
			name: "get trailing",
			path: "/cgi-bin/luci/istore/quickstart/get/",
			body: `{"key":"dockerdir"} trailing`,
		},
		{
			name: "set incomplete",
			path: "/cgi-bin/luci/istore/quickstart/set/",
			body: `{`,
		},
		{
			name: "set trailing",
			path: "/cgi-bin/luci/istore/quickstart/set/",
			body: `{"key":"dockerdir","type":"list","values":["/mnt/data"]} trailing`,
		},
		{
			name: "delete incomplete",
			path: "/cgi-bin/luci/istore/quickstart/delete/",
			body: `{`,
		},
		{
			name: "delete trailing",
			path: "/cgi-bin/luci/istore/quickstart/delete/",
			body: `{"key":"dockerdir"} trailing`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeQuickstartBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestQuickstartRoute(t, router, http.MethodPost, tt.path, tt.body, true)

			if len(backend.calls) != 0 {
				t.Fatalf("expected backend not to be called, got %#v", backend.calls)
			}
			requireEnvelopeCode(t, resp, httpapi.GeneralError)
		})
	}
}

func TestRegisterQuickstartRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeQuickstartBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestQuickstartRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/quickstart/get/", `{}`, false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterQuickstartRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeQuickstartBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestQuickstartRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/quickstart/set/", `{}`, true)

	if len(backend.calls) != 1 || backend.calls[0] != "set" {
		t.Fatalf("expected set backend call, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestQuickstartRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func requireQuickstartSuccessResponse(t *testing.T, resp map[string]any) {
	t.Helper()

	if _, ok := resp["result"]; ok {
		return
	}
	requireEnvelopeCode(t, resp, 0)
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

package smart

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

type fakeSmartBackend struct {
	err error

	calls      []string
	configGets int
	configPost int
	testReq    *models.SmartTestRequest
}

func (backend *fakeSmartBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeSmartBackend) GetSmartList(ctx context.Context) (*models.SmartListResponse, error) {
	backend.record("list")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.SmartListResponse{
		Result: &models.SmartListResponseResult{Disks: []*models.SmartInfo{{Name: "disk-a"}}},
	}, nil
}

func (backend *fakeSmartBackend) GetSmartLog(ctx context.Context) (*models.SmartLogResponse, error) {
	backend.record("log")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.SmartLogResponse{
		Result: &models.SmartLogResponseResult{Result: "smart log"},
	}, nil
}

func (backend *fakeSmartBackend) GetSmartConfig(ctx context.Context) (*models.SmartConfigResponse, error) {
	backend.record("configGet")
	backend.configGets++
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.SmartConfigResponse{
		Result: &models.SmartConfigResponseResult{Devices: []*models.SmartConfigDevice{}, Tasks: []*models.SmartConfigTask{}},
	}, nil
}

func (backend *fakeSmartBackend) PostSmartConfig(ctx context.Context, req models.SmartConfigRequest) (*models.SmartConfigResponse, error) {
	backend.record("configPost")
	backend.configPost++
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.SmartConfigResponse{
		Result: &models.SmartConfigResponseResult{Devices: []*models.SmartConfigDevice{}, Tasks: []*models.SmartConfigTask{}},
	}, nil
}

func (backend *fakeSmartBackend) PostSmartTest(ctx context.Context, req models.SmartTestRequest) (*models.SmartTestResponse, error) {
	backend.record("test")
	backend.testReq = &req
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.SmartTestResponse{
		Result: &models.SmartTestResponseResult{Result: "started"},
	}, nil
}

func (backend *fakeSmartBackend) PostSmartTestResult(ctx context.Context, req models.SmartTestResultRequest) (*models.SmartTestResultResponse, error) {
	backend.record("testResult")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.SmartTestResultResponse{
		Result: &models.SmartTestResultResponseResult{Result: "test result"},
	}, nil
}

func (backend *fakeSmartBackend) PostSmartAttributeResult(ctx context.Context, req models.SmartAttributeResultRequest) (*models.SmartAttributeResultResponse, error) {
	backend.record("attributeResult")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.SmartAttributeResultResponse{
		Result: &models.SmartAttributeResultResponseResult{Result: "attribute result"},
	}, nil
}

func (backend *fakeSmartBackend) PostSmartExtendResult(ctx context.Context, req models.SmartExtendResultRequest) (*models.SmartExtendResultResponse, error) {
	backend.record("extendResult")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.SmartExtendResultResponse{
		Result: &models.SmartExtendResultResponseResult{Result: "extend result"},
	}, nil
}

func TestRegisterSmartRoutesAliasesCallSameBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCall string
	}{
		{
			name:     "list legacy",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/smart/list/",
			wantCall: "list",
		},
		{
			name:     "list user alias",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/u/smart/list/",
			wantCall: "list",
		},
		{
			name:     "test legacy",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/smart/test/",
			wantCall: "test",
		},
		{
			name:     "test user alias",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/u/smart/test/",
			wantCall: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeSmartBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestSmartRoute(t, router, tt.method, tt.path, true)

			requireSmartCalls(t, backend, tt.wantCall)
		})
	}
}

func TestRegisterSmartRoutesPassesDecodedRequestToBackend(t *testing.T) {
	backend := &fakeSmartBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	requestSmartRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/smart/test/", true)

	if backend.testReq == nil {
		t.Fatal("expected smart test request to be recorded")
	}
	if backend.testReq.DevicePath != "/dev/sda" {
		t.Fatalf("DevicePath = %q, want /dev/sda", backend.testReq.DevicePath)
	}
}

func TestRegisterSmartRoutesConfigGetAndPostUseDistinctBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCall string
	}{
		{
			name:     "config get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/smart/config/",
			wantCall: "configGet",
		},
		{
			name:     "config post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/smart/config/",
			wantCall: "configPost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeSmartBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestSmartRoute(t, router, tt.method, tt.path, true)

			requireSmartCalls(t, backend, tt.wantCall)
		})
	}
}

func TestRegisterSmartRoutesLogUsesBackendMethod(t *testing.T) {
	backend := &fakeSmartBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	requestSmartRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/smart/log/", true)

	requireSmartCalls(t, backend, "log")
}

func TestRegisterSmartRoutesResultEndpointsUseDistinctBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantCall string
	}{
		{
			name:     "test result",
			path:     "/cgi-bin/luci/istore/smart/test/result/",
			wantCall: "testResult",
		},
		{
			name:     "attribute result",
			path:     "/cgi-bin/luci/istore/smart/attribute/result/",
			wantCall: "attributeResult",
		},
		{
			name:     "extend result",
			path:     "/cgi-bin/luci/istore/smart/extend/result/",
			wantCall: "extendResult",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeSmartBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestSmartRoute(t, router, http.MethodPost, tt.path, true)

			requireSmartCalls(t, backend, tt.wantCall)
		})
	}
}

func TestRegisterSmartRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeSmartBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestSmartRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/smart/list/", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireSmartEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterSmartRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeSmartBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestSmartRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/smart/list/", true)

	requireSmartCalls(t, backend, "list")
	requireSmartEnvelopeCode(t, resp, httpapi.GeneralError)
}

func TestRegisterSmartRoutesRejectsInvalidPostJSONBeforeBackend(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "config malformed json",
			path: "/cgi-bin/luci/istore/smart/config/",
			body: `{"global":`,
		},
		{
			name: "config trailing json",
			path: "/cgi-bin/luci/istore/smart/config/",
			body: `{"global":null} {}`,
		},
		{
			name: "test malformed json",
			path: "/cgi-bin/luci/istore/smart/test/",
			body: `{"devicePath":`,
		},
		{
			name: "test trailing json",
			path: "/cgi-bin/luci/istore/smart/test/",
			body: `{"devicePath":"/dev/sda"} {}`,
		},
		{
			name: "user test malformed json",
			path: "/cgi-bin/luci/istore/u/smart/test/",
			body: `{"devicePath":`,
		},
		{
			name: "user test trailing json",
			path: "/cgi-bin/luci/istore/u/smart/test/",
			body: `{"devicePath":"/dev/sda"} {}`,
		},
		{
			name: "test result malformed json",
			path: "/cgi-bin/luci/istore/smart/test/result/",
			body: `{"devicePath":`,
		},
		{
			name: "test result trailing json",
			path: "/cgi-bin/luci/istore/smart/test/result/",
			body: `{"devicePath":"/dev/sda"} {}`,
		},
		{
			name: "attribute result malformed json",
			path: "/cgi-bin/luci/istore/smart/attribute/result/",
			body: `{"devicePath":`,
		},
		{
			name: "attribute result trailing json",
			path: "/cgi-bin/luci/istore/smart/attribute/result/",
			body: `{"devicePath":"/dev/sda"} {}`,
		},
		{
			name: "extend result malformed json",
			path: "/cgi-bin/luci/istore/smart/extend/result/",
			body: `{"devicePath":`,
		},
		{
			name: "extend result trailing json",
			path: "/cgi-bin/luci/istore/smart/extend/result/",
			body: `{"devicePath":"/dev/sda"} {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeSmartBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			rec := requestSmartRouteWithBody(t, router, http.MethodPost, tt.path, tt.body, true)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
			}
			var resp map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			requireSmartEnvelopeCode(t, resp, httpapi.GeneralError)
			requireSmartCalls(t, backend)
		})
	}
}

func requestSmartRoute(t *testing.T, router *httprouter.Router, method, path string, withSID bool) map[string]any {
	t.Helper()

	rec := requestSmartRouteWithBody(t, router, method, path, `{"devicePath":"/dev/sda"}`, withSID)

	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s expected status 200, got %d", method, path, rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func requestSmartRouteWithBody(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if withSID {
		req.Header.Set("X-Forwarded-Sid", "sid-1")
	}
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	return rec
}

func requireSmartCalls(t *testing.T, backend *fakeSmartBackend, want ...string) {
	t.Helper()

	if len(backend.calls) != len(want) {
		t.Fatalf("expected calls %#v, got %#v", want, backend.calls)
	}
	for i := range want {
		if backend.calls[i] != want[i] {
			t.Fatalf("expected calls %#v, got %#v", want, backend.calls)
		}
	}
}

func requireSmartEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

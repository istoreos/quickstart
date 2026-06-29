package guideddns

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGuideDDNSBackend struct {
	err error

	calls       []string
	requestPath string
	requestBody string
}

func (backend *fakeGuideDDNSBackend) record(call string, r *http.Request) {
	backend.calls = append(backend.calls, call)
	if r == nil {
		return
	}
	backend.requestPath = r.URL.Path
	body, err := io.ReadAll(r.Body)
	if err == nil {
		backend.requestBody = string(body)
	}
}

func (backend *fakeGuideDDNSBackend) GetGuideDdns(ctx context.Context) (*models.GuideDdnsResponse, error) {
	backend.record("ddns-get", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDdnsResponse{
		Result: &models.GuideDdnsResponseResult{IPV4Domain: "demo.example.com"},
	}, nil
}

func (backend *fakeGuideDDNSBackend) PostGuideDdns(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("ddns-post", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return guideDDNSSuccessResponse(), nil
}

func (backend *fakeGuideDDNSBackend) PostGuideDdnsto(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("ddnsto-write", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return guideDDNSSuccessResponse(), nil
}

func (backend *fakeGuideDDNSBackend) PostGuideDdnstoAddress(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("ddnsto-address", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return guideDDNSSuccessResponse(), nil
}

func (backend *fakeGuideDDNSBackend) GetGuideDdnstoConfig(ctx context.Context) (*models.GuideDdnstoConfigResponse, error) {
	backend.record("ddnsto-config", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDdnstoConfigResponse{
		Result: &models.GuideDdnstoConfigResponseResult{DeviceID: "device-1", NetAddr: "https://demo.example.com"},
	}, nil
}

func TestRegisterGuideDDNSRoutesDDNSGetAliasesCallSameBackendMethod(t *testing.T) {
	for _, path := range []string{
		"/cgi-bin/luci/istore/guide/ddns/",
		"/cgi-bin/luci/istore/u/guide/ddns/",
	} {
		t.Run(path, func(t *testing.T) {
			backend := &fakeGuideDDNSBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestGuideDDNSRoute(t, router, http.MethodGet, path, "", true)

			if len(backend.calls) != 1 || backend.calls[0] != "ddns-get" {
				t.Fatalf("expected ddns-get backend call, got %#v", backend.calls)
			}
			if _, ok := resp["result"]; !ok {
				t.Fatalf("expected result response, got %#v", resp)
			}
		})
	}
}

func TestRegisterGuideDDNSRoutesDDNSPostAliasesCallSameBackendMethod(t *testing.T) {
	for _, path := range []string{
		"/cgi-bin/luci/istore/guide/ddns/",
		"/cgi-bin/luci/istore/u/guide/ddns/",
	} {
		t.Run(path, func(t *testing.T) {
			backend := &fakeGuideDDNSBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestGuideDDNSRoute(t, router, http.MethodPost, path, `{"domain":"demo.example.com"}`, true)

			if len(backend.calls) != 1 || backend.calls[0] != "ddns-post" {
				t.Fatalf("expected ddns-post backend call, got %#v", backend.calls)
			}
			requireEnvelopeCode(t, resp, 0)
		})
	}
}

func TestRegisterGuideDDNSRoutesDDNSTORoutesCallDistinctBackendMethods(t *testing.T) {
	tests := []struct {
		method   string
		path     string
		body     string
		wantCall string
	}{
		{
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/ddnsto/",
			body:     `{"token":"token-1"}`,
			wantCall: "ddnsto-write",
		},
		{
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/ddnsto/address/",
			body:     `{"address":"https://demo.example.com"}`,
			wantCall: "ddnsto-address",
		},
		{
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/ddnsto/config/",
			wantCall: "ddnsto-config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.wantCall, func(t *testing.T) {
			backend := &fakeGuideDDNSBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestGuideDDNSRoute(t, router, tt.method, tt.path, tt.body, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected backend call %q, got %#v", tt.wantCall, backend.calls)
			}
			if tt.wantCall == "ddnsto-config" {
				if _, ok := resp["result"]; !ok {
					t.Fatalf("expected result response, got %#v", resp)
				}
				return
			}
			requireEnvelopeCode(t, resp, 0)
		})
	}
}

func TestRegisterGuideDDNSRoutesPostPassesOriginalRequestPath(t *testing.T) {
	backend := &fakeGuideDDNSBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	body := `{"domain":"demo.example.com"}`
	resp := requestGuideDDNSRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/u/guide/ddns/", body, true)

	if len(backend.calls) != 1 || backend.calls[0] != "ddns-post" {
		t.Fatalf("expected ddns-post backend call, got %#v", backend.calls)
	}
	if backend.requestPath != "/cgi-bin/luci/istore/u/guide/ddns/" {
		t.Fatalf("expected original request path, got %q", backend.requestPath)
	}
	if backend.requestBody != body {
		t.Fatalf("expected original request body %q, got %q", body, backend.requestBody)
	}
	requireEnvelopeCode(t, resp, 0)
}

func TestRegisterGuideDDNSRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeGuideDDNSBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestGuideDDNSRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/guide/ddns/", `{}`, false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, guideDDNSForbiddenError)
}

func TestRegisterGuideDDNSRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeGuideDDNSBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestGuideDDNSRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/guide/ddnsto/", `{}`, true)

	if len(backend.calls) != 1 || backend.calls[0] != "ddnsto-write" {
		t.Fatalf("expected ddnsto-write backend call, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, guideDDNSGeneralError)
}

func requestGuideDDNSRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func guideDDNSSuccessResponse() *models.SDKNormalResponse {
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}
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

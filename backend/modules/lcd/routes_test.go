package lcd

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
)

type fakeLCDBackend struct {
	err error

	calls []string
}

func (backend *fakeLCDBackend) GetLCDST7789(ctx context.Context, r *http.Request) (any, error) {
	backend.calls = append(backend.calls, "st7789")
	if backend.err != nil {
		return nil, backend.err
	}
	return map[string]any{"ip": "192.168.1.1", "cpu": 12}, nil
}

func (backend *fakeLCDBackend) GetLcdSimple(ctx context.Context, r *http.Request) (any, error) {
	backend.calls = append(backend.calls, "simple")
	if backend.err != nil {
		return nil, backend.err
	}
	return map[string]any{"ipv4": "192.168.1.2", "cpu": 34}, nil
}

func TestRegisterRoutesST7789SuccessSetsConnectionClose(t *testing.T) {
	backend := &fakeLCDBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	rec := requestLCDRoute(t, router, "/api/lcd/st7789/")

	if len(backend.calls) != 1 || backend.calls[0] != "st7789" {
		t.Fatalf("expected st7789 backend call, got %#v", backend.calls)
	}
	if got := rec.Header().Get("Connection"); got != "close" {
		t.Fatalf("expected Connection close header, got %q", got)
	}
	requireLCDField(t, rec, "ip", "192.168.1.1")
}

func TestRegisterRoutesSimpleSuccess(t *testing.T) {
	backend := &fakeLCDBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	rec := requestLCDRoute(t, router, "/api/lcd/simple/")

	if len(backend.calls) != 1 || backend.calls[0] != "simple" {
		t.Fatalf("expected simple backend call, got %#v", backend.calls)
	}
	requireLCDField(t, rec, "ipv4", "192.168.1.2")
}

func TestRegisterRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeLCDBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	rec := requestLCDRoute(t, router, "/api/lcd/simple/")

	if len(backend.calls) != 1 || backend.calls[0] != "simple" {
		t.Fatalf("expected simple backend call, got %#v", backend.calls)
	}
	requireLCDEnvelopeCode(t, rec, httpapi.GeneralError)
}

func requestLCDRoute(t *testing.T, router *httprouter.Router, path string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, strings.NewReader("{}"))
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s expected status 200, got %d", path, rec.Code)
	}
	return rec
}

func requireLCDField(t *testing.T, rec *httptest.ResponseRecorder, name string, want any) {
	t.Helper()

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got := resp[name]; got != want {
		t.Fatalf("expected %s %v, got %v in %#v", name, want, got, resp)
	}
}

func requireLCDEnvelopeCode(t *testing.T, rec *httptest.ResponseRecorder, want int64) {
	t.Helper()

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

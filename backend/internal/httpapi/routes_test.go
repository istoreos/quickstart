package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/models"
)

func TestAuthenticatedJSONRequiresForwardedSid(t *testing.T) {
	called := false
	handle := AuthenticatedJSON(func(ctx context.Context, r *http.Request) (any, error) {
		called = true
		return map[string]string{"ok": "true"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handle(rec, req, httprouter.Params{})

	if called {
		t.Fatal("handler was called without X-Forwarded-Sid")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp models.SDKNormalResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Success == nil || int64(*resp.Success) != ForbiddenError {
		t.Fatalf("expected forbidden success code %d, got %#v", ForbiddenError, resp.Success)
	}
}

func TestAuthenticatedJSONWritesSuccess(t *testing.T) {
	handle := AuthenticatedJSON(func(ctx context.Context, r *http.Request) (any, error) {
		return map[string]string{"status": "ok"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Sid", "sid-1")
	rec := httptest.NewRecorder()

	handle(rec, req, httprouter.Params{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %#v", resp)
	}
}

func TestAuthenticatedJSONWritesErrorEnvelope(t *testing.T) {
	handle := AuthenticatedJSON(func(ctx context.Context, r *http.Request) (any, error) {
		return nil, errors.New("broken")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Forwarded-Sid", "sid-1")
	rec := httptest.NewRecorder()

	handle(rec, req, httprouter.Params{})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp models.SDKNormalResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Success == nil || int64(*resp.Success) != GeneralError {
		t.Fatalf("expected general error code %d, got %#v", GeneralError, resp.Success)
	}
}

func TestGetJSONAliasesRegistersEveryPath(t *testing.T) {
	router := httprouter.New()
	calls := 0

	GetJSONAliases(router, []string{"/one", "/two"}, func(ctx context.Context, r *http.Request) (any, error) {
		calls++
		return map[string]string{"path": r.URL.Path}, nil
	})

	for _, path := range []string{"/one", "/two"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Forwarded-Sid", "sid-1")
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("path %s expected status 200, got %d", path, rec.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("path %s decode response: %v", path, err)
		}
		if resp["path"] != path {
			t.Fatalf("path %s expected response path %q, got %#v", path, path, resp)
		}
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

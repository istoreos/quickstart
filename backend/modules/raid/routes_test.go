package raid

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

type fakeRaidBackend struct {
	err error

	calls       []string
	requestPath string
}

func (backend *fakeRaidBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeRaidBackend) PostRaidCreate(ctx context.Context, r *http.Request) (*models.NasDiskPartitionFormatResponse, error) {
	backend.record("create")
	backend.requestPath = r.URL.Path
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasDiskPartitionFormatResponse{}, nil
}

func (backend *fakeRaidBackend) PostRaidDelete(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("delete")
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeRaidBackend) PostRaidAdd(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("add")
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeRaidBackend) PostRaidRemove(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("remove")
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeRaidBackend) PostRaidDetail(ctx context.Context, r *http.Request) (*models.RaidDetailResponse, error) {
	backend.record("detail")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.RaidDetailResponse{
		Result: &models.RaidDetailResponseResult{Detail: "raid detail"},
	}, nil
}

func (backend *fakeRaidBackend) PostRaidRecover(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("recover")
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeRaidBackend) GetRaidList(ctx context.Context) (*models.RaidListResponse, error) {
	backend.record("list")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.RaidListResponse{
		Result: &models.RaidListResponseResult{Disks: []*models.NasDiskInfo{{Name: "md0"}}},
	}, nil
}

func (backend *fakeRaidBackend) PostRaidAutoFix(ctx context.Context) (*models.SDKNormalResponse, error) {
	backend.record("autofix")
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeRaidBackend) GetRaidCreateList(ctx context.Context) (*models.RaidCreateListResponse, error) {
	backend.record("createList")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.RaidCreateListResponse{
		Result: &models.RaidCreateListResponseResult{Members: []*models.RaidMemberInfo{{Path: "/dev/sda"}}},
	}, nil
}

func TestRegisterRaidRoutesSuccessRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantCall   string
		assertions func(t *testing.T, backend *fakeRaidBackend, resp map[string]any)
	}{
		{
			name:     "create",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/raid/create/",
			wantCall: "create",
			assertions: func(t *testing.T, backend *fakeRaidBackend, resp map[string]any) {
				if backend.requestPath != "/cgi-bin/luci/istore/raid/create/" {
					t.Fatalf("expected request path to be recorded, got %q", backend.requestPath)
				}
			},
		},
		{
			name:     "delete",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/raid/delete/",
			wantCall: "delete",
		},
		{
			name:     "add",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/raid/add/",
			wantCall: "add",
		},
		{
			name:     "remove",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/raid/remove/",
			wantCall: "remove",
		},
		{
			name:     "detail",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/raid/detail/",
			wantCall: "detail",
			assertions: func(t *testing.T, backend *fakeRaidBackend, resp map[string]any) {
				requireRaidNestedString(t, resp, []string{"result", "detail"}, "raid detail")
			},
		},
		{
			name:     "recover",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/raid/recover/",
			wantCall: "recover",
		},
		{
			name:     "list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/raid/list/",
			wantCall: "list",
			assertions: func(t *testing.T, backend *fakeRaidBackend, resp map[string]any) {
				requireRaidFirstNestedString(t, resp, []string{"result", "disks"}, "name", "md0")
			},
		},
		{
			name:     "autofix",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/raid/autofix/",
			wantCall: "autofix",
		},
		{
			name:     "create list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/raid/create/list/",
			wantCall: "createList",
			assertions: func(t *testing.T, backend *fakeRaidBackend, resp map[string]any) {
				requireRaidFirstNestedString(t, resp, []string{"result", "members"}, "path", "/dev/sda")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeRaidBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			resp := requestRaidRoute(t, router, tt.method, tt.path, "", true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
			if tt.assertions != nil {
				tt.assertions(t, backend, resp)
			}
		})
	}
}

func TestRegisterRaidRoutesAutoFixIsNotGet(t *testing.T) {
	backend := &fakeRaidBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	req := httptest.NewRequest(http.MethodGet, "/cgi-bin/luci/istore/raid/autofix/", nil)
	req.Header.Set("X-Forwarded-Sid", "sid-1")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK {
		t.Fatalf("expected GET autofix not to be registered")
	}
	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
}

func TestRegisterRaidRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeRaidBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestRaidRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/raid/list/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireRaidEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterRaidRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeRaidBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestRaidRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/raid/create/", "", true)

	if len(backend.calls) != 1 || backend.calls[0] != "create" {
		t.Fatalf("expected create backend call, got %#v", backend.calls)
	}
	requireRaidEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestRaidRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func requireRaidEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

func requireRaidNestedString(t *testing.T, resp map[string]any, path []string, want string) {
	t.Helper()

	value := raidNestedValue(t, resp, path)
	got, ok := value.(string)
	if !ok || got != want {
		t.Fatalf("expected %s to be %q, got %#v", strings.Join(path, "."), want, value)
	}
}

func requireRaidFirstNestedString(t *testing.T, resp map[string]any, path []string, key, want string) {
	t.Helper()

	value := raidNestedValue(t, resp, path)
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

func raidNestedValue(t *testing.T, resp map[string]any, path []string) any {
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

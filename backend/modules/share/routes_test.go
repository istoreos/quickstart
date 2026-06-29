package share

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

type fakeShareBackend struct {
	err error

	calls         []string
	userCreateReq *models.ShareUserCreateRequest
}

func (backend *fakeShareBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeShareBackend) GetShareUserList(ctx context.Context) (*models.ShareUserListResponse, error) {
	backend.record("userList")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.ShareUserListResponse{
		Result: &models.ShareUserListResponseResult{Users: []*models.ShareUserInfo{}},
	}, nil
}

func (backend *fakeShareBackend) PostShareUserCreate(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error) {
	backend.record("userCreate")
	backend.userCreateReq = &req
	if backend.err != nil {
		return nil, backend.err
	}
	return normalShareResponse(), nil
}

func (backend *fakeShareBackend) PostShareUserUpdate(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error) {
	backend.record("userUpdate")
	if backend.err != nil {
		return nil, backend.err
	}
	return normalShareResponse(), nil
}

func (backend *fakeShareBackend) PostShareUserDelete(ctx context.Context, req models.ShareUserDeleteRequest) (*models.SDKNormalResponse, error) {
	backend.record("userDelete")
	if backend.err != nil {
		return nil, backend.err
	}
	return normalShareResponse(), nil
}

func (backend *fakeShareBackend) GetShareServiceList(ctx context.Context) (*models.ShareServiceListResponse, error) {
	backend.record("serviceList")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.ShareServiceListResponse{
		Result: &models.ShareServiceListResponseResult{Services: []*models.ShareServiceInfo{}},
	}, nil
}

func (backend *fakeShareBackend) PostShareServiceCreate(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error) {
	backend.record("serviceCreate")
	if backend.err != nil {
		return nil, backend.err
	}
	return normalShareResponse(), nil
}

func (backend *fakeShareBackend) PostShareServiceUpdate(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error) {
	backend.record("serviceUpdate")
	if backend.err != nil {
		return nil, backend.err
	}
	return normalShareResponse(), nil
}

func (backend *fakeShareBackend) PostShareServiceDelete(ctx context.Context, req models.ShareServicDeleteRequest) (*models.SDKNormalResponse, error) {
	backend.record("serviceDelete")
	if backend.err != nil {
		return nil, backend.err
	}
	return normalShareResponse(), nil
}

func (backend *fakeShareBackend) GetShareWebdavConfig(ctx context.Context) (*models.ShareProtocolWebdavResponse, error) {
	backend.record("webdavGet")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.ShareProtocolWebdavResponse{Result: &models.ShareProtocolWebdavConfig{}}, nil
}

func (backend *fakeShareBackend) PostShareWebdavConfig(ctx context.Context, req models.ShareProtocolWebdavConfig) (*models.SDKNormalResponse, error) {
	backend.record("webdavPost")
	if backend.err != nil {
		return nil, backend.err
	}
	return normalShareResponse(), nil
}

func (backend *fakeShareBackend) GetShareSambaConfig(ctx context.Context) (*models.ShareProtocolSambaResponse, error) {
	backend.record("sambaGet")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.ShareProtocolSambaResponse{Result: &models.ShareProtocolSambaConfig{}}, nil
}

func (backend *fakeShareBackend) PostShareSambaConfig(ctx context.Context, req models.ShareProtocolSambaConfig) (*models.SDKNormalResponse, error) {
	backend.record("sambaPost")
	if backend.err != nil {
		return nil, backend.err
	}
	return normalShareResponse(), nil
}

func TestRegisterShareRoutesMapsEndpointsToBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCall string
	}{
		{
			name:     "user list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/share/user/list/",
			wantCall: "userList",
		},
		{
			name:     "user create",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/user/create/",
			wantCall: "userCreate",
		},
		{
			name:     "user update",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/user/update/",
			wantCall: "userUpdate",
		},
		{
			name:     "user delete",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/user/delete/",
			wantCall: "userDelete",
		},
		{
			name:     "service list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/share/service/list/",
			wantCall: "serviceList",
		},
		{
			name:     "service create",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/service/create/",
			wantCall: "serviceCreate",
		},
		{
			name:     "service update",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/service/update/",
			wantCall: "serviceUpdate",
		},
		{
			name:     "service delete",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/service/delete/",
			wantCall: "serviceDelete",
		},
		{
			name:     "webdav get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/share/protocol/webdav/",
			wantCall: "webdavGet",
		},
		{
			name:     "webdav post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/protocol/webdav/",
			wantCall: "webdavPost",
		},
		{
			name:     "samba get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/share/protocol/samba/",
			wantCall: "sambaGet",
		},
		{
			name:     "samba post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/protocol/samba/",
			wantCall: "sambaPost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeShareBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestShareRoute(t, router, tt.method, tt.path, true)

			requireShareCalls(t, backend, tt.wantCall)
		})
	}
}

func TestRegisterShareRoutesPassesDecodedRequestToBackend(t *testing.T) {
	backend := &fakeShareBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	requestShareRouteRecorder(t, router, http.MethodPost, "/cgi-bin/luci/istore/share/user/create/", `{"userName":"bob","password":"pw"}`, true)

	if backend.userCreateReq == nil {
		t.Fatal("expected user create request to be recorded")
	}
	if backend.userCreateReq.UserName != "bob" || backend.userCreateReq.Password != "pw" {
		t.Fatalf("unexpected user create request: %#v", backend.userCreateReq)
	}
}

func TestRegisterShareRoutesProtocolGetAndPostUseDistinctBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCall string
	}{
		{
			name:     "webdav get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/share/protocol/webdav/",
			wantCall: "webdavGet",
		},
		{
			name:     "webdav post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/protocol/webdav/",
			wantCall: "webdavPost",
		},
		{
			name:     "samba get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/share/protocol/samba/",
			wantCall: "sambaGet",
		},
		{
			name:     "samba post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/share/protocol/samba/",
			wantCall: "sambaPost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeShareBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestShareRoute(t, router, tt.method, tt.path, true)

			requireShareCalls(t, backend, tt.wantCall)
		})
	}
}

func TestRegisterShareRoutesRejectsInvalidPostJSONBeforeBackend(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
	}{
		{
			name: "user create malformed",
			path: "/cgi-bin/luci/istore/share/user/create/",
			body: `{"userName":`,
		},
		{
			name: "user create trailing",
			path: "/cgi-bin/luci/istore/share/user/create/",
			body: `{"userName":"bob","password":"pw"} trailing`,
		},
		{
			name: "user update malformed",
			path: "/cgi-bin/luci/istore/share/user/update/",
			body: `{"userName":`,
		},
		{
			name: "user update trailing",
			path: "/cgi-bin/luci/istore/share/user/update/",
			body: `{"userName":"bob","password":"pw"} trailing`,
		},
		{
			name: "user delete malformed",
			path: "/cgi-bin/luci/istore/share/user/delete/",
			body: `{"userName":`,
		},
		{
			name: "user delete trailing",
			path: "/cgi-bin/luci/istore/share/user/delete/",
			body: `{"userName":"bob"} trailing`,
		},
		{
			name: "service create malformed",
			path: "/cgi-bin/luci/istore/share/service/create/",
			body: `{"name":`,
		},
		{
			name: "service create trailing",
			path: "/cgi-bin/luci/istore/share/service/create/",
			body: `{"name":"docs","path":"/mnt/docs"} trailing`,
		},
		{
			name: "service update malformed",
			path: "/cgi-bin/luci/istore/share/service/update/",
			body: `{"name":`,
		},
		{
			name: "service update trailing",
			path: "/cgi-bin/luci/istore/share/service/update/",
			body: `{"name":"docs","path":"/mnt/docs"} trailing`,
		},
		{
			name: "service delete malformed",
			path: "/cgi-bin/luci/istore/share/service/delete/",
			body: `{"name":`,
		},
		{
			name: "service delete trailing",
			path: "/cgi-bin/luci/istore/share/service/delete/",
			body: `{"name":"docs"} trailing`,
		},
		{
			name: "webdav malformed",
			path: "/cgi-bin/luci/istore/share/protocol/webdav/",
			body: `{"port":`,
		},
		{
			name: "webdav trailing",
			path: "/cgi-bin/luci/istore/share/protocol/webdav/",
			body: `{"port":6087} trailing`,
		},
		{
			name: "samba malformed",
			path: "/cgi-bin/luci/istore/share/protocol/samba/",
			body: `{"workgroup":`,
		},
		{
			name: "samba trailing",
			path: "/cgi-bin/luci/istore/share/protocol/samba/",
			body: `{"workgroup":"WORKGROUP"} trailing`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeShareBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			rec := requestShareRouteRecorder(t, router, http.MethodPost, tt.path, tt.body, true)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d with body %s", rec.Code, rec.Body.String())
			}
			var resp map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			requireShareEnvelopeCode(t, resp, httpapi.GeneralError)
			if len(backend.calls) != 0 {
				t.Fatalf("expected backend not to be called, got %#v", backend.calls)
			}
		})
	}
}

func TestRegisterShareRoutesPostDecodesRequestBody(t *testing.T) {
	backend := &fakeShareBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	requestShareRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/share/service/update/", true)

	requireShareCalls(t, backend, "serviceUpdate")
}

func TestRegisterShareRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeShareBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestShareRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/share/user/list/", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireShareEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterShareRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeShareBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestShareRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/share/user/list/", true)

	requireShareCalls(t, backend, "userList")
	requireShareEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestShareRoute(t *testing.T, router *httprouter.Router, method, path string, withSID bool) map[string]any {
	t.Helper()

	rec := requestShareRouteRecorder(t, router, method, path, `{"name":"share"}`, withSID)

	if rec.Code != http.StatusOK {
		t.Fatalf("%s %s expected status 200, got %d", method, path, rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func requestShareRouteRecorder(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if withSID {
		req.Header.Set("X-Forwarded-Sid", "sid-1")
	}
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	return rec
}

func requireShareCalls(t *testing.T, backend *fakeShareBackend, want ...string) {
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

func requireShareEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

func normalShareResponse() *models.SDKNormalResponse {
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}
}

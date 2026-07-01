package nas

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/istoreos/quickstart/backend/internal/httpapi"
	"github.com/istoreos/quickstart/backend/models"
	"github.com/julienschmidt/httprouter"
)

type fakeNasBackend struct {
	err error

	calls       []string
	requestPath string
}

func (backend *fakeNasBackend) record(call string) {
	backend.calls = append(backend.calls, call)
}

func (backend *fakeNasBackend) recordRequest(call string, r *http.Request) {
	backend.record(call)
	backend.requestPath = r.URL.Path
}

func (backend *fakeNasBackend) GetNasDiskStatus(ctx context.Context) (*models.NasDiskStatusResponse, error) {
	backend.record("diskStatus")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasDiskStatusResponse{
		Result: &models.NasDiskStatusResponseResult{Disks: []*models.NasDiskInfo{{Name: "sda"}}},
	}, nil
}

func (backend *fakeNasBackend) GetNasServiceStatus(ctx context.Context) (*models.NasServiceResponse, error) {
	backend.record("serviceStatus")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasServiceResponse{
		Result: &models.NasServiceResponseResult{Sambas: []*models.NasServiceSambaInfo{{ShareName: "share"}}},
	}, nil
}

func (backend *fakeNasBackend) PostNasDiskInit(ctx context.Context, r *http.Request) (*models.NasDiskInitDiskResponse, error) {
	backend.recordRequest("diskInit", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasDiskInitDiskResponse{Result: &models.NasDiskInfo{Name: "sdb"}}, nil
}

func (backend *fakeNasBackend) PostNasDiskMountPoint(ctx context.Context, r *http.Request) (*models.NasDiskMountPointResponse, error) {
	backend.recordRequest("diskMountPoint", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasDiskMountPointResponse{
		Result: &models.NasDiskMountPointResponseResult{Mountpoint: "/mnt/sda1"},
	}, nil
}

func (backend *fakeNasBackend) PostNasDiskInitRest(ctx context.Context, r *http.Request) (*models.NasDiskInitDiskResponse, error) {
	backend.recordRequest("diskInitRest", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasDiskInitDiskResponse{Result: &models.NasDiskInfo{Name: "sdc"}}, nil
}

func (backend *fakeNasBackend) PostNasDiskPartFormat(ctx context.Context, r *http.Request) (*models.NasDiskPartitionFormatResponse, error) {
	backend.recordRequest("diskPartFormat", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasDiskPartitionFormatResponse{}, nil
}

func (backend *fakeNasBackend) PostNasSanboxFormat(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.recordRequest("sandboxFormat", r)
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeNasBackend) PostNasSanboxCommit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.recordRequest("sandboxCommit", r)
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeNasBackend) PostNasSanboxReset(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.recordRequest("sandboxReset", r)
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeNasBackend) PostNasSanboxExit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.recordRequest("sandboxExit", r)
	if backend.err != nil {
		return nil, backend.err
	}
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success}, nil
}

func (backend *fakeNasBackend) GetNasSanboxDisks(ctx context.Context) (*models.NasSandboxDisksResponse, error) {
	backend.record("sandboxDisks")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasSandboxDisksResponse{
		Result: &models.NasSandboxDisksResponseResult{Disks: []*models.NasDiskInfo{{Name: "usb0"}}},
	}, nil
}

func (backend *fakeNasBackend) GetNasSanboxStatus(ctx context.Context) (*models.NasSandboxStatusResponse, error) {
	backend.record("sandboxStatus")
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasSandboxStatusResponse{
		Result: &models.NasSandboxStatusResponseResult{Status: "running"},
	}, nil
}

func (backend *fakeNasBackend) PostNasDiskPartMount(ctx context.Context, r *http.Request) (*models.NasDiskPartitionMountResponse, error) {
	backend.recordRequest("diskPartMount", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasDiskPartitionMountResponse{}, nil
}

func (backend *fakeNasBackend) PostNasDiskLinkeaseEnable(ctx context.Context, r *http.Request) (*models.NasLinkeaseEnableResponse, error) {
	backend.recordRequest("linkeaseEnable", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.NasLinkeaseEnableResponse{
		Result: &models.NasLinkeaseEnableResponseResult{Port: "8897"},
	}, nil
}

func TestRegisterNasRoutesSuccessRoutes(t *testing.T) {
	tests := []struct {
		name            string
		method          string
		path            string
		wantCall        string
		wantRequestPath string
	}{
		{
			name:     "service status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/nas/service/status/",
			wantCall: "serviceStatus",
		},
		{
			name:     "service status user alias",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/u/nas/service/status/",
			wantCall: "serviceStatus",
		},
		{
			name:     "disk status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/nas/disk/status/",
			wantCall: "diskStatus",
		},
		{
			name:            "disk init",
			method:          http.MethodPost,
			path:            "/cgi-bin/luci/istore/nas/disk/init/",
			wantCall:        "diskInit",
			wantRequestPath: "/cgi-bin/luci/istore/nas/disk/init/",
		},
		{
			name:     "disk mountpoint",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/disk/mountpoint/",
			wantCall: "diskMountPoint",
		},
		{
			name:     "disk init rest",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/disk/initrest/",
			wantCall: "diskInitRest",
		},
		{
			name:     "disk partition format",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/disk/partition/format",
			wantCall: "diskPartFormat",
		},
		{
			name:     "disk partition mount",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/disk/partition/mount",
			wantCall: "diskPartMount",
		},
		{
			name:     "linkease enable",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/linkease/enable",
			wantCall: "linkeaseEnable",
		},
		{
			name:     "linkease enable user alias",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/u/nas/linkease/enable",
			wantCall: "linkeaseEnable",
		},
		{
			name:     "sandbox format",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/sandbox/",
			wantCall: "sandboxFormat",
		},
		{
			name:     "sandbox commit",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/sandbox/commit/",
			wantCall: "sandboxCommit",
		},
		{
			name:     "sandbox commit user alias",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/u/nas/sandbox/commit/",
			wantCall: "sandboxCommit",
		},
		{
			name:     "sandbox exit",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/sandbox/exit/",
			wantCall: "sandboxExit",
		},
		{
			name:     "sandbox reset",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/nas/sandbox/reset/",
			wantCall: "sandboxReset",
		},
		{
			name:     "sandbox disks",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/nas/sandbox/disks/",
			wantCall: "sandboxDisks",
		},
		{
			name:     "sandbox status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/nas/sandbox/",
			wantCall: "sandboxStatus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeNasBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestNasRoute(t, router, tt.method, tt.path, `{"path":"/dev/sda1"}`, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
			if tt.wantRequestPath != "" && backend.requestPath != tt.wantRequestPath {
				t.Fatalf("expected request path %q, got %q", tt.wantRequestPath, backend.requestPath)
			}
		})
	}
}

func TestRegisterRoutesDoesNotExposeLegacyNasShareQuickCreate(t *testing.T) {
	router := httprouter.New()
	RegisterRoutes(router, &fakeNasBackend{})

	cases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/cgi-bin/luci/istore/nas/samba/create"},
		{http.MethodPost, "/cgi-bin/luci/istore/nas/webdav/create"},
		{http.MethodGet, "/cgi-bin/luci/istore/nas/webdav/status/"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want 404", tc.method, tc.path, rec.Code)
		}
	}
}

func TestRegisterNasRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeNasBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestNasRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/nas/disk/status/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireNasEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterNasRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeNasBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestNasRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/nas/disk/status/", "", true)

	if len(backend.calls) != 1 || backend.calls[0] != "diskStatus" {
		t.Fatalf("expected diskStatus backend call, got %#v", backend.calls)
	}
	requireNasEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestNasRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func requireNasEnvelopeCode(t *testing.T, resp map[string]any, want int64) {
	t.Helper()

	got, ok := resp["success"].(float64)
	if !ok {
		t.Fatalf("expected success code in response, got %#v", resp)
	}
	if int64(got) != want {
		t.Fatalf("expected success code %d, got %v in %#v", want, got, resp)
	}
}

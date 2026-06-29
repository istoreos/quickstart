package guidestorage

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
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGuideStorageBackend struct {
	err error

	calls       []string
	requestPath string
	requestBody string
}

func (backend *fakeGuideStorageBackend) record(call string, r *http.Request) {
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

func (backend *fakeGuideStorageBackend) PostGuideAria2Init(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("aria2Init", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return guideStorageNormalResponse(), nil
}

func (backend *fakeGuideStorageBackend) PostGuideQbittorrentInit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("qbittorrentInit", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return guideStorageNormalResponse(), nil
}

func (backend *fakeGuideStorageBackend) PostGuideTransmissionInit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("transmissionInit", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return guideStorageNormalResponse(), nil
}

func (backend *fakeGuideStorageBackend) GetGuideDownloadServiceStatus(ctx context.Context) (*models.GuideDownloadServiceResponse, error) {
	backend.record("downloadServiceStatus", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDownloadServiceResponse{}, nil
}

func (backend *fakeGuideStorageBackend) GetGuideDownloadPartList(ctx context.Context) (*models.GuideDownloadPartitionListResponse, error) {
	backend.record("downloadPartList", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDownloadPartitionListResponse{}, nil
}

func (backend *fakeGuideStorageBackend) GetGuideDockerPartList(ctx context.Context) (*models.GuideDockerPartitionListResponse, error) {
	backend.record("dockerPartList", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDockerPartitionListResponse{}, nil
}

func (backend *fakeGuideStorageBackend) GetGuideDockerStatus(ctx context.Context) (*models.GuideDockerStatusResponse, error) {
	backend.record("dockerStatus", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDockerStatusResponse{}, nil
}

func (backend *fakeGuideStorageBackend) PostGuideDockerTransfer(ctx context.Context, r *http.Request) (*models.GuideDockerTransferResponse, error) {
	backend.record("dockerTransfer", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideDockerTransferResponse{
		Result: &models.GuideDockerTransferResponseResult{Path: "/mnt/data"},
	}, nil
}

func (backend *fakeGuideStorageBackend) PostGuideDockerSwitch(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("dockerSwitch", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return guideStorageNormalResponse(), nil
}

func (backend *fakeGuideStorageBackend) GetGuideSoftSource(ctx context.Context) (*models.GuideSoftSourceResponse, error) {
	backend.record("softSourceGet", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideSoftSourceResponse{}, nil
}

func (backend *fakeGuideStorageBackend) PostGuideSoftSource(ctx context.Context, r *http.Request) (*models.GuideSoftSourceResponse, error) {
	backend.record("softSourcePost", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideSoftSourceResponse{}, nil
}

func (backend *fakeGuideStorageBackend) GetGuideSoftSourceList(ctx context.Context) (*models.GuideSoftSourceListResponse, error) {
	backend.record("softSourceList", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GuideSoftSourceListResponse{}, nil
}

func (backend *fakeGuideStorageBackend) GetGlobalFolders(ctx context.Context) (*models.GlobalFoldersResponse, error) {
	backend.record("globalFoldersGet", nil)
	if backend.err != nil {
		return nil, backend.err
	}
	return &models.GlobalFoldersResponse{}, nil
}

func (backend *fakeGuideStorageBackend) PostGlobalFolders(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	backend.record("globalFoldersPost", r)
	if backend.err != nil {
		return nil, backend.err
	}
	return guideStorageNormalResponse(), nil
}

func TestRegisterGuideStorageRoutesMapsRoutesToBackendMethods(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		body     string
		wantCall string
	}{
		{
			name:     "aria2 init",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/aria2/init/",
			body:     `{"enabled":true}`,
			wantCall: "aria2Init",
		},
		{
			name:     "qbittorrent init",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/qbittorrent/init/",
			body:     `{"enabled":true}`,
			wantCall: "qbittorrentInit",
		},
		{
			name:     "transmission init",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/transmission/init/",
			body:     `{"enabled":true}`,
			wantCall: "transmissionInit",
		},
		{
			name:     "download service status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/download-service/status/",
			wantCall: "downloadServiceStatus",
		},
		{
			name:     "download partition list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/download/partition/list/",
			wantCall: "downloadPartList",
		},
		{
			name:     "docker partition list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/docker/partition/list/",
			wantCall: "dockerPartList",
		},
		{
			name:     "docker status",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/docker/status/",
			wantCall: "dockerStatus",
		},
		{
			name:     "docker transfer",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/docker/transfer/",
			body:     `{"path":"/mnt/data"}`,
			wantCall: "dockerTransfer",
		},
		{
			name:     "docker switch",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/docker/switch/",
			body:     `{"enable":true}`,
			wantCall: "dockerSwitch",
		},
		{
			name:     "soft source get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/soft-source/",
			wantCall: "softSourceGet",
		},
		{
			name:     "soft source post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/soft-source/",
			body:     `{"source":"default"}`,
			wantCall: "softSourcePost",
		},
		{
			name:     "soft source list",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/soft-source/list/",
			wantCall: "softSourceList",
		},
		{
			name:     "global folders get",
			method:   http.MethodGet,
			path:     "/cgi-bin/luci/istore/guide/global-folders/",
			wantCall: "globalFoldersGet",
		},
		{
			name:     "global folders post",
			method:   http.MethodPost,
			path:     "/cgi-bin/luci/istore/guide/global-folders/",
			body:     `{"download":"/mnt/download"}`,
			wantCall: "globalFoldersPost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeGuideStorageBackend{}
			router := httprouter.New()
			RegisterRoutes(router, backend)

			requestGuideStorageRoute(t, router, tt.method, tt.path, tt.body, true)

			if len(backend.calls) != 1 || backend.calls[0] != tt.wantCall {
				t.Fatalf("expected call %q, got %#v", tt.wantCall, backend.calls)
			}
		})
	}
}

func TestRegisterGuideStorageRoutesPostPassesOriginalRequestPath(t *testing.T) {
	backend := &fakeGuideStorageBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	body := `{"path":"/mnt/data"}`
	resp := requestGuideStorageRoute(t, router, http.MethodPost, "/cgi-bin/luci/istore/guide/docker/transfer/", body, true)

	if len(backend.calls) != 1 || backend.calls[0] != "dockerTransfer" {
		t.Fatalf("expected dockerTransfer backend call, got %#v", backend.calls)
	}
	if backend.requestPath != "/cgi-bin/luci/istore/guide/docker/transfer/" {
		t.Fatalf("expected original request path, got %q", backend.requestPath)
	}
	if backend.requestBody != body {
		t.Fatalf("expected original request body %q, got %q", body, backend.requestBody)
	}
	if _, ok := resp["result"]; !ok {
		t.Fatalf("expected result response, got %#v", resp)
	}
}

func TestRegisterGuideStorageRoutesRequiresForwardedSid(t *testing.T) {
	backend := &fakeGuideStorageBackend{}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestGuideStorageRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/guide/docker/status/", "", false)

	if len(backend.calls) != 0 {
		t.Fatalf("expected backend not to be called, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.ForbiddenError)
}

func TestRegisterGuideStorageRoutesBackendErrorReturnsErrorEnvelope(t *testing.T) {
	backend := &fakeGuideStorageBackend{err: errors.New("backend failed")}
	router := httprouter.New()
	RegisterRoutes(router, backend)

	resp := requestGuideStorageRoute(t, router, http.MethodGet, "/cgi-bin/luci/istore/guide/docker/status/", "", true)

	if len(backend.calls) != 1 || backend.calls[0] != "dockerStatus" {
		t.Fatalf("expected dockerStatus backend call, got %#v", backend.calls)
	}
	requireEnvelopeCode(t, resp, httpapi.GeneralError)
}

func requestGuideStorageRoute(t *testing.T, router *httprouter.Router, method, path, body string, withSID bool) map[string]any {
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

func guideStorageNormalResponse() *models.SDKNormalResponse {
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

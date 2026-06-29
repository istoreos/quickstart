package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeQuickstartConfigFacade struct {
	setReq        *models.QuickstartConfigRequest
	getReq        *models.QuickstartGetConfigRequest
	deleteReq     *models.QuickstartDeleteConfigRequest
	globalFolders *models.GlobalFolders

	setResp        *models.SDKNormalResponse
	getResp        *models.QuickstartConfigResponse
	deleteResp     *models.SDKNormalResponse
	globalGetResp  *models.GlobalFoldersResponse
	globalPostResp *models.SDKNormalResponse
}

func (facade *fakeQuickstartConfigFacade) Set(ctx context.Context, req models.QuickstartConfigRequest) (*models.SDKNormalResponse, error) {
	facade.setReq = &req
	return facade.setResp, nil
}

func (facade *fakeQuickstartConfigFacade) Get(ctx context.Context, req models.QuickstartGetConfigRequest) (*models.QuickstartConfigResponse, error) {
	facade.getReq = &req
	return facade.getResp, nil
}

func (facade *fakeQuickstartConfigFacade) Delete(ctx context.Context, req models.QuickstartDeleteConfigRequest) (*models.SDKNormalResponse, error) {
	facade.deleteReq = &req
	return facade.deleteResp, nil
}

func (facade *fakeQuickstartConfigFacade) GetGlobalFolders(ctx context.Context) (*models.GlobalFoldersResponse, error) {
	return facade.globalGetResp, nil
}

func (facade *fakeQuickstartConfigFacade) SetGlobalFolders(ctx context.Context, req models.GlobalFolders) (*models.SDKNormalResponse, error) {
	facade.globalFolders = &req
	return facade.globalPostResp, nil
}

func TestQuickstartConfigTypedFunctionsDelegateRequests(t *testing.T) {
	orig := newQuickstartConfigServiceFacade
	defer func() { newQuickstartConfigServiceFacade = orig }()

	success := models.ResponseSuccess(0)
	facade := &fakeQuickstartConfigFacade{
		setResp:    &models.SDKNormalResponse{Success: &success},
		getResp:    &models.QuickstartConfigResponse{Result: &models.QuickstartConfigResponseResult{Key: "dockerdir"}},
		deleteResp: &models.SDKNormalResponse{Success: &success},
	}
	newQuickstartConfigServiceFacade = func() quickstartConfigFacade { return facade }

	if _, err := QuickstartSetConfigValue(context.Background(), models.QuickstartConfigRequest{Key: "dockerdir", Type: "list", Values: []string{"/mnt/a"}}); err != nil {
		t.Fatalf("typed set: %v", err)
	}
	if facade.setReq == nil || facade.setReq.Key != "dockerdir" || facade.setReq.Type != "list" || len(facade.setReq.Values) != 1 {
		t.Fatalf("unexpected typed set request: %#v", facade.setReq)
	}

	if _, err := QuickstartGetConfigValue(context.Background(), models.QuickstartGetConfigRequest{Key: "dockerdir"}); err != nil {
		t.Fatalf("typed get: %v", err)
	}
	if facade.getReq == nil || facade.getReq.Key != "dockerdir" {
		t.Fatalf("unexpected typed get request: %#v", facade.getReq)
	}

	if _, err := QuickstartDeleteConfigValue(context.Background(), models.QuickstartDeleteConfigRequest{Key: "dockerdir"}); err != nil {
		t.Fatalf("typed delete: %v", err)
	}
	if facade.deleteReq == nil || facade.deleteReq.Key != "dockerdir" {
		t.Fatalf("unexpected typed delete request: %#v", facade.deleteReq)
	}
}

func TestQuickstartConfigWrappersDelegateParsedRequests(t *testing.T) {
	orig := newQuickstartConfigServiceFacade
	defer func() { newQuickstartConfigServiceFacade = orig }()

	success := models.ResponseSuccess(0)
	facade := &fakeQuickstartConfigFacade{
		setResp:    &models.SDKNormalResponse{Success: &success},
		getResp:    &models.QuickstartConfigResponse{Result: &models.QuickstartConfigResponseResult{Key: "dockerdir"}},
		deleteResp: &models.SDKNormalResponse{Success: &success},
	}
	newQuickstartConfigServiceFacade = func() quickstartConfigFacade { return facade }

	if _, err := QuickstartSetConfig(context.Background(), jsonRequest(http.MethodPost, "/quickstart/set", `{"key":"dockerdir","type":"list","values":["/mnt/a","/mnt/b"]}`)); err != nil {
		t.Fatalf("set wrapper: %v", err)
	}
	if facade.setReq == nil || facade.setReq.Key != "dockerdir" || facade.setReq.Type != "list" || len(facade.setReq.Values) != 2 {
		t.Fatalf("unexpected set request: %#v", facade.setReq)
	}

	if _, err := QuickstartGetConfig(context.Background(), jsonRequest(http.MethodPost, "/quickstart/get", `{"key":"dockerdir"}`)); err != nil {
		t.Fatalf("get wrapper: %v", err)
	}
	if facade.getReq == nil || facade.getReq.Key != "dockerdir" {
		t.Fatalf("unexpected get request: %#v", facade.getReq)
	}

	if _, err := QuickstartDeleteConfig(context.Background(), jsonRequest(http.MethodPost, "/quickstart/delete", `{"key":"dockerdir"}`)); err != nil {
		t.Fatalf("delete wrapper: %v", err)
	}
	if facade.deleteReq == nil || facade.deleteReq.Key != "dockerdir" {
		t.Fatalf("unexpected delete request: %#v", facade.deleteReq)
	}
}

func TestGlobalFoldersWrappersDelegateParsedRequests(t *testing.T) {
	orig := newQuickstartConfigServiceFacade
	defer func() { newQuickstartConfigServiceFacade = orig }()

	success := models.ResponseSuccess(0)
	facade := &fakeQuickstartConfigFacade{
		globalGetResp:  &models.GlobalFoldersResponse{Result: &models.GlobalFolders{Home: "/mnt/main"}},
		globalPostResp: &models.SDKNormalResponse{Success: &success},
	}
	newQuickstartConfigServiceFacade = func() quickstartConfigFacade { return facade }

	getResp, err := GlobalFoldersGetConfig(context.Background())
	if err != nil {
		t.Fatalf("global folders get wrapper: %v", err)
	}
	if getResp.Result == nil || getResp.Result.Home != "/mnt/main" {
		t.Fatalf("unexpected global folders get response: %#v", getResp)
	}

	if _, err := GlobalFoldersPostConfig(context.Background(), jsonRequest(http.MethodPost, "/global-folders", `{"home":"/mnt/main","configs":"/mnt/configs","public":"/mnt/public","downloads":"/mnt/downloads","caches":"/mnt/cache"}`)); err != nil {
		t.Fatalf("global folders post wrapper: %v", err)
	}
	if facade.globalFolders == nil || facade.globalFolders.Home != "/mnt/main" || facade.globalFolders.Caches != "/mnt/cache" {
		t.Fatalf("unexpected global folders request: %#v", facade.globalFolders)
	}
}

func jsonRequest(method string, path string, body string) *http.Request {
	return httptest.NewRequest(method, path, strings.NewReader(body))
}

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeAppServiceFacade struct {
	checkReq   *models.AppCheckRequest
	installReq *models.AppInstallRequest
	listCalled bool
}

func (facade *fakeAppServiceFacade) Check(ctx context.Context, req models.AppCheckRequest) (*models.AppCheckResponse, error) {
	facade.checkReq = &req
	return &models.AppCheckResponse{Result: &models.AppCheckResponseResult{Name: req.Name, Status: "running"}}, nil
}

func (facade *fakeAppServiceFacade) Install(ctx context.Context, req models.AppInstallRequest) (*models.SDKNormalResponse, error) {
	facade.installReq = &req
	success := models.ResponseSuccess(0)
	return &models.SDKNormalResponse{Success: &success, Detail: "installing"}, nil
}

func (facade *fakeAppServiceFacade) InstalledList(ctx context.Context) (models.AppInstalledListResponse, error) {
	facade.listCalled = true
	return models.AppInstalledListResponse{{Name: "demo"}}, nil
}

func TestAppWrappersDelegateParsedRequests(t *testing.T) {
	orig := newAppServiceFacade
	defer func() { newAppServiceFacade = orig }()

	facade := &fakeAppServiceFacade{}
	newAppServiceFacade = func() appServiceFacade { return facade }

	if _, err := AppCheck(context.Background(), jsonRequest(http.MethodPost, "/app/check", `{"name":"demo","checkRunning":true}`)); err != nil {
		t.Fatalf("AppCheck: %v", err)
	}
	if facade.checkReq == nil || facade.checkReq.Name != "demo" || !facade.checkReq.CheckRunning {
		t.Fatalf("unexpected check request: %#v", facade.checkReq)
	}

	if _, err := AppInstall(context.Background(), jsonRequest(http.MethodPost, "/app/install", `{"name":"demo"}`)); err != nil {
		t.Fatalf("AppInstall: %v", err)
	}
	if facade.installReq == nil || facade.installReq.Name != "demo" {
		t.Fatalf("unexpected install request: %#v", facade.installReq)
	}

	resp, err := AppInstalledList(context.Background(), jsonRequest(http.MethodGet, "/app/install-list", ``))
	if err != nil {
		t.Fatalf("AppInstalledList: %v", err)
	}
	if !facade.listCalled || len(resp) != 1 || resp[0].Name != "demo" {
		t.Fatalf("unexpected list call/response: called=%v resp=%#v", facade.listCalled, resp)
	}
}

func TestAppTypedFunctionsDelegateRequests(t *testing.T) {
	orig := newAppServiceFacade
	defer func() { newAppServiceFacade = orig }()

	facade := &fakeAppServiceFacade{}
	newAppServiceFacade = func() appServiceFacade { return facade }

	if _, err := AppCheckValue(context.Background(), models.AppCheckRequest{Name: "demo", CheckRunning: true}); err != nil {
		t.Fatalf("AppCheckValue: %v", err)
	}
	if facade.checkReq == nil || facade.checkReq.Name != "demo" || !facade.checkReq.CheckRunning {
		t.Fatalf("unexpected typed check request: %#v", facade.checkReq)
	}

	if _, err := AppInstallValue(context.Background(), models.AppInstallRequest{Name: "demo"}); err != nil {
		t.Fatalf("AppInstallValue: %v", err)
	}
	if facade.installReq == nil || facade.installReq.Name != "demo" {
		t.Fatalf("unexpected typed install request: %#v", facade.installReq)
	}

	resp, err := AppInstalledListValue(context.Background())
	if err != nil {
		t.Fatalf("AppInstalledListValue: %v", err)
	}
	if !facade.listCalled || len(resp) != 1 || resp[0].Name != "demo" {
		t.Fatalf("unexpected typed list call/response: called=%v resp=%#v", facade.listCalled, resp)
	}
}

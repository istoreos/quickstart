package service

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSystemUpdateFacade struct {
	checkResult *models.SystemCheckUpdateResponseResult
	checkErr    error
	autoResp    *models.SDKNormalResponse
	autoErr     error
	autoReq     models.SystemAutoCheckUpdateRequest
	autoCalled  bool
}

func (svc *fakeSystemUpdateFacade) Check(ctx context.Context) (*models.SystemCheckUpdateResponseResult, error) {
	return svc.checkResult, svc.checkErr
}

func (svc *fakeSystemUpdateFacade) SetAutoCheck(ctx context.Context, req models.SystemAutoCheckUpdateRequest) (*models.SDKNormalResponse, error) {
	svc.autoCalled = true
	svc.autoReq = req
	return svc.autoResp, svc.autoErr
}

func TestSystemCheckUpdateDelegatesToFacade(t *testing.T) {
	original := newSystemUpdateService
	defer func() { newSystemUpdateService = original }()

	newSystemUpdateService = func() systemUpdateFacade {
		return &fakeSystemUpdateFacade{
			checkResult: &models.SystemCheckUpdateResponseResult{
				NeedUpdate: true,
				Msg:        "new firmware",
			},
		}
	}

	resp, err := SystemCheckUpdate(context.Background())
	if err != nil {
		t.Fatalf("SystemCheckUpdate returned error: %v", err)
	}
	if resp.Result == nil || !resp.Result.NeedUpdate || resp.Result.Msg != "new firmware" {
		t.Fatalf("Result = %#v", resp.Result)
	}
}

func TestSystemCheckUpdatePropagatesFacadeError(t *testing.T) {
	original := newSystemUpdateService
	defer func() { newSystemUpdateService = original }()

	expectedErr := errors.New("check failed")
	newSystemUpdateService = func() systemUpdateFacade {
		return &fakeSystemUpdateFacade{checkErr: expectedErr}
	}

	if _, err := SystemCheckUpdate(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemCheckUpdate error = %v, want expectedErr", err)
	}
}

func TestSystemAutoCheckUpdateDelegatesToFacade(t *testing.T) {
	original := newSystemUpdateService
	defer func() { newSystemUpdateService = original }()

	success := models.ResponseSuccess(0)
	facade := &fakeSystemUpdateFacade{
		autoResp: &models.SDKNormalResponse{Success: &success},
	}
	newSystemUpdateService = func() systemUpdateFacade {
		return facade
	}

	req := httptest.NewRequest("POST", "/update/auto", bytes.NewBufferString(`{"enable":true}`))
	resp, err := SystemAutoCheckUpdate(context.Background(), req)
	if err != nil {
		t.Fatalf("SystemAutoCheckUpdate returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want 0", resp.Success)
	}
	if !facade.autoCalled {
		t.Fatal("SetAutoCheck was not called")
	}
	if !reflect.DeepEqual(facade.autoReq, models.SystemAutoCheckUpdateRequest{Enable: true}) {
		t.Fatalf("autoReq = %#v", facade.autoReq)
	}
}

func TestSystemAutoCheckUpdatePropagatesFacadeError(t *testing.T) {
	original := newSystemUpdateService
	defer func() { newSystemUpdateService = original }()

	expectedErr := errors.New("auto failed")
	newSystemUpdateService = func() systemUpdateFacade {
		return &fakeSystemUpdateFacade{autoErr: expectedErr}
	}

	req := httptest.NewRequest("POST", "/update/auto", bytes.NewBufferString(`{"enable":false}`))
	if _, err := SystemAutoCheckUpdate(context.Background(), req); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemAutoCheckUpdate error = %v, want expectedErr", err)
	}
}

func TestSystemAutoCheckUpdateReturnsParseError(t *testing.T) {
	original := newSystemUpdateService
	defer func() { newSystemUpdateService = original }()

	newSystemUpdateService = func() systemUpdateFacade {
		return &fakeSystemUpdateFacade{}
	}

	req := httptest.NewRequest("POST", "/update/auto", bytes.NewBufferString(`{`))
	if _, err := SystemAutoCheckUpdate(context.Background(), req); err == nil || err.Error() != "请求解析失败" {
		t.Fatalf("SystemAutoCheckUpdate error = %v, want 请求解析失败", err)
	}
}

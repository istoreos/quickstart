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

type fakeSystemModuleSettingsFacade struct {
	result    *models.SystemModuleSettingsResponseResult
	err       error
	setResp   *models.SDKNormalResponse
	setErr    error
	setReq    models.SystemModuleSettingsRequest
	setCalled bool
}

func (svc *fakeSystemModuleSettingsFacade) Get(ctx context.Context) (*models.SystemModuleSettingsResponseResult, error) {
	return svc.result, svc.err
}

func (svc *fakeSystemModuleSettingsFacade) Set(ctx context.Context, req models.SystemModuleSettingsRequest) (*models.SDKNormalResponse, error) {
	svc.setCalled = true
	svc.setReq = req
	return svc.setResp, svc.setErr
}

func TestSystemModuleSettingsGetDelegatesToFacade(t *testing.T) {
	original := newSystemModuleSettingsService
	defer func() { newSystemModuleSettingsService = original }()

	newSystemModuleSettingsService = func() systemModuleSettingsFacade {
		return &fakeSystemModuleSettingsFacade{
			result: &models.SystemModuleSettingsResponseResult{DiableDisplay: []string{"smart", "raid"}},
		}
	}

	resp, err := SystemModuleSettingsGet(context.Background())
	if err != nil {
		t.Fatalf("SystemModuleSettingsGet returned error: %v", err)
	}
	if !reflect.DeepEqual(resp.Result.DiableDisplay, []string{"smart", "raid"}) {
		t.Fatalf("DiableDisplay = %#v", resp.Result.DiableDisplay)
	}
}

func TestSystemModuleSettingsGetPropagatesFacadeError(t *testing.T) {
	original := newSystemModuleSettingsService
	defer func() { newSystemModuleSettingsService = original }()

	expectedErr := errors.New("facade failed")
	newSystemModuleSettingsService = func() systemModuleSettingsFacade {
		return &fakeSystemModuleSettingsFacade{err: expectedErr}
	}

	if _, err := SystemModuleSettingsGet(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemModuleSettingsGet error = %v, want expectedErr", err)
	}
}

func TestSystemModuleSettingsPostDelegatesToFacade(t *testing.T) {
	original := newSystemModuleSettingsService
	defer func() { newSystemModuleSettingsService = original }()

	success := models.ResponseSuccess(0)
	facade := &fakeSystemModuleSettingsFacade{
		setResp: &models.SDKNormalResponse{Success: &success},
	}
	newSystemModuleSettingsService = func() systemModuleSettingsFacade {
		return facade
	}

	req := httptest.NewRequest("POST", "/settings", bytes.NewBufferString(`{"diableDisplay":["smart","raid"]}`))
	resp, err := SystemModuleSettingsPost(context.Background(), req)
	if err != nil {
		t.Fatalf("SystemModuleSettingsPost returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want code 0", resp.Success)
	}
	if !facade.setCalled {
		t.Fatal("Set was not called")
	}
	if !reflect.DeepEqual(facade.setReq.DiableDisplay, []string{"smart", "raid"}) {
		t.Fatalf("DiableDisplay = %#v", facade.setReq.DiableDisplay)
	}
}

func TestSystemModuleSettingsPostPropagatesFacadeError(t *testing.T) {
	original := newSystemModuleSettingsService
	defer func() { newSystemModuleSettingsService = original }()

	expectedErr := errors.New("set failed")
	newSystemModuleSettingsService = func() systemModuleSettingsFacade {
		return &fakeSystemModuleSettingsFacade{setErr: expectedErr}
	}

	req := httptest.NewRequest("POST", "/settings", bytes.NewBufferString(`{"diableDisplay":["smart"]}`))
	if _, err := SystemModuleSettingsPost(context.Background(), req); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemModuleSettingsPost error = %v, want expectedErr", err)
	}
}

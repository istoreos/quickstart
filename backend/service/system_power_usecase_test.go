package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSystemPowerFacade struct {
	rebootResp   *models.SDKNormalResponse
	rebootErr    error
	rebootCalled bool
	powerResp    *models.SDKNormalResponse
	powerErr     error
	powerCalled  bool
}

func (svc *fakeSystemPowerFacade) Reboot(ctx context.Context) (*models.SDKNormalResponse, error) {
	svc.rebootCalled = true
	return svc.rebootResp, svc.rebootErr
}

func (svc *fakeSystemPowerFacade) PowerOff(ctx context.Context) (*models.SDKNormalResponse, error) {
	svc.powerCalled = true
	return svc.powerResp, svc.powerErr
}

func TestSystemRebootDelegatesToFacade(t *testing.T) {
	original := newSystemPowerService
	defer func() { newSystemPowerService = original }()

	success := models.ResponseSuccess(0)
	facade := &fakeSystemPowerFacade{
		rebootResp: &models.SDKNormalResponse{Success: &success},
	}
	newSystemPowerService = func() systemPowerFacade {
		return facade
	}

	resp, err := SystemReboot(context.Background())
	if err != nil {
		t.Fatalf("SystemReboot returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want 0", resp.Success)
	}
	if !facade.rebootCalled {
		t.Fatal("Reboot was not called")
	}
}

func TestSystemRebootPropagatesFacadeError(t *testing.T) {
	original := newSystemPowerService
	defer func() { newSystemPowerService = original }()

	expectedErr := errors.New("重启失败permission denied")
	newSystemPowerService = func() systemPowerFacade {
		return &fakeSystemPowerFacade{rebootErr: expectedErr}
	}

	if _, err := SystemReboot(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemReboot error = %v, want expectedErr", err)
	}
}

func TestSystemPowerOffDelegatesToFacade(t *testing.T) {
	original := newSystemPowerService
	defer func() { newSystemPowerService = original }()

	success := models.ResponseSuccess(0)
	facade := &fakeSystemPowerFacade{
		powerResp: &models.SDKNormalResponse{Success: &success},
	}
	newSystemPowerService = func() systemPowerFacade {
		return facade
	}

	resp, err := SystemPowerOff(context.Background())
	if err != nil {
		t.Fatalf("SystemPowerOff returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want 0", resp.Success)
	}
	if !facade.powerCalled {
		t.Fatal("PowerOff was not called")
	}
}

func TestSystemPowerOffPropagatesFacadeError(t *testing.T) {
	original := newSystemPowerService
	defer func() { newSystemPowerService = original }()

	expectedErr := errors.New("关机失败permission denied")
	newSystemPowerService = func() systemPowerFacade {
		return &fakeSystemPowerFacade{powerErr: expectedErr}
	}

	if _, err := SystemPowerOff(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemPowerOff error = %v, want expectedErr", err)
	}
}

package service

import (
	"bytes"
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSystemPasswordFacade struct {
	resp   *models.SDKNormalResponse
	err    error
	req    models.NasSystemSetPasswordRequest
	called bool
}

func (svc *fakeSystemPasswordFacade) SetRootPassword(ctx context.Context, req models.NasSystemSetPasswordRequest) (*models.SDKNormalResponse, error) {
	svc.called = true
	svc.req = req
	return svc.resp, svc.err
}

func TestSystemSetPasswordDelegatesToFacade(t *testing.T) {
	original := newSystemPasswordService
	defer func() { newSystemPasswordService = original }()

	success := models.ResponseSuccess(0)
	facade := &fakeSystemPasswordFacade{
		resp: &models.SDKNormalResponse{Success: &success},
	}
	newSystemPasswordService = func() systemPasswordFacade {
		return facade
	}

	req := httptest.NewRequest("POST", "/setPassword", bytes.NewBufferString(`{"password":"secret"}`))
	resp, err := SystemSetPassword(context.Background(), req)
	if err != nil {
		t.Fatalf("SystemSetPassword returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want 0", resp.Success)
	}
	if !facade.called {
		t.Fatal("SetRootPassword was not called")
	}
	if facade.req.Password != "secret" {
		t.Fatalf("Password = %q", facade.req.Password)
	}
}

func TestSystemSetPasswordPropagatesFacadeError(t *testing.T) {
	original := newSystemPasswordService
	defer func() { newSystemPasswordService = original }()

	expectedErr := errors.New("设置密码错误")
	newSystemPasswordService = func() systemPasswordFacade {
		return &fakeSystemPasswordFacade{err: expectedErr}
	}

	req := httptest.NewRequest("POST", "/setPassword", bytes.NewBufferString(`{"password":"secret"}`))
	if _, err := SystemSetPassword(context.Background(), req); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemSetPassword error = %v, want expectedErr", err)
	}
}

func TestSystemSetPasswordReturnsParseError(t *testing.T) {
	original := newSystemPasswordService
	defer func() { newSystemPasswordService = original }()

	newSystemPasswordService = func() systemPasswordFacade {
		return &fakeSystemPasswordFacade{}
	}

	req := httptest.NewRequest("POST", "/setPassword", bytes.NewBufferString(`{`))
	if _, err := SystemSetPassword(context.Background(), req); err == nil || err.Error() != "请求解析失败" {
		t.Fatalf("SystemSetPassword error = %v, want 请求解析失败", err)
	}
}

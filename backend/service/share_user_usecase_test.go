package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
	shareuser "github.com/linkease/quick-start/istore-backend/modules/share/user"
)

func shareTestRequest(body string) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	return req
}

type fakeShareUserFacade struct {
	listResult  []*models.ShareUserInfo
	listErr     error
	createInput shareuser.CreateInput
	createErr   error
	updateInput shareuser.UpdateInput
	updateErr   error
	deleteInput shareuser.DeleteInput
	deleteErr   error
}

func (svc *fakeShareUserFacade) List(ctx context.Context) ([]*models.ShareUserInfo, error) {
	return svc.listResult, svc.listErr
}

func (svc *fakeShareUserFacade) Create(ctx context.Context, input shareuser.CreateInput) error {
	svc.createInput = input
	return svc.createErr
}

func (svc *fakeShareUserFacade) Update(ctx context.Context, input shareuser.UpdateInput) error {
	svc.updateInput = input
	return svc.updateErr
}

func (svc *fakeShareUserFacade) Delete(ctx context.Context, input shareuser.DeleteInput) error {
	svc.deleteInput = input
	return svc.deleteErr
}

func TestShareUserCompatibilityDelegatesListCreateUpdateDelete(t *testing.T) {
	original := newShareUserService
	defer func() { newShareUserService = original }()

	facade := &fakeShareUserFacade{
		listResult: []*models.ShareUserInfo{{UserName: "alice", Password: "pw"}},
	}
	newShareUserService = func() shareUserFacade {
		return facade
	}

	listResp, err := ShareUserList(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if listResp == nil || listResp.Result == nil || len(listResp.Result.Users) != 1 || listResp.Result.Users[0].UserName != "alice" {
		t.Fatalf("unexpected list response: %#v", listResp)
	}

	if _, err := ShareUserCreate(context.Background(), shareTestRequest(`{"userName":"bob","password":"pw"}`)); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if facade.createInput.UserName != "bob" || facade.createInput.Password != "pw" {
		t.Fatalf("unexpected create input: %#v", facade.createInput)
	}

	if _, err := ShareUserUpdate(context.Background(), shareTestRequest(`{"userName":"bob","password":"new"}`)); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if facade.updateInput.UserName != "bob" || facade.updateInput.Password != "new" {
		t.Fatalf("unexpected update input: %#v", facade.updateInput)
	}

	if _, err := ShareUserDelete(context.Background(), shareTestRequest(`{"userName":"bob"}`)); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if facade.deleteInput.UserName != "bob" {
		t.Fatalf("unexpected delete input: %#v", facade.deleteInput)
	}
}

func TestShareUserCompatibilityPropagatesFacadeErrors(t *testing.T) {
	original := newShareUserService
	defer func() { newShareUserService = original }()

	expectedErr := errors.New("share user failed")
	newShareUserService = func() shareUserFacade {
		return &fakeShareUserFacade{
			listErr:   expectedErr,
			createErr: expectedErr,
			updateErr: expectedErr,
			deleteErr: expectedErr,
		}
	}

	if _, err := ShareUserList(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected list error, got %v", err)
	}
	if _, err := ShareUserCreate(context.Background(), shareTestRequest(`{"userName":"bob","password":"pw"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("expected create error, got %v", err)
	}
	if _, err := ShareUserUpdate(context.Background(), shareTestRequest(`{"userName":"bob","password":"pw"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("expected update error, got %v", err)
	}
	if _, err := ShareUserDelete(context.Background(), shareTestRequest(`{"userName":"bob"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("expected delete error, got %v", err)
	}
}

func TestShareUserCompatibilityKeepsDecodeErrors(t *testing.T) {
	original := newShareUserService
	defer func() { newShareUserService = original }()

	newShareUserService = func() shareUserFacade {
		return &fakeShareUserFacade{}
	}
	if _, err := ShareUserCreate(context.Background(), shareTestRequest(`{`)); err == nil {
		t.Fatalf("expected create decode error")
	}
	if _, err := ShareUserUpdate(context.Background(), shareTestRequest(`{`)); err == nil {
		t.Fatalf("expected update decode error")
	}
	if _, err := ShareUserDelete(context.Background(), shareTestRequest(`{`)); err == nil {
		t.Fatalf("expected delete decode error")
	}
}

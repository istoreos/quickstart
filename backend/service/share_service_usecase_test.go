package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
	shareservice "github.com/linkease/quick-start/istore-backend/modules/share/service"
)

type fakeShareServiceFacade struct {
	listResult []*models.ShareServiceInfo
	err        error

	createInput shareservice.CreateInput
	updateInput shareservice.UpdateInput
	deleteInput shareservice.DeleteInput
}

func (svc *fakeShareServiceFacade) List(ctx context.Context) ([]*models.ShareServiceInfo, error) {
	return svc.listResult, svc.err
}

func (svc *fakeShareServiceFacade) Create(ctx context.Context, input shareservice.CreateInput) error {
	svc.createInput = input
	return svc.err
}

func (svc *fakeShareServiceFacade) Update(ctx context.Context, input shareservice.UpdateInput) error {
	svc.updateInput = input
	return svc.err
}

func (svc *fakeShareServiceFacade) Delete(ctx context.Context, input shareservice.DeleteInput) error {
	svc.deleteInput = input
	return svc.err
}

func TestShareServiceCompatibilityDelegatesListCreateUpdateDelete(t *testing.T) {
	original := newShareService
	defer func() { newShareService = original }()

	facade := &fakeShareServiceFacade{
		listResult: []*models.ShareServiceInfo{{Name: "media", Path: "/mnt/media"}},
	}
	newShareService = func() shareServiceFacade {
		return facade
	}

	listResp, err := ShareServiceList(context.Background())
	if err != nil {
		t.Fatalf("ShareServiceList returned error: %v", err)
	}
	if len(listResp.Result.Services) != 1 || listResp.Result.Services[0].Name != "media" {
		t.Fatalf("ShareServiceList response = %#v", listResp)
	}

	body := `{"name":"docs","path":"/mnt/docs","samba":true,"webdav":true,"users":[{"userName":"alice","rw":true},{"userName":"bob","ro":true}]}`
	if _, err := ShareServiceCreate(context.Background(), shareTestRequest(body)); err != nil {
		t.Fatalf("ShareServiceCreate returned error: %v", err)
	}
	if facade.createInput.Name != "docs" || facade.createInput.Path != "/mnt/docs" || !facade.createInput.Samba || !facade.createInput.Webdav {
		t.Fatalf("create input = %#v", facade.createInput)
	}
	if len(facade.createInput.Users) != 2 || !facade.createInput.Users[0].Rw || !facade.createInput.Users[1].Ro {
		t.Fatalf("create users = %#v", facade.createInput.Users)
	}

	if _, err := ShareServiceUpdate(context.Background(), shareTestRequest(body)); err != nil {
		t.Fatalf("ShareServiceUpdate returned error: %v", err)
	}
	if facade.updateInput.Name != "docs" || facade.updateInput.Path != "/mnt/docs" || !facade.updateInput.Samba || !facade.updateInput.Webdav {
		t.Fatalf("update input = %#v", facade.updateInput)
	}

	if _, err := ShareServiceDelete(context.Background(), shareTestRequest(`{"name":"docs"}`)); err != nil {
		t.Fatalf("ShareServiceDelete returned error: %v", err)
	}
	if facade.deleteInput.Name != "docs" {
		t.Fatalf("delete input = %#v", facade.deleteInput)
	}
}

func TestShareServiceCompatibilityPropagatesFacadeErrors(t *testing.T) {
	original := newShareService
	defer func() { newShareService = original }()

	expectedErr := errors.New("facade failed")
	newShareService = func() shareServiceFacade {
		return &fakeShareServiceFacade{err: expectedErr}
	}

	if _, err := ShareServiceList(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("ShareServiceList error = %v, want expectedErr", err)
	}
	if _, err := ShareServiceCreate(context.Background(), shareTestRequest(`{"name":"docs","path":"/mnt/docs"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("ShareServiceCreate error = %v, want expectedErr", err)
	}
	if _, err := ShareServiceUpdate(context.Background(), shareTestRequest(`{"name":"docs","path":"/mnt/docs"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("ShareServiceUpdate error = %v, want expectedErr", err)
	}
	if _, err := ShareServiceDelete(context.Background(), shareTestRequest(`{"name":"docs"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("ShareServiceDelete error = %v, want expectedErr", err)
	}
}

func TestShareServiceCompatibilityKeepsDecodeErrors(t *testing.T) {
	original := newShareService
	defer func() { newShareService = original }()

	newShareService = func() shareServiceFacade {
		return &fakeShareServiceFacade{}
	}
	if _, err := ShareServiceCreate(context.Background(), shareTestRequest(`{`)); err == nil {
		t.Fatal("ShareServiceCreate expected decode error")
	}
	if _, err := ShareServiceUpdate(context.Background(), shareTestRequest(`{`)); err == nil {
		t.Fatal("ShareServiceUpdate expected decode error")
	}
	if _, err := ShareServiceDelete(context.Background(), shareTestRequest(`{`)); err == nil {
		t.Fatal("ShareServiceDelete expected decode error")
	}
}

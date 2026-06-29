package service

import (
	"context"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSmartInventoryFacade struct {
	called bool
	resp   *models.SmartListResponse
}

func (facade *fakeSmartInventoryFacade) List(ctx context.Context) (*models.SmartListResponse, error) {
	facade.called = true
	return facade.resp, nil
}

func TestSmartGetListDelegatesToInventoryFacade(t *testing.T) {
	orig := newSmartInventoryServiceFacade
	defer func() { newSmartInventoryServiceFacade = orig }()

	facade := &fakeSmartInventoryFacade{
		resp: &models.SmartListResponse{Result: &models.SmartListResponseResult{
			Disks: []*models.SmartInfo{{Name: "sda", Path: "/dev/sda"}},
		}},
	}
	newSmartInventoryServiceFacade = func() smartInventoryFacade { return facade }

	resp, err := SmartGetList(context.Background())
	if err != nil {
		t.Fatalf("SmartGetList: %v", err)
	}
	if !facade.called {
		t.Fatal("expected inventory facade to be called")
	}
	if resp.Result == nil || len(resp.Result.Disks) != 1 || resp.Result.Disks[0].Name != "sda" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

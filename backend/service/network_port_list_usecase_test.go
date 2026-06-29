package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var networkPortListFacadeTestMu sync.Mutex

type fakeNetworkPortListFacade struct {
	resp  *models.NetworkPortListResponse
	err   error
	calls int
}

func (facade *fakeNetworkPortListFacade) GetPortList(ctx context.Context) (*models.NetworkPortListResponse, error) {
	facade.calls++
	return facade.resp, facade.err
}

func TestNetworkPortListCompatibilityDelegatesToService(t *testing.T) {
	networkPortListFacadeTestMu.Lock()
	defer networkPortListFacadeTestMu.Unlock()

	facade := &fakeNetworkPortListFacade{
		resp: &models.NetworkPortListResponse{
			Result: &models.NetworkPortListResponseResult{
				Ports: []*models.NetworkPortInfo{{Name: "eth0"}},
			},
		},
	}
	original := newNetworkPortListService
	newNetworkPortListService = func() networkPortListFacade { return facade }
	defer func() { newNetworkPortListService = original }()

	resp, err := NetworkPortList(context.Background())
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if facade.calls != 1 {
		t.Fatalf("expected facade to be called once, got %d", facade.calls)
	}
	if resp == nil || resp.Result == nil || len(resp.Result.Ports) != 1 || resp.Result.Ports[0].Name != "eth0" {
		t.Fatalf("unexpected delegated response: %#v", resp)
	}
}

func TestNetworkPortListCompatibilityPropagatesServiceError(t *testing.T) {
	networkPortListFacadeTestMu.Lock()
	defer networkPortListFacadeTestMu.Unlock()

	delegateErr := errors.New("delegate failed")
	facade := &fakeNetworkPortListFacade{err: delegateErr}
	original := newNetworkPortListService
	newNetworkPortListService = func() networkPortListFacade { return facade }
	defer func() { newNetworkPortListService = original }()

	if _, err := NetworkPortList(context.Background()); !errors.Is(err, delegateErr) {
		t.Fatalf("expected delegated error, got %v", err)
	}
	if facade.calls != 1 {
		t.Fatalf("expected facade to be called once, got %d", facade.calls)
	}
}

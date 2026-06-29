package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var homeBoxEnableFacadeTestMu sync.Mutex

type fakeHomeBoxEnableFacade struct {
	resp  *models.NetworkHomeBoxEnableResponse
	err   error
	calls int
}

func (facade *fakeHomeBoxEnableFacade) Enable(ctx context.Context) (*models.NetworkHomeBoxEnableResponse, error) {
	facade.calls++
	return facade.resp, facade.err
}

func TestNetworkHomeBoxEnableCompatibilityDelegatesToService(t *testing.T) {
	homeBoxEnableFacadeTestMu.Lock()
	defer homeBoxEnableFacadeTestMu.Unlock()

	facade := &fakeHomeBoxEnableFacade{
		resp: &models.NetworkHomeBoxEnableResponse{
			Result: &models.NetworkHomeBoxEnableResponseResult{Port: "3300"},
		},
	}
	original := newHomeBoxEnableService
	newHomeBoxEnableService = func() homeBoxEnableFacade { return facade }
	defer func() { newHomeBoxEnableService = original }()

	resp, err := NetworkHomeBoxEnable(context.Background())
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if facade.calls != 1 {
		t.Fatalf("expected facade to be called once, got %d", facade.calls)
	}
	if resp == nil || resp.Result == nil || resp.Result.Port != "3300" {
		t.Fatalf("unexpected delegated response: %#v", resp)
	}
}

func TestNetworkHomeBoxEnableCompatibilityPropagatesServiceError(t *testing.T) {
	homeBoxEnableFacadeTestMu.Lock()
	defer homeBoxEnableFacadeTestMu.Unlock()

	delegateErr := errors.New("delegate failed")
	facade := &fakeHomeBoxEnableFacade{err: delegateErr}
	original := newHomeBoxEnableService
	newHomeBoxEnableService = func() homeBoxEnableFacade { return facade }
	defer func() { newHomeBoxEnableService = original }()

	if _, err := NetworkHomeBoxEnable(context.Background()); !errors.Is(err, delegateErr) {
		t.Fatalf("expected delegated error, got %v", err)
	}
	if facade.calls != 1 {
		t.Fatalf("expected facade to be called once, got %d", facade.calls)
	}
}

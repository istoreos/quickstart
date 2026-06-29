package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var networkStatusUsecaseTestMu sync.Mutex

type fakeNetworkStatusFacade struct {
	resp            *models.NetworkStatusResponse
	err             error
	lastSetupFinish bool
	called          int
}

func (svc *fakeNetworkStatusFacade) GetNetworkStatus(ctx context.Context, setupFinish bool) (*models.NetworkStatusResponse, error) {
	svc.called++
	svc.lastSetupFinish = setupFinish
	return svc.resp, svc.err
}

func TestNetworkStatusCompatibilityDelegatesToService(t *testing.T) {
	networkStatusUsecaseTestMu.Lock()
	defer networkStatusUsecaseTestMu.Unlock()

	oldFactory := newNetworkStatusService
	defer func() {
		newNetworkStatusService = oldFactory
	}()

	fakeSvc := &fakeNetworkStatusFacade{
		resp: &models.NetworkStatusResponse{
			Result: &models.NetworkStatusResponseResult{
				DefaultInterface: "wan",
				DNSProto:         "auto",
			},
		},
	}
	newNetworkStatusService = func(netChecker *NetworkOnlineChecker) networkStatusFacade {
		if netChecker == nil {
			t.Fatal("expected wrapper to forward netChecker")
		}
		return fakeSvc
	}

	resp, err := NetworkStatus(context.Background(), &NetworkOnlineChecker{}, true)
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if fakeSvc.called != 1 {
		t.Fatalf("expected facade called once, got %d", fakeSvc.called)
	}
	if !fakeSvc.lastSetupFinish {
		t.Fatal("expected setupFinish forwarded to facade")
	}
	if resp != fakeSvc.resp {
		t.Fatalf("expected wrapper to return service response pointer")
	}
}

func TestNetworkStatusCompatibilityPropagatesServiceError(t *testing.T) {
	networkStatusUsecaseTestMu.Lock()
	defer networkStatusUsecaseTestMu.Unlock()

	oldFactory := newNetworkStatusService
	defer func() {
		newNetworkStatusService = oldFactory
	}()

	fakeSvc := &fakeNetworkStatusFacade{err: errors.New("network status failed")}
	newNetworkStatusService = func(netChecker *NetworkOnlineChecker) networkStatusFacade {
		return fakeSvc
	}

	_, err := NetworkStatus(context.Background(), &NetworkOnlineChecker{}, false)
	if !errors.Is(err, fakeSvc.err) {
		t.Fatalf("expected service error propagated, got %v", err)
	}
	if fakeSvc.called != 1 {
		t.Fatalf("expected facade called once, got %d", fakeSvc.called)
	}
	if fakeSvc.lastSetupFinish {
		t.Fatal("expected setupFinish=false forwarded")
	}
}

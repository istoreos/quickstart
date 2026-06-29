package service

import (
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var networkPublicAddressFacadeTestMu sync.Mutex

type fakeNetworkPublicAddressFacade struct {
	resp      *models.NetworkCheckPublicNetResponse
	err       error
	ipVersion string
}

func (facade *fakeNetworkPublicAddressFacade) CheckPublicAddress(ipVersion string) (*models.NetworkCheckPublicNetResponse, error) {
	facade.ipVersion = ipVersion
	return facade.resp, facade.err
}

func TestNetworkCheckPublicNetCompatibilityDelegatesToService(t *testing.T) {
	networkPublicAddressFacadeTestMu.Lock()
	defer networkPublicAddressFacadeTestMu.Unlock()

	facade := &fakeNetworkPublicAddressFacade{
		resp: &models.NetworkCheckPublicNetResponse{
			Result: &models.NetworkCheckPublicNetResponseResult{Address: "203.0.113.10"},
		},
	}
	original := newNetworkPublicAddressService
	newNetworkPublicAddressService = func() networkPublicAddressFacade { return facade }
	defer func() { newNetworkPublicAddressService = original }()

	req := httptest.NewRequest("POST", "/network/check-public-net", strings.NewReader(`{"ipVersion":"ipv4"}`))

	resp, err := NetworkCheckPublicNet(nil, req)
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if facade.ipVersion != "ipv4" {
		t.Fatalf("expected ipVersion to be delegated, got %q", facade.ipVersion)
	}
	if resp == nil || resp.Result == nil || resp.Result.Address != "203.0.113.10" {
		t.Fatalf("unexpected delegated response: %#v", resp)
	}
}

func TestNetworkCheckPublicNetCompatibilityPropagatesServiceError(t *testing.T) {
	networkPublicAddressFacadeTestMu.Lock()
	defer networkPublicAddressFacadeTestMu.Unlock()

	delegateErr := errors.New("delegate failed")
	facade := &fakeNetworkPublicAddressFacade{err: delegateErr}
	original := newNetworkPublicAddressService
	newNetworkPublicAddressService = func() networkPublicAddressFacade { return facade }
	defer func() { newNetworkPublicAddressService = original }()

	req := httptest.NewRequest("POST", "/network/check-public-net", strings.NewReader(`{"ipVersion":"ipv6"}`))

	if _, err := NetworkCheckPublicNet(nil, req); !errors.Is(err, delegateErr) {
		t.Fatalf("expected delegated error, got %v", err)
	}
	if facade.ipVersion != "ipv6" {
		t.Fatalf("expected ipVersion to be delegated, got %q", facade.ipVersion)
	}
}

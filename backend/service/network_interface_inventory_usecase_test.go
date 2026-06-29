package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var networkInterfaceInventoryFacadeTestMu sync.Mutex

type fakeNetworkInterfaceInventoryFacade struct {
	interfaces []*models.NetworkInterfaceInfo
	err        error
	calls      int
}

func (svc *fakeNetworkInterfaceInventoryFacade) ListInventory(ctx context.Context) ([]*models.NetworkInterfaceInfo, error) {
	svc.calls++
	return svc.interfaces, svc.err
}

func TestNetworkInterfaceStatusCompatibilityDelegatesToService(t *testing.T) {
	networkInterfaceInventoryFacadeTestMu.Lock()
	defer networkInterfaceInventoryFacadeTestMu.Unlock()

	oldFactory := newNetworkInterfaceInventoryService
	defer func() {
		newNetworkInterfaceInventoryService = oldFactory
	}()

	facade := &fakeNetworkInterfaceInventoryFacade{
		interfaces: []*models.NetworkInterfaceInfo{
			{Name: "wan", Proto: "dhcp"},
			{Name: "wan6", Proto: "dhcpv6"},
		},
	}
	newNetworkInterfaceInventoryService = func() networkInterfaceInventoryFacade {
		return facade
	}

	resp, err := NetworkInterfaceStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected NetworkInterfaceStatus error: %v", err)
	}
	if facade.calls != 1 {
		t.Fatalf("expected facade to be called once, got %d", facade.calls)
	}
	if len(resp.Result.Interfaces) != 2 || resp.Result.Interfaces[1].Proto != "dhcpv6" {
		t.Fatalf("expected full inventory response, got %#v", resp.Result.Interfaces)
	}
}

func TestNetworkInterfaceStatusCompatibilityPropagatesServiceError(t *testing.T) {
	networkInterfaceInventoryFacadeTestMu.Lock()
	defer networkInterfaceInventoryFacadeTestMu.Unlock()

	oldFactory := newNetworkInterfaceInventoryService
	defer func() {
		newNetworkInterfaceInventoryService = oldFactory
	}()

	facadeErr := errors.New("inventory service failed")
	newNetworkInterfaceInventoryService = func() networkInterfaceInventoryFacade {
		return &fakeNetworkInterfaceInventoryFacade{err: facadeErr}
	}

	if _, err := NetworkInterfaceStatus(context.Background()); !errors.Is(err, facadeErr) {
		t.Fatalf("expected service error, got %v", err)
	}
}

func TestNetworkInterfaceGetConfigCompatibilityDelegatesToServiceAndFiltersDhcpv6(t *testing.T) {
	networkInterfaceInventoryFacadeTestMu.Lock()
	defer networkInterfaceInventoryFacadeTestMu.Unlock()

	oldFactory := newNetworkInterfaceInventoryService
	oldReadNetworkPortStatus := readNetworkPortStatus
	defer func() {
		newNetworkInterfaceInventoryService = oldFactory
		readNetworkPortStatus = oldReadNetworkPortStatus
	}()

	facade := &fakeNetworkInterfaceInventoryFacade{
		interfaces: []*models.NetworkInterfaceInfo{
			{Name: "wan", Proto: "dhcp"},
			{Name: "wan6", Proto: "dhcpv6"},
		},
	}
	newNetworkInterfaceInventoryService = func() networkInterfaceInventoryFacade {
		return facade
	}
	readNetworkPortStatus = func(ctx context.Context) ([]*models.NetworkPortInfo, error) {
		return []*models.NetworkPortInfo{{Name: "eth0"}}, nil
	}

	resp, err := NetworkInterfaceGetConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected NetworkInterfaceGetConfig error: %v", err)
	}
	if facade.calls != 1 {
		t.Fatalf("expected facade to be called once, got %d", facade.calls)
	}
	if len(resp.Result.Devices) != 1 || resp.Result.Devices[0].Name != "eth0" {
		t.Fatalf("expected devices to pass through, got %#v", resp.Result.Devices)
	}
	if len(resp.Result.Interfaces) != 1 || resp.Result.Interfaces[0].Proto == "dhcpv6" {
		t.Fatalf("expected dhcpv6 interfaces to be filtered, got %#v", resp.Result.Interfaces)
	}
}

func TestNetworkInterfaceGetConfigCompatibilityPropagatesServiceError(t *testing.T) {
	networkInterfaceInventoryFacadeTestMu.Lock()
	defer networkInterfaceInventoryFacadeTestMu.Unlock()

	oldFactory := newNetworkInterfaceInventoryService
	oldReadNetworkPortStatus := readNetworkPortStatus
	defer func() {
		newNetworkInterfaceInventoryService = oldFactory
		readNetworkPortStatus = oldReadNetworkPortStatus
	}()

	facadeErr := errors.New("inventory service failed")
	newNetworkInterfaceInventoryService = func() networkInterfaceInventoryFacade {
		return &fakeNetworkInterfaceInventoryFacade{err: facadeErr}
	}
	readNetworkPortStatus = func(ctx context.Context) ([]*models.NetworkPortInfo, error) {
		return []*models.NetworkPortInfo{{Name: "eth0"}}, nil
	}

	if _, err := NetworkInterfaceGetConfig(context.Background()); !errors.Is(err, facadeErr) {
		t.Fatalf("expected service error, got %v", err)
	}
}

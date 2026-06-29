package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeNetworkDeviceListFacade struct {
	devices []*models.DeviceInfo
	err     error
}

func (svc fakeNetworkDeviceListFacade) List(ctx context.Context) ([]*models.DeviceInfo, error) {
	return svc.devices, svc.err
}

func TestNetworkDeviceListDelegatesToFacade(t *testing.T) {
	original := newNetworkDeviceListService
	defer func() { newNetworkDeviceListService = original }()

	expected := []*models.DeviceInfo{{Ipv4addr: "192.168.1.10", Mac: "AA:BB"}}
	newNetworkDeviceListService = func() networkDeviceListFacade {
		return fakeNetworkDeviceListFacade{devices: expected}
	}

	resp, err := NetworkDeviceList(context.Background())
	if err != nil {
		t.Fatalf("NetworkDeviceList returned error: %v", err)
	}
	if !reflect.DeepEqual(resp.Result.Devices, expected) {
		t.Fatalf("devices = %#v, want %#v", resp.Result.Devices, expected)
	}
}

func TestNetworkDeviceListPropagatesFacadeError(t *testing.T) {
	original := newNetworkDeviceListService
	defer func() { newNetworkDeviceListService = original }()

	expectedErr := errors.New("facade failed")
	newNetworkDeviceListService = func() networkDeviceListFacade {
		return fakeNetworkDeviceListFacade{err: expectedErr}
	}

	if _, err := NetworkDeviceList(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("NetworkDeviceList error = %v, want expectedErr", err)
	}
}

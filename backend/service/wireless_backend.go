package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) WirelessListIfaces(ctx context.Context) (*models.WirelessListIfaceResponse, error) {
	return WirelessListIfaces(ctx)
}

func (backend *ServiceBackend) WirelessEnableIface(ctx context.Context, req models.WirelessEnableIfaceRequest) error {
	return WirelessEnableIfaceWithRequest(ctx, req)
}

func (backend *ServiceBackend) WirelessSetDevicePower(ctx context.Context, req models.WirelessSetDevicePowerRequest) error {
	return WirelessSetDevicePowerWithRequest(ctx, req)
}

func (backend *ServiceBackend) WirelessEditIface(ctx context.Context, req models.WirelessIfaceInfo) error {
	return WirelessEditIfaceWithRequest(ctx, req)
}

func (backend *ServiceBackend) WirelessQuickSetupIface(ctx context.Context, req models.WirelessQuickSetupRequest) error {
	return WirelessQuickSetupIfaceWithRequest(ctx, req)
}

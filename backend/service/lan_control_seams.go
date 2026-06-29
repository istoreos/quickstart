package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

var newLanFloatGatewayWriteService = func() lanFloatGatewayWriteFacade {
	return NewDefaultLanFloatGatewayWriteService()
}

type lanSpeedLimitWriteFacade interface {
	UpsertSpeedLimitRule(ctx context.Context, input SpeedLimitWriteInput) error
	SetSpeedLimitModule(ctx context.Context, input SpeedLimitModuleInput) error
}

var newLanSpeedLimitWriteService = func() lanSpeedLimitWriteFacade {
	return NewDefaultLanSpeedLimitWriteService()
}

type lanStaticAssignmentWriteFacade interface {
	ApplyStaticAssignment(ctx context.Context, input StaticAssignmentWriteInput) error
}

var newLanStaticAssignmentWriteService = func() lanStaticAssignmentWriteFacade {
	return NewDefaultLanStaticAssignmentWriteService()
}

type lanGlobalConfigFacade interface {
	GetGlobalConfigs(ctx context.Context) (*models.LANCtrlGlobalConfigResponse, error)
}

var newLanGlobalConfigService = func() lanGlobalConfigFacade {
	return NewLanGlobalConfigService()
}

type lanDeviceListFacade interface {
	GetListDevices(ctx context.Context, serviceBackend *ServiceBackend) (*models.LANDeviceResponse, error)
}

var newLanDeviceListService = func() lanDeviceListFacade {
	return NewLanDeviceListService()
}

type lanStaticDeviceListFacade interface {
	GetListStaticDevices(ctx context.Context) (*models.LANCtrlStaticAssignedResponse, error)
}

var newLanStaticDeviceListService = func() lanStaticDeviceListFacade {
	return NewLanStaticDeviceListService()
}

type lanSpeedLimitedDeviceListFacade interface {
	GetListSpeedLimitedDevices(ctx context.Context) (*models.LANCtrlSpeedLimitResponse, error)
}

var newLanSpeedLimitedDeviceListService = func() lanSpeedLimitedDeviceListFacade {
	return NewLanSpeedLimitedDeviceListService()
}

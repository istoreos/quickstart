package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/interfacewrite"
)

type networkInterfaceConfigFacade interface {
	ApplyConfigSet(ctx context.Context, input NetworkInterfaceWriteInput) (*models.SDKNormalResponse, error)
}

var newNetworkInterfaceConfigService = func() networkInterfaceConfigFacade {
	return NewDefaultNetworkInterfaceConfigService()
}

func NewDefaultNetworkInterfaceConfigService() *interfacewrite.Service {
	return interfacewrite.NewService(NewDefaultNetworkInterfaceConfigStore(), NewDefaultNetworkInterfaceConfigApply())
}

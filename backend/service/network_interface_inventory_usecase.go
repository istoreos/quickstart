package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/interfaceinventory"
)

type networkInterfaceInventoryFacade interface {
	ListInventory(ctx context.Context) ([]*models.NetworkInterfaceInfo, error)
}

var newNetworkInterfaceInventoryService = func() networkInterfaceInventoryFacade {
	return NewDefaultNetworkInterfaceInventoryService()
}

func NewDefaultNetworkInterfaceInventoryService() *interfaceinventory.Service {
	return interfaceinventory.NewService(
		newDefaultNetworkPortStatusReader(),
		newDefaultNetworkInterfaceInventoryReader(),
		newDefaultNetworkInterfaceFirewallBindingReader(),
		newDefaultNetworkInterfacePortAttachmentResolver(),
	)
}

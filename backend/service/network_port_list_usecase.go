package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/portlist"
)

type networkPortListFacade interface {
	GetPortList(ctx context.Context) (*models.NetworkPortListResponse, error)
}

var newNetworkPortListService = func() networkPortListFacade {
	return portlist.NewService(newDefaultNetworkPortStatusReader(), newDefaultNetworkPortMembershipReader())
}

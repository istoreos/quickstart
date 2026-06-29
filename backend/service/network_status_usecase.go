package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	networkstatus "github.com/linkease/quick-start/istore-backend/modules/network/status"
)

type networkStatusFacade interface {
	GetNetworkStatus(ctx context.Context, setupFinish bool) (*models.NetworkStatusResponse, error)
}

var newNetworkStatusService = func(netChecker *NetworkOnlineChecker) networkStatusFacade {
	return newDefaultNetworkStatusService(netChecker)
}

func newDefaultNetworkStatusService(netChecker *NetworkOnlineChecker) *networkstatus.Service {
	return networkstatus.NewService(
		newDefaultNetworkStatusReader(),
		newDefaultNetworkOnlineStatusChecker(netChecker),
		newDefaultNetworkSetupMarker(),
	)
}

package service

import (
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/publicaddress"
)

type networkPublicAddressFacade interface {
	CheckPublicAddress(ipVersion string) (*models.NetworkCheckPublicNetResponse, error)
}

var newNetworkPublicAddressService = func() networkPublicAddressFacade {
	return publicaddress.NewService(newDefaultNetworkPublicAddressReader(), newDefaultNetworkPublicAddressClassifier())
}

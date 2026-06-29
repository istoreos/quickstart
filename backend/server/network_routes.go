package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/network"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ network.Backend = (*service.ServiceBackend)(nil)

func registerNetworkRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	network.RegisterRoutes(router, serviceBackend)
}

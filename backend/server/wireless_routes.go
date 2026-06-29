package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/wireless"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ wireless.Backend = (*service.ServiceBackend)(nil)

func registerWirelessRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	wireless.RegisterRoutes(router, serviceBackend)
}

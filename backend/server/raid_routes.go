package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/raid"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ raid.Backend = (*service.ServiceBackend)(nil)

func registerRaidRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	raid.RegisterRoutes(router, serviceBackend)
}

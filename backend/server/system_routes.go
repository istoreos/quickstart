package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/system"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ system.Backend = (*service.ServiceBackend)(nil)

func registerSystemRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	system.RegisterRoutes(router, serviceBackend)
}

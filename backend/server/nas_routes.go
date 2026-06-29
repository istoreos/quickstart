package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/nas"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ nas.Backend = (*service.ServiceBackend)(nil)

func registerNasRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	nas.RegisterRoutes(router, serviceBackend)
}

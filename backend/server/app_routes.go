package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/app"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ app.Backend = (*service.ServiceBackend)(nil)

func registerAppRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	app.RegisterRoutes(router, serviceBackend)
}

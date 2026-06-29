package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/quickstart"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ quickstart.Backend = (*service.ServiceBackend)(nil)

func registerQuickstartRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	quickstart.RegisterRoutes(router, serviceBackend)
}

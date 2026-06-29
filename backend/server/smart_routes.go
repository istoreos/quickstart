package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/smart"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ smart.Backend = (*service.ServiceBackend)(nil)

func registerSmartRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	smart.RegisterRoutes(router, serviceBackend)
}

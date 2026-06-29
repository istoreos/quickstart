package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/dhns"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ dhns.Backend = (*service.ServiceBackend)(nil)

func dhnsRouterInit(router *httprouter.Router, serviceBackend *service.ServiceBackend) *httprouter.Router {
	return dhns.RegisterRoutes(router, serviceBackend)
}

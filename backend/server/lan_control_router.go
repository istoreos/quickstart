package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/lancontrol"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ lancontrol.Backend = (*service.ServiceBackend)(nil)

func lanControlRouterInit(router *httprouter.Router, serviceBackend *service.ServiceBackend) *httprouter.Router {
	lancontrol.RegisterRoutes(router, serviceBackend)
	return router
}

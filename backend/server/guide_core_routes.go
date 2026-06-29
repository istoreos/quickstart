package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/guidecore"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ guidecore.Backend = (*service.ServiceBackend)(nil)

func registerGuideCoreRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	guidecore.RegisterRoutes(router, serviceBackend)
}

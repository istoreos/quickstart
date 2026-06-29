package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/guidestorage"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ guidestorage.Backend = (*service.ServiceBackend)(nil)

func registerGuideStorageRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	guidestorage.RegisterRoutes(router, serviceBackend)
}

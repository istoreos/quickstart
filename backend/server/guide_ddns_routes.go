package server

import (
	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/guideddns"
	"github.com/linkease/quick-start/istore-backend/service"
)

var _ guideddns.Backend = (*service.ServiceBackend)(nil)

func registerGuideDDNSRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	guideddns.RegisterRoutes(router, serviceBackend)
}

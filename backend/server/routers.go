package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/service"
)

func RouterInit(serviceBackend *service.ServiceBackend) http.Handler {
	router := httprouter.New()

	registerNetworkRoutes(router, serviceBackend)
	registerSystemRoutes(router, serviceBackend)
	registerAppRoutes(router, serviceBackend)
	registerRaidRoutes(router, serviceBackend)
	registerSmartRoutes(router, serviceBackend)
	registerQuickstartRoutes(router, serviceBackend)
	registerNasRoutes(router, serviceBackend)
	registerGuideCoreRoutes(router, serviceBackend)
	registerGuideStorageRoutes(router, serviceBackend)
	registerGuideDDNSRoutes(router, serviceBackend)
	registerShareRoutes(router, serviceBackend)
	registerWirelessRoutes(router, serviceBackend)
	router = lanControlRouterInit(router, serviceBackend)

	return router
}

func UnixRouterInit(serviceBackend *service.ServiceBackend) http.Handler {
	router := httprouter.New()

	registerLCDRoutes(router, serviceBackend)
	dhnsRouterInit(router, serviceBackend)

	return router
}

package server

import (
	"github.com/istoreos/quickstart/backend/modules/homecards"
	"github.com/istoreos/quickstart/backend/modules/homeconfig"
	"github.com/julienschmidt/httprouter"
)

func registerHomeConfigRoutes(router *httprouter.Router) {
	router.Handler("GET", "/cgi-bin/luci/istore/home/v1/cards/", homecards.Handler("/usr/share/quickstart/cards.d", homeconfig.RootAuthorizer(homeconfig.LocalSession)))
	store := homeconfig.NewInitializedStore(homeconfig.NewStore("/etc/quickstart"), homeconfig.LocalLegacySettings)
	registerHomeConfigRepository(router, store, homeconfig.RootAuthorizer(homeconfig.LocalSession))
}
func registerHomeConfigRepository(router *httprouter.Router, store homeconfig.Repository, authorize homeconfig.Authorize) {
	for _, domain := range []string{"layout", "launcher"} {
		path := "/cgi-bin/luci/istore/home/v1/" + domain + "/"
		handler := homeconfig.Handler(store, domain, authorize)
		router.Handler("GET", path, handler)
		router.Handler("POST", path, handler)
		router.Handler("POST", path+"restore/", homeconfig.RestoreHandler(store, domain, authorize))
		router.Handler("POST", path+"reset-damaged/", homeconfig.ResetHandler(store, domain, authorize))
	}
}

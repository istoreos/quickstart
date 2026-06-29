package dhns

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/models"
)

type Backend interface {
	DhnsDisabled() bool
	DhnsConnect(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
	DhnsProxy(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
	DhnsForward(w http.ResponseWriter, r *http.Request, ps httprouter.Params)
	HandleDhnsChange(evt models.DHNSChangeRequest) bool
	HandleDhcpValid(info models.DHNSDhcpValidRequest)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) *httprouter.Router {
	serviceDisabled := func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.Error(w, "ServiceDisabled", http.StatusNotImplemented)
	}

	if backend.DhnsDisabled() {
		router.GET("/api/dhns/connect/", serviceDisabled)
		router.GET("/api/dhns/proxy/", serviceDisabled)
		router.GET("/api/dhns/forward/", serviceDisabled)
		router.POST("/api/dhns/dhnsChange/", serviceDisabled)
		router.POST("/api/dhns/dhcpValid/", serviceDisabled)
		return router
	}

	router.GET("/api/dhns/connect/", backend.DhnsConnect)
	router.GET("/api/dhns/proxy/", backend.DhnsProxy)
	router.GET("/api/dhns/forward/", backend.DhnsForward)
	router.POST("/api/dhns/dhnsChange/", handleDhnsChange(backend))
	router.POST("/api/dhns/dhcpValid/", handleDhcpValid(backend))
	return router
}

func handleDhnsChange(backend Backend) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		var evt models.DHNSChangeRequest
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if backend.HandleDhnsChange(evt) {
			w.Write([]byte("OK"))
			return
		}
		http.Error(w, "error event", http.StatusBadRequest)
	}
}

func handleDhcpValid(backend Backend) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		var info models.DHNSDhcpValidRequest
		if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if info.Ip == "" || info.Subnet == "" || info.Gateway == "" {
			http.Error(w, "empty ip", http.StatusBadRequest)
			return
		}
		backend.HandleDhcpValid(info)
		w.Write([]byte("Found DHCP Server"))
	}
}

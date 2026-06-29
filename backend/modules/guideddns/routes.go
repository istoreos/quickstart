package guideddns

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/models"
)

const (
	guideDDNSGeneralError   = -1000
	guideDDNSForbiddenError = -1001
)

type guideDDNSJSONHandler func(context.Context, *http.Request) (any, error)

type Backend interface {
	GetGuideDdns(ctx context.Context) (*models.GuideDdnsResponse, error)
	PostGuideDdns(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostGuideDdnsto(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	PostGuideDdnstoAddress(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error)
	GetGuideDdnstoConfig(ctx context.Context) (*models.GuideDdnstoConfigResponse, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	getGuideDDNSJSONAliases(router, []string{
		"/cgi-bin/luci/istore/guide/ddns/",
		"/cgi-bin/luci/istore/u/guide/ddns/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideDdns(ctx)
	})

	postGuideDDNSJSONAliases(router, []string{
		"/cgi-bin/luci/istore/guide/ddns/",
		"/cgi-bin/luci/istore/u/guide/ddns/",
	}, func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideDdns(ctx, r)
	})

	postGuideDDNSJSON(router, "/cgi-bin/luci/istore/guide/ddnsto/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideDdnsto(ctx, r)
	})

	postGuideDDNSJSON(router, "/cgi-bin/luci/istore/guide/ddnsto/address/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.PostGuideDdnstoAddress(ctx, r)
	})

	getGuideDDNSJSON(router, "/cgi-bin/luci/istore/guide/ddnsto/config/", func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetGuideDdnstoConfig(ctx)
	})
}

func guideDDNSAuthenticatedJSON(fn guideDDNSJSONHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		if r.Header.Get("X-Forwarded-Sid") == "" {
			respondGuideDDNSErrorString(guideDDNSForbiddenError, "", "", w)
			return
		}
		resp, err := fn(ctx, r)
		if err != nil {
			respondGuideDDNSErrorString(guideDDNSGeneralError, "", err.Error(), w)
			return
		}
		respondGuideDDNSJSON(http.StatusOK, resp, w)
	}
}

func registerGuideDDNSJSON(router *httprouter.Router, method string, path string, fn guideDDNSJSONHandler) {
	router.Handle(method, path, guideDDNSAuthenticatedJSON(fn))
}

func getGuideDDNSJSON(router *httprouter.Router, path string, fn guideDDNSJSONHandler) {
	registerGuideDDNSJSON(router, http.MethodGet, path, fn)
}

func postGuideDDNSJSON(router *httprouter.Router, path string, fn guideDDNSJSONHandler) {
	registerGuideDDNSJSON(router, http.MethodPost, path, fn)
}

func getGuideDDNSJSONAliases(router *httprouter.Router, paths []string, fn guideDDNSJSONHandler) {
	for _, path := range paths {
		getGuideDDNSJSON(router, path, fn)
	}
}

func postGuideDDNSJSONAliases(router *httprouter.Router, paths []string, fn guideDDNSJSONHandler) {
	for _, path := range paths {
		postGuideDDNSJSON(router, path, fn)
	}
}

func respondGuideDDNSJSON(code int, obj any, w http.ResponseWriter) {
	if code == 0 {
		code = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(obj)
}

func respondGuideDDNSErrorString(code int, scope, msg string, w http.ResponseWriter) {
	sCode := models.ResponseSuccess(code)
	respondGuideDDNSJSON(http.StatusOK, &models.SDKNormalResponse{
		Success: &sCode,
		Scope:   models.ResponseScope(scope),
		Error:   models.ResponseError(msg),
	}, w)
}

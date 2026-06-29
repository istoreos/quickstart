package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/lib/scope_error"
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/service"
)

const (
	GeneralError   = -1000
	ForbiddenError = -1001
)

type JSONHandler func(context.Context, *http.Request) (any, error)

func RespondJSON(code int, obj any, w http.ResponseWriter) {
	if code == 0 {
		code = http.StatusOK
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(obj)
}

func RespondErrorString(code int, scope, lang, msg string, w http.ResponseWriter) {
	sCode := models.ResponseSuccess(code)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&models.SDKNormalResponse{
		Success: &sCode,
		Scope:   models.ResponseScope(scope),
		Error:   models.ResponseError(msg),
	})
}

func RespondError(err error, w http.ResponseWriter) {
	var code int
	var scope, lang, msg, detail string
	var scopeErr *scope_error.ScopeErr
	if errors.As(err, &scopeErr) {
		if scopeErr.Scope() != "" {
			scope = scopeErr.Scope()
		} else {
			scope = "common"
		}
		code = scopeErr.Code()
	}
	if code == 0 {
		code = GeneralError
	}
	msg, detail = service.ScopeMessage(code, scope, err.Error(), lang)

	sCode := models.ResponseSuccess(code)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&models.SDKNormalResponse{
		Success: &sCode,
		Scope:   models.ResponseScope(scope),
		Error:   models.ResponseError(msg),
		Detail:  detail,
	})
}

func AuthenticatedJSON(fn JSONHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		sid := r.Header.Get("X-Forwarded-Sid")
		if sid == "" {
			RespondErrorString(ForbiddenError, "", "Forbidden", "", w)
			return
		}
		resp, err := fn(ctx, r)
		if err != nil {
			RespondError(err, w)
			return
		}
		RespondJSON(http.StatusOK, resp, w)
	}
}

func RegisterJSON(router *httprouter.Router, method string, path string, fn JSONHandler) {
	router.Handle(method, path, AuthenticatedJSON(fn))
}

func GetJSON(router *httprouter.Router, path string, fn JSONHandler) {
	RegisterJSON(router, http.MethodGet, path, fn)
}

func PostJSON(router *httprouter.Router, path string, fn JSONHandler) {
	RegisterJSON(router, http.MethodPost, path, fn)
}

func GetJSONAliases(router *httprouter.Router, paths []string, fn JSONHandler) {
	for _, path := range paths {
		GetJSON(router, path, fn)
	}
}

func PostJSONAliases(router *httprouter.Router, paths []string, fn JSONHandler) {
	for _, path := range paths {
		PostJSON(router, path, fn)
	}
}

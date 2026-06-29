package lcd

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/internal/httpapi"
)

type Backend interface {
	GetLCDST7789(ctx context.Context, r *http.Request) (any, error)
	GetLcdSimple(ctx context.Context, r *http.Request) (any, error)
}

func RegisterRoutes(router *httprouter.Router, backend Backend) {
	router.GET("/api/lcd/st7789/", unixJSON(func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLCDST7789(ctx, r)
	}))

	router.GET("/api/lcd/simple/", unixJSON(func(ctx context.Context, r *http.Request) (any, error) {
		return backend.GetLcdSimple(ctx, r)
	}))
}

func unixJSON(fn httpapi.JSONHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Set("Connection", "close")
		defer r.Body.Close()

		resp, err := fn(r.Context(), r)
		if err != nil {
			httpapi.RespondError(err, w)
			return
		}
		httpapi.RespondJSON(http.StatusOK, resp, w)
	}
}

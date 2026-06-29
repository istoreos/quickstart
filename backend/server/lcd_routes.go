package server

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/modules/lcd"
	"github.com/linkease/quick-start/istore-backend/service"
)

type lcdServiceBackend struct {
	backend *service.ServiceBackend
}

var _ lcd.Backend = (*lcdServiceBackend)(nil)

func registerLCDRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	lcd.RegisterRoutes(router, &lcdServiceBackend{backend: serviceBackend})
}

func (backend *lcdServiceBackend) GetLCDST7789(ctx context.Context, r *http.Request) (any, error) {
	return backend.backend.GetLCDST7789(ctx, r)
}

func (backend *lcdServiceBackend) GetLcdSimple(ctx context.Context, r *http.Request) (any, error) {
	return backend.backend.GetLcdSimple(ctx, r)
}

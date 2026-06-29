package service

import (
	"context"
	"net/http"
)

func (backend *ServiceBackend) GetLCDST7789(ctx context.Context, r *http.Request) (*LcdSt7789Response, error) {
	return LcdSt7789(ctx, r, backend, backend.st)
}

func (backend *ServiceBackend) GetLcdSimple(ctx context.Context, r *http.Request) (*LcdSimpleResponse, error) {
	return LcdSimple(ctx, r, backend, backend.st)
}

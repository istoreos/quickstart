package service

import (
	"context"
	"net/http"

	"github.com/istoreos/quickstart/backend/models"
)

func (backend *ServiceBackend) GetRouterContextV2(ctx context.Context, request *http.Request) (*models.RouterContextResponse, error) {
	return backend.routerContext.Get(ctx, request.URL.Query().Get("lan"))
}

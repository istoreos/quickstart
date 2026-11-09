package service

import (
	"context"
	"net/http"

	"github.com/istoreos/quickstart/backend/models"
)

func (backend *ServiceBackend) GetRouterContextV2(ctx context.Context, request *http.Request) (*models.RouterContextResponse, error) {
	backend.mu.Lock()
	if backend.routerContext == nil {
		backend.routerContext = NewDefaultRouterContextModule()
	}
	module := backend.routerContext
	backend.mu.Unlock()
	if scope := selectedLANScope(ctx); scope != nil {
		return module.Get(ctx, scope.NetworkID)
	}
	return module.Get(ctx, request.URL.Query().Get("lan"))
}

package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) GetGlobalFolders(ctx context.Context) (*models.GlobalFoldersResponse, error) {
	return GlobalFoldersGetConfig(ctx)
}

func (backend *ServiceBackend) PostGlobalFolders(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return GlobalFoldersPostConfig(ctx, r)
}

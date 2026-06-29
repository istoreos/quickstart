package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) SetQuickstartConfig(ctx context.Context, req models.QuickstartConfigRequest) (*models.SDKNormalResponse, error) {
	return QuickstartSetConfigValue(ctx, req)
}

func (backend *ServiceBackend) GetQuickstartConfig(ctx context.Context, req models.QuickstartGetConfigRequest) (*models.QuickstartConfigResponse, error) {
	return QuickstartGetConfigValue(ctx, req)
}

func (backend *ServiceBackend) DeleteQuickstartConfig(ctx context.Context, req models.QuickstartDeleteConfigRequest) (*models.SDKNormalResponse, error) {
	return QuickstartDeleteConfigValue(ctx, req)
}

func (backend *ServiceBackend) PostQuickstartConfigSet(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return QuickstartSetConfig(ctx, r)
}

func (backend *ServiceBackend) PostQuickstartConfigGet(ctx context.Context, r *http.Request) (*models.QuickstartConfigResponse, error) {
	return QuickstartGetConfig(ctx, r)
}

func (backend *ServiceBackend) PostQuickstartConfigDelete(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return QuickstartDeleteConfig(ctx, r)
}

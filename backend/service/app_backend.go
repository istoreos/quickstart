package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) PostAppCheck(ctx context.Context, r *http.Request) (*models.AppCheckResponse, error) {
	return AppCheck(ctx, r)
}

func (backend *ServiceBackend) CheckApp(ctx context.Context, req models.AppCheckRequest) (*models.AppCheckResponse, error) {
	return AppCheckValue(ctx, req)
}

func (backend *ServiceBackend) PostAppInstall(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return AppInstall(ctx, r)
}

func (backend *ServiceBackend) InstallAppPackage(ctx context.Context, req models.AppInstallRequest) (*models.SDKNormalResponse, error) {
	return AppInstallValue(ctx, req)
}

func (backend *ServiceBackend) AppInstalledList(ctx context.Context, r *http.Request) (models.AppInstalledListResponse, error) {
	return AppInstalledList(ctx, r)
}

func (backend *ServiceBackend) ListInstalledApps(ctx context.Context) (models.AppInstalledListResponse, error) {
	return AppInstalledListValue(ctx)
}

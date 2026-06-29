package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) GetSmartList(ctx context.Context) (*models.SmartListResponse, error) {
	return SmartGetList(ctx)
}

func (backend *ServiceBackend) GetSmartLog(ctx context.Context) (*models.SmartLogResponse, error) {
	return SmartGetLog(ctx)
}

func (backend *ServiceBackend) GetSmartConfig(ctx context.Context) (*models.SmartConfigResponse, error) {
	return SmartGetConfig(ctx)
}

func (backend *ServiceBackend) PostSmartConfig(ctx context.Context, req models.SmartConfigRequest) (*models.SmartConfigResponse, error) {
	return SmartPostConfigTyped(ctx, req)
}

func (backend *ServiceBackend) PostSmartTest(ctx context.Context, req models.SmartTestRequest) (*models.SmartTestResponse, error) {
	return SmartPostTestTyped(ctx, req)
}

func (backend *ServiceBackend) PostSmartTestResult(ctx context.Context, req models.SmartTestResultRequest) (*models.SmartTestResultResponse, error) {
	return SmartPostTestResultTyped(ctx, req)
}

func (backend *ServiceBackend) PostSmartAttributeResult(ctx context.Context, req models.SmartAttributeResultRequest) (*models.SmartAttributeResultResponse, error) {
	return SmartPostAttributeResultTyped(ctx, req)
}

func (backend *ServiceBackend) PostSmartExtendResult(ctx context.Context, req models.SmartExtendResultRequest) (*models.SmartExtendResultResponse, error) {
	return SmartPostExtendResultTyped(ctx, req)
}

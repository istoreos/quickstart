package server

import (
	"context"

	"github.com/julienschmidt/httprouter"
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/share"
	"github.com/linkease/quick-start/istore-backend/service"
)

type shareBackend struct{}

var _ share.Backend = (*shareBackend)(nil)

func registerShareRoutes(router *httprouter.Router, serviceBackend *service.ServiceBackend) {
	share.RegisterRoutes(router, shareBackend{})
}

func (backend shareBackend) GetShareUserList(ctx context.Context) (*models.ShareUserListResponse, error) {
	return service.ShareUserList(ctx)
}

func (backend shareBackend) PostShareUserCreate(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error) {
	return service.ShareUserCreateTyped(ctx, req)
}

func (backend shareBackend) PostShareUserUpdate(ctx context.Context, req models.ShareUserCreateRequest) (*models.SDKNormalResponse, error) {
	return service.ShareUserUpdateTyped(ctx, req)
}

func (backend shareBackend) PostShareUserDelete(ctx context.Context, req models.ShareUserDeleteRequest) (*models.SDKNormalResponse, error) {
	return service.ShareUserDeleteTyped(ctx, req)
}

func (backend shareBackend) GetShareServiceList(ctx context.Context) (*models.ShareServiceListResponse, error) {
	return service.ShareServiceList(ctx)
}

func (backend shareBackend) PostShareServiceCreate(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error) {
	return service.ShareServiceCreateTyped(ctx, req)
}

func (backend shareBackend) PostShareServiceUpdate(ctx context.Context, req models.ShareServiceCreateRequest) (*models.SDKNormalResponse, error) {
	return service.ShareServiceUpdateTyped(ctx, req)
}

func (backend shareBackend) PostShareServiceDelete(ctx context.Context, req models.ShareServicDeleteRequest) (*models.SDKNormalResponse, error) {
	return service.ShareServiceDeleteTyped(ctx, req)
}

func (backend shareBackend) GetShareWebdavConfig(ctx context.Context) (*models.ShareProtocolWebdavResponse, error) {
	return service.ShareWebdavConfig(ctx)
}

func (backend shareBackend) PostShareWebdavConfig(ctx context.Context, req models.ShareProtocolWebdavConfig) (*models.SDKNormalResponse, error) {
	return service.ShareWebdavConfigUpdateTyped(ctx, req)
}

func (backend shareBackend) GetShareSambaConfig(ctx context.Context) (*models.ShareProtocolSambaResponse, error) {
	return service.ShareSambaConfig(ctx)
}

func (backend shareBackend) PostShareSambaConfig(ctx context.Context, req models.ShareProtocolSambaConfig) (*models.SDKNormalResponse, error) {
	return service.ShareSambaConfigUpdateTyped(ctx, req)
}

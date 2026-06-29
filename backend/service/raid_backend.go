package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) PostRaidCreate(ctx context.Context, r *http.Request) (*models.NasDiskPartitionFormatResponse, error) {
	return RaidPostCreate(ctx, r)
}

func (backend *ServiceBackend) PostRaidDelete(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return RaidPostDelete(ctx, r)
}

func (backend *ServiceBackend) PostRaidAdd(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return RaidPostAdd(ctx, r)
}

func (backend *ServiceBackend) PostRaidRemove(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return RaidPostRemove(ctx, r)
}

func (backend *ServiceBackend) PostRaidRecover(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return RaidPostRecover(ctx, r)
}

func (backend *ServiceBackend) PostRaidDetail(ctx context.Context, r *http.Request) (*models.RaidDetailResponse, error) {
	return RaidPostDetail(ctx, r)
}

func (backend *ServiceBackend) GetRaidList(ctx context.Context) (*models.RaidListResponse, error) {
	return RaidGetList(ctx)
}

func (backend *ServiceBackend) GetRaidCreateList(ctx context.Context) (*models.RaidCreateListResponse, error) {
	return RaidGetCreateList(ctx)
}

func (backend *ServiceBackend) PostRaidAutoFix(ctx context.Context) (*models.SDKNormalResponse, error) {
	return RaidAutoFix(ctx)
}

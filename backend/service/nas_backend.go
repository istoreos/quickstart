package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
)

func (backend *ServiceBackend) GetNasDiskStatus(ctx context.Context) (*models.NasDiskStatusResponse, error) {
	return NasDiskStatus(ctx)
}

func (backend *ServiceBackend) GetNasServiceStatus(ctx context.Context) (*models.NasServiceResponse, error) {
	return NasServiceStatus(ctx)
}

func (backend *ServiceBackend) PostNasDiskInit(ctx context.Context, r *http.Request) (*models.NasDiskInitDiskResponse, error) {
	return NasDiskInit(ctx, r)
}

func (backend *ServiceBackend) PostNasDiskMountPoint(ctx context.Context, r *http.Request) (*models.NasDiskMountPointResponse, error) {
	req := models.NasDiskMountPointRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, err
	}
	return NasDiskMountPoint(ctx, req)
}

func (backend *ServiceBackend) PostNasDiskInitRest(ctx context.Context, r *http.Request) (*models.NasDiskInitDiskResponse, error) {
	return NasDiskInitRest(ctx, r)
}

func (backend *ServiceBackend) PostNasDiskPartFormat(ctx context.Context, r *http.Request) (*models.NasDiskPartitionFormatResponse, error) {
	return NasDiskPartitionFormat(ctx, r)
}

func (backend *ServiceBackend) PostNasSanboxFormat(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return NasSanboxPartitionFormat(ctx, r)
}

func (backend *ServiceBackend) PostNasSanboxCommit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return NasSanboxSubmit(ctx, r)
}

func (backend *ServiceBackend) PostNasSanboxReset(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return NasSanboxReset(ctx, r)
}

func (backend *ServiceBackend) PostNasSanboxExit(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	return NasSanboxExit(ctx, r)
}

func (backend *ServiceBackend) GetNasSanboxDisks(ctx context.Context) (*models.NasSandboxDisksResponse, error) {
	return NasSanboxDisks(ctx)
}

func (backend *ServiceBackend) GetNasSanboxStatus(ctx context.Context) (*models.NasSandboxStatusResponse, error) {
	return NasSanboxStatus(ctx)
}

func (backend *ServiceBackend) PostNasDiskPartMount(ctx context.Context, r *http.Request) (*models.NasDiskPartitionMountResponse, error) {
	return NasDiskPartitionMount(ctx, r)
}

func (backend *ServiceBackend) PostNasDiskSambaCreate(ctx context.Context, r *http.Request) (*models.NasSambaCreateResponse, error) {
	return NasServiceSambaCreate(ctx, r)
}

func (backend *ServiceBackend) PostNasDiskWebdavCreate(ctx context.Context, r *http.Request) (*models.NasWebdavCreateResponse, error) {
	return NasServiceWebdavCreate(ctx, r)
}

func (backend *ServiceBackend) PostNasDiskWebdavStatus(ctx context.Context, r *http.Request) (*models.NasWebdavStatusResponse, error) {
	return NasServiceWebdavStatus(ctx)
}

func (backend *ServiceBackend) PostNasDiskLinkeaseEnable(ctx context.Context, r *http.Request) (*models.NasLinkeaseEnableResponse, error) {
	return NasServiceLinkeaseEnable(ctx, r)
}

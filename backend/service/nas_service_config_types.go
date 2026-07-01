package service

import (
	"context"

	"github.com/istoreos/quickstart/backend/models"
)

type NasServiceStatusReader interface {
	ReadSambaShares() []*models.NasServiceSambaInfo
	ReadWebdavPort() (string, bool)
	ReadWebdavInfo() models.NasServiceWebdavInfo
	ReadLinkeaseInfo(ctx context.Context) (enabledByConfig bool, port string, err error)
}

type NasServiceRuntimeReader interface {
	ReadLANIPv4(ctx context.Context) (string, error)
	HasLinkeaseBinary() bool
}

package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/nas/serviceconfig"
)

type NasSambaCreateService = serviceconfig.SambaCreateService
type NasWebdavCreateService = serviceconfig.WebdavCreateService
type NasWebdavStatusService = serviceconfig.WebdavStatusService
type NasServiceStatusService = serviceconfig.StatusService

type nasSambaCreateFacade interface {
	Create(ctx context.Context, input NasSambaCreateInput) (*models.NasSambaCreateResponseResult, error)
}

var newNasSambaCreateServiceFacade = func() nasSambaCreateFacade {
	return newNasSambaCreateService()
}

func newNasSambaCreateService() *NasSambaCreateService {
	return serviceconfig.NewSambaCreateService(
		newDefaultNasServiceStatusReader(),
		newDefaultNasServiceRuntimeReader(),
		newDefaultNasServiceConfigWriter(),
		newDefaultNasSambaTemplateWriter(),
	)
}

type nasServiceStatusFacade interface {
	Read(ctx context.Context) (*models.NasServiceResponseResult, error)
}

var newNasServiceStatusServiceFacade = func() nasServiceStatusFacade {
	return newNasServiceStatusService()
}

func newNasServiceStatusService() *NasServiceStatusService {
	return serviceconfig.NewStatusService(newDefaultNasServiceStatusReader(), newDefaultNasServiceRuntimeReader())
}

type nasWebdavCreateFacade interface {
	Create(ctx context.Context, input NasWebdavCreateInput) (*models.NasWebdavCreateResponseResult, error)
}

var newNasWebdavCreateServiceFacade = func() nasWebdavCreateFacade {
	return newNasWebdavCreateService()
}

func newNasWebdavCreateService() *NasWebdavCreateService {
	return serviceconfig.NewWebdavCreateService(
		newDefaultNasServiceStatusReader(),
		newDefaultNasServiceRuntimeReader(),
		newDefaultNasServiceConfigWriter(),
	)
}

type nasWebdavStatusFacade interface {
	Read(ctx context.Context) (*models.NasWebdavStatusResponseResult, error)
}

var newNasWebdavStatusServiceFacade = func() nasWebdavStatusFacade {
	return newNasWebdavStatusService()
}

func newNasWebdavStatusService() *NasWebdavStatusService {
	return serviceconfig.NewWebdavStatusService(newDefaultNasServiceStatusReader())
}

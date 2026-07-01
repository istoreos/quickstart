package service

import (
	"context"

	"github.com/istoreos/quickstart/backend/models"
)

type nasServiceStatusFacade interface {
	Read(ctx context.Context) (*models.NasServiceResponseResult, error)
}

var newNasServiceStatusServiceFacade = func() nasServiceStatusFacade {
	return newNasServiceStatusService()
}

func newNasServiceStatusService() *NasServiceStatusService {
	return NewNasServiceStatusService(newDefaultNasServiceStatusReader(), newDefaultNasServiceRuntimeReader())
}

type NasServiceStatusService struct {
	statusReader  NasServiceStatusReader
	runtimeReader NasServiceRuntimeReader
}

func NewNasServiceStatusService(statusReader NasServiceStatusReader, runtimeReader NasServiceRuntimeReader) *NasServiceStatusService {
	return &NasServiceStatusService{
		statusReader:  statusReader,
		runtimeReader: runtimeReader,
	}
}

func (s NasServiceStatusService) Read(ctx context.Context) (*models.NasServiceResponseResult, error) {
	model := &models.NasServiceResponseResult{}
	model.Sambas = s.statusReader.ReadSambaShares()

	webdav := s.statusReader.ReadWebdavInfo()
	model.Webdav = &webdav

	linkease := &models.NasServiceLinkeaseInfo{}
	enabledByConfig, port, err := s.statusReader.ReadLinkeaseInfo(ctx)
	if err != nil {
		return nil, err
	}
	if enabledByConfig && s.runtimeReader.HasLinkeaseBinary() {
		linkease.Enabel = true
		linkease.Port = port
	}
	model.Linkease = linkease

	return model, nil
}

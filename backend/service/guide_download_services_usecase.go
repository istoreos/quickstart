package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	downloadservices "github.com/linkease/quick-start/istore-backend/modules/guidestorage/downloadservices"
)

type guideAria2InitFacade interface {
	InitAria2(ctx context.Context, input GuideAria2InitInput) (*models.SDKNormalResponse, error)
}

var newGuideAria2InitServiceFacade = func() guideAria2InitFacade {
	return newGuideAria2InitService()
}

type guideQbittorrentInitFacade interface {
	InitQbittorrent(ctx context.Context, input GuideQbittorrentInitInput) (*models.SDKNormalResponse, error)
}

var newGuideQbittorrentInitServiceFacade = func() guideQbittorrentInitFacade {
	return newGuideQbittorrentInitService()
}

type guideTransmissionInitFacade interface {
	InitTransmission(ctx context.Context, input GuideTransmissionInitInput) (*models.SDKNormalResponse, error)
}

var newGuideTransmissionInitServiceFacade = func() guideTransmissionInitFacade {
	return newGuideTransmissionInitService()
}

type guideDownloadServiceStatusFacade interface {
	Get(ctx context.Context) (*models.GuideDownloadServiceResponse, error)
}

var newGuideDownloadServiceStatusFacade = func() guideDownloadServiceStatusFacade {
	return newGuideDownloadServiceStatusService()
}

type GuideDownloadServiceStatusService struct {
	reader GuideDownloadServicesReader
}

func newGuideDownloadServiceStatusService() *GuideDownloadServiceStatusService {
	return &GuideDownloadServiceStatusService{
		reader: newDefaultGuideDownloadServicesReader(),
	}
}

type GuideAria2InitService struct {
	writer  GuideDownloadServicesWriter
	runtime GuideDownloadServicesRuntime
}

type GuideQbittorrentInitService struct {
	writer GuideDownloadServicesWriter
}

type GuideTransmissionInitService struct {
	writer GuideDownloadServicesWriter
}

func newGuideAria2InitService() *GuideAria2InitService {
	return &GuideAria2InitService{
		writer:  newDefaultGuideDownloadServicesWriter(),
		runtime: newDefaultGuideDownloadServicesRuntime(),
	}
}

func newGuideQbittorrentInitService() *GuideQbittorrentInitService {
	return &GuideQbittorrentInitService{
		writer: newDefaultGuideDownloadServicesWriter(),
	}
}

func newGuideTransmissionInitService() *GuideTransmissionInitService {
	return &GuideTransmissionInitService{
		writer: newDefaultGuideDownloadServicesWriter(),
	}
}

func (service *GuideDownloadServiceStatusService) Get(ctx context.Context) (*models.GuideDownloadServiceResponse, error) {
	aria2, err := service.reader.ReadAria2Status(ctx)
	if err != nil {
		return nil, err
	}
	qbit, err := service.reader.ReadQbittorrentStatus(ctx)
	if err != nil {
		return nil, err
	}
	transmission, err := service.reader.ReadTransmissionStatus(ctx)
	if err != nil {
		return nil, err
	}
	return buildGuideDownloadServiceStatusResult(aria2, qbit, transmission), nil
}

func (service *GuideAria2InitService) InitAria2(ctx context.Context, input GuideAria2InitInput) (*models.SDKNormalResponse, error) {
	return downloadservices.NewAria2InitService(service.writer, service.runtime).InitAria2(ctx, input)
}

func (service *GuideQbittorrentInitService) InitQbittorrent(ctx context.Context, input GuideQbittorrentInitInput) (*models.SDKNormalResponse, error) {
	return downloadservices.NewQbittorrentInitService(service.writer).InitQbittorrent(ctx, input)
}

func (service *GuideTransmissionInitService) InitTransmission(ctx context.Context, input GuideTransmissionInitInput) (*models.SDKNormalResponse, error) {
	return downloadservices.NewTransmissionInitService(service.writer).InitTransmission(ctx, input)
}

func buildGuideDownloadServiceStatusResult(
	aria2 *GuideDownloadAria2Snapshot,
	qbit *GuideDownloadQbittorrentSnapshot,
	transmission *GuideDownloadTransmissionSnapshot,
) *models.GuideDownloadServiceResponse {
	return downloadservices.BuildStatusResponse(aria2, qbit, transmission)
}

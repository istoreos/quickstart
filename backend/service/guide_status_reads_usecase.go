package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type GuideDdnstoConfigService struct {
	reader GuideStatusReadsReader
}

type GuideDDNSStatusService struct {
	reader GuideStatusReadsReader
}

type GuideDownloadPartitionListService struct {
	reader GuideStatusReadsReader
}

var newGuideDdnstoConfigServiceFacade = func() *GuideDdnstoConfigService {
	return newGuideDdnstoConfigService()
}

var newGuideDDNSStatusServiceFacade = func() *GuideDDNSStatusService {
	return newGuideDDNSStatusService()
}

var newGuideDownloadPartitionListServiceFacade = func() *GuideDownloadPartitionListService {
	return newGuideDownloadPartitionListService()
}

func newGuideDdnstoConfigService() *GuideDdnstoConfigService {
	return &GuideDdnstoConfigService{reader: newDefaultGuideStatusReadsReader()}
}

func newGuideDDNSStatusService() *GuideDDNSStatusService {
	return &GuideDDNSStatusService{reader: newDefaultGuideStatusReadsReader()}
}

func newGuideDownloadPartitionListService() *GuideDownloadPartitionListService {
	return &GuideDownloadPartitionListService{reader: newDefaultGuideStatusReadsReader()}
}

func (service *GuideDdnstoConfigService) Get(ctx context.Context) (*models.GuideDdnstoConfigResponseResult, error) {
	snapshot, err := service.reader.ReadDdnstoConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &models.GuideDdnstoConfigResponseResult{
		NetAddr:  "127.0.0.1",
		DeviceID: snapshot.DeviceID,
	}, nil
}

func (service *GuideDDNSStatusService) Get(ctx context.Context) (*models.GuideDdnsResponseResult, error) {
	snapshot, err := service.reader.ReadDDNSStatus(ctx)
	if err != nil {
		return nil, err
	}
	return &models.GuideDdnsResponseResult{
		DdnstoDomain: snapshot.DdnstoDomain,
		IPV4Domain:   snapshot.IPV4Domain,
		IPV6Domain:   snapshot.IPV6Domain,
	}, nil
}

func (service *GuideDownloadPartitionListService) Get(ctx context.Context) (*models.GuideDownloadPartitionListResponseResult, error) {
	partitions, err := service.reader.ReadDownloadPartitions(ctx)
	if err != nil {
		return nil, err
	}
	return &models.GuideDownloadPartitionListResponseResult{
		PartitionList: partitions,
	}, nil
}

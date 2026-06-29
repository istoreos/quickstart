package wanstats

import (
	"context"
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Sample struct {
	StartTime     time.Time
	EndTime       time.Time
	UploadSpeed   int64
	DownloadSpeed int64
}

type Sampler interface {
	Samples(ctx context.Context) ([]Sample, error)
}

type Service struct {
	sampler Sampler
	slots   int64
}

func NewService(sampler Sampler, slots int64) *Service {
	return &Service{
		sampler: sampler,
		slots:   slots,
	}
}

func (svc *Service) GetNetworkStatistic(ctx context.Context) (*models.NetworkStatisticsResponse, error) {
	samples, err := svc.sampler.Samples(ctx)
	if err != nil {
		return nil, err
	}
	return BuildResponse(samples, svc.slots), nil
}

func BuildResponse(samples []Sample, slots int64) *models.NetworkStatisticsResponse {
	items := make([]*models.NetworkStatisticsItem, 0, len(samples))
	for _, sample := range samples {
		items = append(items, &models.NetworkStatisticsItem{
			StartTime:     sample.StartTime.Unix(),
			EndTime:       sample.EndTime.Unix(),
			UploadSpeed:   sample.UploadSpeed,
			DownloadSpeed: sample.DownloadSpeed,
		})
	}
	return &models.NetworkStatisticsResponse{
		Result: &models.NetworkStatisticsResponseResult{
			Slots: slots,
			Items: items,
		},
	}
}

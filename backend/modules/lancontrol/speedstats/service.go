package speedstats

import (
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type Sample struct {
	StartTime     time.Time
	EndTime       time.Time
	UploadSpeed   int64
	DownloadSpeed int64
}

type Host struct {
	IP      string
	Samples []Sample
}

func BuildAllDeviceResponse(hosts []Host) *models.DeviceSpeedStatsResponse {
	resp := &models.DeviceSpeedStatsResponse{
		Result: make([]*models.DeviceSpeedStat, 0, len(hosts)+1),
	}
	for _, host := range hosts {
		if len(host.Samples) == 0 {
			continue
		}
		sample := host.Samples[0]
		resp.Result = append(resp.Result, &models.DeviceSpeedStat{
			IP:               host.IP,
			DownloadSpeed:    sample.DownloadSpeed,
			UploadSpeed:      sample.UploadSpeed,
			DownloadSpeedStr: utils.ByteCountDecimal(uint64(sample.DownloadSpeed)) + "/s",
			UploadSpeedStr:   utils.ByteCountDecimal(uint64(sample.UploadSpeed)) + "/s",
		})
	}
	return resp
}

func BuildHistoryResponse(samples []Sample, slots int64) *models.NetworkStatisticsResponse {
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

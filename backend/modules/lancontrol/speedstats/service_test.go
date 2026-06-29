package speedstats

import (
	"reflect"
	"testing"
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestBuildAllDeviceResponseUsesReportedSpeedSamplePerHost(t *testing.T) {
	t.Parallel()

	resp := BuildAllDeviceResponse([]Host{
		{
			IP: "192.168.1.2",
			Samples: []Sample{
				{UploadSpeed: 1536, DownloadSpeed: 2048},
			},
		},
		{IP: "192.168.1.3"},
		{
			IP: "192.168.1.4",
			Samples: []Sample{
				{UploadSpeed: 1, DownloadSpeed: 2},
				{UploadSpeed: 3000, DownloadSpeed: 4096},
			},
		},
	})

	want := &models.DeviceSpeedStatsResponse{
		Result: []*models.DeviceSpeedStat{
			{
				IP:               "192.168.1.2",
				DownloadSpeed:    2048,
				UploadSpeed:      1536,
				DownloadSpeedStr: "2.0 KB/s",
				UploadSpeedStr:   "1.5 KB/s",
			},
			{
				IP:               "192.168.1.4",
				DownloadSpeed:    2,
				UploadSpeed:      1,
				DownloadSpeedStr: "2 B/s",
				UploadSpeedStr:   "1 B/s",
			},
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Fatalf("response mismatch\nwant: %#v\n got: %#v", want, resp)
	}
}

func TestBuildHistoryResponseKeepsSlotsWithEmptyItems(t *testing.T) {
	t.Parallel()

	resp := BuildHistoryResponse(nil, 12)

	want := &models.NetworkStatisticsResponse{
		Result: &models.NetworkStatisticsResponseResult{
			Slots: 12,
			Items: []*models.NetworkStatisticsItem{},
		},
	}
	if !reflect.DeepEqual(resp, want) {
		t.Fatalf("response mismatch\nwant: %#v\n got: %#v", want, resp)
	}
}

func TestBuildHistoryResponseMapsSamples(t *testing.T) {
	t.Parallel()

	start := time.Unix(1710000100, 0)
	end := time.Unix(1710000103, 0)
	resp := BuildHistoryResponse([]Sample{
		{
			StartTime:     start,
			EndTime:       end,
			UploadSpeed:   10,
			DownloadSpeed: 20,
		},
	}, 8)

	wantItems := []*models.NetworkStatisticsItem{
		{StartTime: start.Unix(), EndTime: end.Unix(), UploadSpeed: 10, DownloadSpeed: 20},
	}
	if resp.Result == nil {
		t.Fatal("expected result")
	}
	if resp.Result.Slots != 8 {
		t.Fatalf("Slots = %d, want 8", resp.Result.Slots)
	}
	if !reflect.DeepEqual(resp.Result.Items, wantItems) {
		t.Fatalf("items mismatch\nwant: %#v\n got: %#v", wantItems, resp.Result.Items)
	}
}

package wanstats

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSampler struct {
	samples []Sample
}

func (sampler fakeSampler) Samples(ctx context.Context) ([]Sample, error) {
	return sampler.samples, nil
}

func TestServiceBuildsEmptyStatisticsResponse(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeSampler{}, 12)

	resp, err := svc.GetNetworkStatistic(context.Background())
	if err != nil {
		t.Fatalf("GetNetworkStatistic returned error: %v", err)
	}

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

func TestServiceMapsSamplesToStatisticsItems(t *testing.T) {
	t.Parallel()

	startA := time.Unix(1710000000, 0)
	endA := time.Unix(1710000005, 0)
	startB := time.Unix(1710000010, 0)
	endB := time.Unix(1710000015, 0)
	svc := NewService(fakeSampler{
		samples: []Sample{
			{
				StartTime:     startA,
				EndTime:       endA,
				UploadSpeed:   123,
				DownloadSpeed: 456,
			},
			{
				StartTime:     startB,
				EndTime:       endB,
				UploadSpeed:   789,
				DownloadSpeed: 1011,
			},
		},
	}, 8)

	resp, err := svc.GetNetworkStatistic(context.Background())
	if err != nil {
		t.Fatalf("GetNetworkStatistic returned error: %v", err)
	}

	wantItems := []*models.NetworkStatisticsItem{
		{StartTime: startA.Unix(), EndTime: endA.Unix(), UploadSpeed: 123, DownloadSpeed: 456},
		{StartTime: startB.Unix(), EndTime: endB.Unix(), UploadSpeed: 789, DownloadSpeed: 1011},
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

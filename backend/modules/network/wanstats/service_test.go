package wanstats

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
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

func TestStatisticsJSONIncludesZeroSpeeds(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name             string
		upload, download int64
	}{
		{name: "idle"},
		{name: "download only", download: 1234},
		{name: "upload only", upload: 5678},
		{name: "both directions", upload: 5678, download: 1234},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := BuildResponse([]Sample{{
				StartTime:     time.Unix(1710000000, 0),
				EndTime:       time.Unix(1710000005, 0),
				UploadSpeed:   tc.upload,
				DownloadSpeed: tc.download,
			}}, 12)
			body, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err)
			}
			var wire struct {
				Result struct {
					Items []map[string]int64 `json:"items"`
				} `json:"result"`
			}
			if err := json.Unmarshal(body, &wire); err != nil {
				t.Fatal(err)
			}
			for field, want := range map[string]int64{"uploadSpeed": tc.upload, "downloadSpeed": tc.download} {
				got, present := wire.Result.Items[0][field]
				if !present || got != want {
					t.Errorf("%s: got %d (present=%v), want %d; response=%s", field, got, present, want, body)
				}
			}
		})
	}
}

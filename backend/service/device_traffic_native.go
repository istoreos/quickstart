package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"
)

type nativeTrafficCounter struct {
	UploadBytes, DownloadBytes int64
	UploadSpeed, DownloadSpeed int64
}

type scopedDeviceTrafficSampler struct {
	legacy deviceTrafficSampler
	native *nativeRateLimitProvider
}

func (sampler scopedDeviceTrafficSampler) Sample(ctx context.Context) lanStatsSnapshot {
	if selectedLAN(ctx) == defaultLAN && scopedRateLimitProvider(ctx) != nativePolicyProviderName {
		return sampler.legacy.Sample(ctx)
	}
	provider, err := sampler.native.forContext(ctx)
	if err != nil {
		return nativeTrafficFailure(err)
	}
	if err := provider.probe(ctx); err != nil {
		return nativeTrafficFailure(err)
	}
	if provider.offload != nil {
		state, err := provider.offload.Offload(ctx)
		if err != nil || state != "compatible" {
			return lanStatsSnapshot{err: errors.New("traffic_offload_unverified"), reasons: []string{"traffic_offload_unverified"}}
		}
	}
	var health nativeEngineHealth
	if err := provider.call(ctx, http.MethodGet, "/api/v1/system/health", nil, &health); err != nil {
		return nativeTrafficFailure(err)
	}
	if health.State != "healthy" {
		return nativeTrafficFailure(errors.New("traffic_engine_unhealthy"))
	}
	if health.Context != nil && provider.binding != nil {
		b := provider.binding
		if health.Interface != b.Interface || health.Context.NetworkID != b.NetworkID || health.Context.InstanceID != b.InstanceID || health.Context.InterfaceIndex != b.InterfaceIndex || health.Context.Epoch == "" {
			return nativeTrafficFailure(errors.New("traffic_runtime_binding_mismatch"))
		}
	}
	if health.Context == nil {
		// Original main engines lack sample timestamps/epochs. Retain their
		// existing conntrack accounting instead of guessing a native baseline.
		if selectedLAN(ctx) == defaultLAN {
			return sampler.legacy.Sample(ctx)
		}
		return nativeTrafficFailure(errors.New("traffic_scope_unsupported"))
	}
	var response struct {
		SampledAtMS  int64  `json:"sampledAtMs"`
		CounterEpoch string `json:"counterEpoch"`
		Devices      []struct {
			MAC          string `json:"mac"`
			Upload       uint64 `json:"w_tx_b"`
			Download     uint64 `json:"w_rx_b"`
			UploadRate   uint64 `json:"w_tx_r"`
			DownloadRate uint64 `json:"w_rx_r"`
		} `json:"d"`
	}
	if err := provider.call(ctx, http.MethodGet, "/api/traffic/devices", nil, &response); err != nil {
		return nativeTrafficFailure(err)
	}
	if response.CounterEpoch == "" || response.CounterEpoch != health.Context.Epoch {
		return nativeTrafficFailure(errors.New("traffic_runtime_changed"))
	}
	if response.SampledAtMS == 0 {
		return lanStatsSnapshot{native: map[string]nativeTrafficCounter{}, epoch: response.CounterEpoch}
	}
	if response.SampledAtMS > time.Now().Add(2*time.Second).UnixMilli() {
		return nativeTrafficFailure(errors.New("traffic_sample_clock_invalid"))
	}
	result := lanStatsSnapshot{native: map[string]nativeTrafficCounter{}, epoch: health.Context.Epoch, sampledAt: time.UnixMilli(response.SampledAtMS), samples: 2}
	for _, device := range response.Devices {
		mac := normalizeInventoryMAC(device.MAC)
		if mac == "" {
			continue
		}
		if device.Upload > math.MaxInt64 || device.Download > math.MaxInt64 || device.UploadRate > math.MaxInt64 || device.DownloadRate > math.MaxInt64 {
			return nativeTrafficFailure(fmt.Errorf("traffic_counter_out_of_range"))
		}
		result.native[mac] = nativeTrafficCounter{UploadBytes: int64(device.Upload), DownloadBytes: int64(device.Download), UploadSpeed: int64(device.UploadRate), DownloadSpeed: int64(device.DownloadRate)}
	}
	return result
}

func nativeTrafficFailure(err error) lanStatsSnapshot {
	return lanStatsSnapshot{err: err, reasons: []string{"traffic_engine_unavailable"}}
}

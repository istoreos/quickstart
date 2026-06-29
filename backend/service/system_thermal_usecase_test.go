package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeThermalGetter struct {
	temp int
	err  error
}

func (getter fakeThermalGetter) CPUTemperature() (int, error) {
	return getter.temp, getter.err
}

func TestGetSystemCpuTemperatureReturnsTemperature(t *testing.T) {
	backend := &ServiceBackend{thermalZone: fakeThermalGetter{temp: 47}}

	resp, err := backend.GetSystemCpuTemperature(context.Background())
	if err != nil {
		t.Fatalf("GetSystemCpuTemperature returned error: %v", err)
	}
	if resp.Result == nil || resp.Result.Temperature != 47 {
		t.Fatalf("Result = %#v", resp.Result)
	}
}

func TestGetSystemCpuTemperatureFallsBackToZeroOnError(t *testing.T) {
	backend := &ServiceBackend{thermalZone: fakeThermalGetter{err: errors.New("read failed")}}

	resp, err := backend.GetSystemCpuTemperature(context.Background())
	if err != nil {
		t.Fatalf("GetSystemCpuTemperature returned error: %v", err)
	}
	if resp.Result == nil || resp.Result.Temperature != 0 {
		t.Fatalf("Result = %#v", resp.Result)
	}
}

func TestGetSystemStatusAttachesTemperature(t *testing.T) {
	original := newSystemRuntimeService
	defer func() { newSystemRuntimeService = original }()

	newSystemRuntimeService = func() systemRuntimeFacade {
		return fakeSystemRuntimeFacade{
			statusResult: &models.SystemStatusResponseResult{
				CPUUsage: 12,
				MemTotal: "512MB",
			},
		}
	}
	backend := &ServiceBackend{thermalZone: fakeThermalGetter{temp: 51}}

	resp, err := backend.GetSystemStatus(context.Background())
	if err != nil {
		t.Fatalf("GetSystemStatus returned error: %v", err)
	}
	if resp.Result == nil || resp.Result.CPUTemperature != 51 {
		t.Fatalf("Result = %#v", resp.Result)
	}
}

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSystemRuntimeFacade struct {
	timeResult   *models.SystemTimeResponseResult
	timeErr      error
	cpuResult    *models.SystemCPUStatusResponseResult
	cpuErr       error
	memoryResult *models.SystemMemeryStatusResponseResult
	memoryErr    error
	statusResult *models.SystemStatusResponseResult
	statusErr    error
}

func (svc fakeSystemRuntimeFacade) Time(ctx context.Context) (*models.SystemTimeResponseResult, error) {
	return svc.timeResult, svc.timeErr
}

func (svc fakeSystemRuntimeFacade) CPU(ctx context.Context) (*models.SystemCPUStatusResponseResult, error) {
	return svc.cpuResult, svc.cpuErr
}

func (svc fakeSystemRuntimeFacade) Memory(ctx context.Context) (*models.SystemMemeryStatusResponseResult, error) {
	return svc.memoryResult, svc.memoryErr
}

func (svc fakeSystemRuntimeFacade) Status(ctx context.Context) (*models.SystemStatusResponseResult, error) {
	return svc.statusResult, svc.statusErr
}

func TestSystemTimeDelegatesToRuntimeFacade(t *testing.T) {
	original := newSystemRuntimeService
	defer func() { newSystemRuntimeService = original }()

	newSystemRuntimeService = func() systemRuntimeFacade {
		return fakeSystemRuntimeFacade{timeResult: &models.SystemTimeResponseResult{Localtime: "2024-01-01 00:00:00"}}
	}

	resp, err := SystemTime(context.Background())
	if err != nil {
		t.Fatalf("SystemTime returned error: %v", err)
	}
	if resp.Result == nil || resp.Result.Localtime != "2024-01-01 00:00:00" {
		t.Fatalf("Result = %#v", resp.Result)
	}
}

func TestSystemCpuStatusDelegatesToRuntimeFacade(t *testing.T) {
	original := newSystemRuntimeService
	defer func() { newSystemRuntimeService = original }()

	newSystemRuntimeService = func() systemRuntimeFacade {
		return fakeSystemRuntimeFacade{cpuResult: &models.SystemCPUStatusResponseResult{Usage: 42}}
	}

	resp, err := SystemCpuStatus(context.Background())
	if err != nil {
		t.Fatalf("SystemCpuStatus returned error: %v", err)
	}
	if resp.Result == nil || resp.Result.Usage != 42 {
		t.Fatalf("Result = %#v", resp.Result)
	}
}

func TestSystemMemeryStatusDelegatesToRuntimeFacade(t *testing.T) {
	original := newSystemRuntimeService
	defer func() { newSystemRuntimeService = original }()

	newSystemRuntimeService = func() systemRuntimeFacade {
		return fakeSystemRuntimeFacade{memoryResult: &models.SystemMemeryStatusResponseResult{Available: "128MB"}}
	}

	resp, err := SystemMemeryStatus(context.Background())
	if err != nil {
		t.Fatalf("SystemMemeryStatus returned error: %v", err)
	}
	if resp.Result == nil || resp.Result.Available != "128MB" {
		t.Fatalf("Result = %#v", resp.Result)
	}
}

func TestSystemStatusDelegatesToRuntimeFacade(t *testing.T) {
	original := newSystemRuntimeService
	defer func() { newSystemRuntimeService = original }()

	newSystemRuntimeService = func() systemRuntimeFacade {
		return fakeSystemRuntimeFacade{statusResult: &models.SystemStatusResponseResult{CPUUsage: 55}}
	}

	resp, err := SystemStatus(context.Background())
	if err != nil {
		t.Fatalf("SystemStatus returned error: %v", err)
	}
	if resp.Result == nil || resp.Result.CPUUsage != 55 {
		t.Fatalf("Result = %#v", resp.Result)
	}
}

func TestSystemRuntimeWrappersPropagateFacadeErrors(t *testing.T) {
	original := newSystemRuntimeService
	defer func() { newSystemRuntimeService = original }()

	expectedErr := errors.New("runtime failed")
	newSystemRuntimeService = func() systemRuntimeFacade {
		return fakeSystemRuntimeFacade{
			timeErr:   expectedErr,
			cpuErr:    expectedErr,
			memoryErr: expectedErr,
			statusErr: expectedErr,
		}
	}

	if _, err := SystemTime(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemTime error = %v, want expectedErr", err)
	}
	if _, err := SystemCpuStatus(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemCpuStatus error = %v, want expectedErr", err)
	}
	if _, err := SystemMemeryStatus(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemMemeryStatus error = %v, want expectedErr", err)
	}
	if _, err := SystemStatus(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemStatus error = %v, want expectedErr", err)
	}
}

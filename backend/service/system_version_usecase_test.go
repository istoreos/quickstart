package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSystemVersionFacade struct {
	result *models.SystemVersionResponseResult
	err    error
}

func (svc fakeSystemVersionFacade) Get(ctx context.Context) (*models.SystemVersionResponseResult, error) {
	return svc.result, svc.err
}

func TestSystemVersionDelegatesToFacade(t *testing.T) {
	original := newSystemVersionService
	defer func() { newSystemVersionService = original }()

	newSystemVersionService = func() systemVersionFacade {
		return fakeSystemVersionFacade{
			result: &models.SystemVersionResponseResult{
				Model:           "x86 Generic",
				FirmwareVersion: "iStoreOS",
				KernelVersion:   "6.1",
				Quickstart:      "dev",
			},
		}
	}

	resp, err := SystemVersion(context.Background())
	if err != nil {
		t.Fatalf("SystemVersion returned error: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("Result = nil")
	}
	if resp.Result.Model != "x86 Generic" {
		t.Fatalf("Model = %q", resp.Result.Model)
	}
	if resp.Result.FirmwareVersion != "iStoreOS" {
		t.Fatalf("FirmwareVersion = %q", resp.Result.FirmwareVersion)
	}
	if resp.Result.KernelVersion != "6.1" {
		t.Fatalf("KernelVersion = %q", resp.Result.KernelVersion)
	}
	if resp.Result.Quickstart != "dev" {
		t.Fatalf("Quickstart = %q", resp.Result.Quickstart)
	}
}

func TestSystemVersionPropagatesFacadeError(t *testing.T) {
	original := newSystemVersionService
	defer func() { newSystemVersionService = original }()

	expectedErr := errors.New("version failed")
	newSystemVersionService = func() systemVersionFacade {
		return fakeSystemVersionFacade{err: expectedErr}
	}

	if _, err := SystemVersion(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemVersion error = %v, want expectedErr", err)
	}
}

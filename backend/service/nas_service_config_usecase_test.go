package service

import (
	"context"
	"errors"
	"testing"

	"github.com/istoreos/quickstart/backend/models"
)

type fakeNasServiceStatusFacade struct {
	result *models.NasServiceResponseResult
	err    error
	calls  int
}

func (f *fakeNasServiceStatusFacade) Read(ctx context.Context) (*models.NasServiceResponseResult, error) {
	f.calls++
	return f.result, f.err
}

func TestNasServiceStatusCompatibilityDelegatesToService(t *testing.T) {
	originalFactory := newNasServiceStatusServiceFacade
	defer func() {
		newNasServiceStatusServiceFacade = originalFactory
	}()

	facade := &fakeNasServiceStatusFacade{
		result: &models.NasServiceResponseResult{
			Sambas: []*models.NasServiceSambaInfo{{ShareName: "share"}},
			Webdav: &models.NasServiceWebdavInfo{Port: "5244"},
			Linkease: &models.NasServiceLinkeaseInfo{
				Enabel: true,
				Port:   "8897",
			},
		},
	}
	newNasServiceStatusServiceFacade = func() nasServiceStatusFacade {
		return facade
	}

	resp, err := NasServiceStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || len(resp.Result.Sambas) != 1 || resp.Result.Sambas[0].ShareName != "share" {
		t.Fatalf("unexpected wrapper response: %#v", resp)
	}
	if resp.Result.Webdav == nil || resp.Result.Webdav.Port != "5244" {
		t.Fatalf("unexpected webdav response: %#v", resp.Result.Webdav)
	}
	if resp.Result.Linkease == nil || !resp.Result.Linkease.Enabel || resp.Result.Linkease.Port != "8897" {
		t.Fatalf("unexpected linkease response: %#v", resp.Result.Linkease)
	}
	if facade.calls != 1 {
		t.Fatalf("expected one service call, got %d", facade.calls)
	}
}

func TestNasServiceStatusCompatibilityPropagatesServiceError(t *testing.T) {
	originalFactory := newNasServiceStatusServiceFacade
	defer func() {
		newNasServiceStatusServiceFacade = originalFactory
	}()

	serviceErr := errors.New("nas service status failed")
	newNasServiceStatusServiceFacade = func() nasServiceStatusFacade {
		return &fakeNasServiceStatusFacade{err: serviceErr}
	}

	if _, err := NasServiceStatus(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected wrapper to propagate service error, got %v", err)
	}
}

type fakeNasStatusReader struct {
	sambaShares  []*models.NasServiceSambaInfo
	webdavInfo   models.NasServiceWebdavInfo
	linkeaseOn   bool
	linkeasePort string
	linkeaseErr  error
}

func (f fakeNasStatusReader) ReadSambaShares() []*models.NasServiceSambaInfo {
	return f.sambaShares
}

func (f fakeNasStatusReader) ReadWebdavPort() (string, bool) {
	return f.webdavInfo.Port, f.webdavInfo.Port != ""
}

func (f fakeNasStatusReader) ReadWebdavInfo() models.NasServiceWebdavInfo {
	return f.webdavInfo
}

func (f fakeNasStatusReader) ReadLinkeaseInfo(ctx context.Context) (bool, string, error) {
	return f.linkeaseOn, f.linkeasePort, f.linkeaseErr
}

type fakeNasRuntimeReader struct {
	linkeaseBinary bool
}

func (f fakeNasRuntimeReader) ReadLANIPv4(ctx context.Context) (string, error) {
	return "", nil
}

func (f fakeNasRuntimeReader) HasLinkeaseBinary() bool {
	return f.linkeaseBinary
}

func TestNasServiceStatusServiceAggregatesSambaWebdavAndLinkease(t *testing.T) {
	service := NewNasServiceStatusService(
		fakeNasStatusReader{
			sambaShares: []*models.NasServiceSambaInfo{{ShareName: "share", Path: "/mnt/data"}},
			webdavInfo: models.NasServiceWebdavInfo{
				Path: "/mnt/data",
				Port: "6086",
			},
			linkeaseOn:   true,
			linkeasePort: "8897",
		},
		fakeNasRuntimeReader{linkeaseBinary: true},
	)

	result, err := service.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected status read error: %v", err)
	}
	if len(result.Sambas) != 1 || result.Sambas[0].ShareName != "share" || result.Sambas[0].Path != "/mnt/data" {
		t.Fatalf("unexpected samba status: %#v", result.Sambas)
	}
	if result.Webdav == nil || result.Webdav.Path != "/mnt/data" || result.Webdav.Port != "6086" {
		t.Fatalf("unexpected webdav status: %#v", result.Webdav)
	}
	if result.Linkease == nil || !result.Linkease.Enabel || result.Linkease.Port != "8897" {
		t.Fatalf("unexpected linkease status: %#v", result.Linkease)
	}
}

func TestNasServiceStatusServiceKeepsLinkeaseDisabledWithoutBinary(t *testing.T) {
	service := NewNasServiceStatusService(
		fakeNasStatusReader{
			linkeaseOn:   true,
			linkeasePort: "8897",
		},
		fakeNasRuntimeReader{linkeaseBinary: false},
	)

	result, err := service.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected status read error: %v", err)
	}
	if result.Linkease == nil {
		t.Fatalf("expected linkease status")
	}
	if result.Linkease.Enabel || result.Linkease.Port != "" {
		t.Fatalf("expected LinkEase disabled without binary, got %#v", result.Linkease)
	}
}

func TestNasServiceStatusServicePropagatesReaderErrors(t *testing.T) {
	expectedErr := errors.New("linkease read failed")
	service := NewNasServiceStatusService(fakeNasStatusReader{linkeaseErr: expectedErr}, fakeNasRuntimeReader{})

	if _, err := service.Read(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected reader error, got %v", err)
	}
}

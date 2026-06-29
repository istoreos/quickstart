package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeNasSambaCreateFacade struct {
	result *models.NasSambaCreateResponseResult
	err    error
	inputs []NasSambaCreateInput
}

func (f *fakeNasSambaCreateFacade) Create(ctx context.Context, input NasSambaCreateInput) (*models.NasSambaCreateResponseResult, error) {
	f.inputs = append(f.inputs, input)
	return f.result, f.err
}

type fakeNasWebdavCreateFacade struct {
	result *models.NasWebdavCreateResponseResult
	err    error
	inputs []NasWebdavCreateInput
}

func (f *fakeNasWebdavCreateFacade) Create(ctx context.Context, input NasWebdavCreateInput) (*models.NasWebdavCreateResponseResult, error) {
	f.inputs = append(f.inputs, input)
	return f.result, f.err
}

type fakeNasWebdavStatusFacade struct {
	result *models.NasWebdavStatusResponseResult
	err    error
	calls  int
}

func (f *fakeNasWebdavStatusFacade) Read(ctx context.Context) (*models.NasWebdavStatusResponseResult, error) {
	f.calls++
	return f.result, f.err
}

type fakeNasServiceStatusFacade struct {
	result *models.NasServiceResponseResult
	err    error
	calls  int
}

func (f *fakeNasServiceStatusFacade) Read(ctx context.Context) (*models.NasServiceResponseResult, error) {
	f.calls++
	return f.result, f.err
}

func TestNasServiceSambaCreateCompatibilityDelegatesToService(t *testing.T) {
	originalFactory := newNasSambaCreateServiceFacade
	defer func() {
		newNasSambaCreateServiceFacade = originalFactory
	}()

	facade := &fakeNasSambaCreateFacade{
		result: &models.NasSambaCreateResponseResult{SambaURL: "smb://192.168.100.1/share"},
	}
	newNasSambaCreateServiceFacade = func() nasSambaCreateFacade {
		return facade
	}

	req := httptest.NewRequest("POST", "/nas/samba", strings.NewReader(`{"shareName":"share","rootPath":"/mnt/data","username":"user","password":"pw","allowLegacy":true}`))
	resp, err := NasServiceSambaCreate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.SambaURL != "smb://192.168.100.1/share" {
		t.Fatalf("unexpected wrapper response: %#v", resp)
	}
	if len(facade.inputs) != 1 {
		t.Fatalf("expected one service call, got %d", len(facade.inputs))
	}
	if facade.inputs[0] != (NasSambaCreateInput{
		ShareName:   "share",
		RootPath:    "/mnt/data",
		Username:    "user",
		Password:    "pw",
		AllowLegacy: true,
	}) {
		t.Fatalf("unexpected service input: %#v", facade.inputs[0])
	}
}

func TestNasServiceSambaCreateCompatibilityPropagatesServiceError(t *testing.T) {
	originalFactory := newNasSambaCreateServiceFacade
	defer func() {
		newNasSambaCreateServiceFacade = originalFactory
	}()

	serviceErr := errors.New("samba create failed")
	newNasSambaCreateServiceFacade = func() nasSambaCreateFacade {
		return &fakeNasSambaCreateFacade{err: serviceErr}
	}

	req := httptest.NewRequest("POST", "/nas/samba", strings.NewReader(`{"shareName":"share","rootPath":"/mnt/data","username":"user","password":"pw"}`))
	if _, err := NasServiceSambaCreate(context.Background(), req); !errors.Is(err, serviceErr) {
		t.Fatalf("expected wrapper to propagate service error, got %v", err)
	}
}

func TestNasServiceWebdavCreateCompatibilityDelegatesToService(t *testing.T) {
	originalFactory := newNasWebdavCreateServiceFacade
	defer func() {
		newNasWebdavCreateServiceFacade = originalFactory
	}()

	facade := &fakeNasWebdavCreateFacade{
		result: &models.NasWebdavCreateResponseResult{
			Username:  "user",
			WebdavURL: "http://192.168.100.1:5244",
		},
	}
	newNasWebdavCreateServiceFacade = func() nasWebdavCreateFacade {
		return facade
	}

	req := httptest.NewRequest("POST", "/nas/webdav", strings.NewReader(`{"rootPath":"/mnt/data","username":"user","password":"pw"}`))
	resp, err := NasServiceWebdavCreate(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.Username != "user" || resp.Result.WebdavURL != "http://192.168.100.1:5244" {
		t.Fatalf("unexpected wrapper response: %#v", resp)
	}
	if len(facade.inputs) != 1 {
		t.Fatalf("expected one service call, got %d", len(facade.inputs))
	}
	if facade.inputs[0] != (NasWebdavCreateInput{
		RootPath: "/mnt/data",
		Username: "user",
		Password: "pw",
	}) {
		t.Fatalf("unexpected service input: %#v", facade.inputs[0])
	}
}

func TestNasServiceWebdavCreateCompatibilityPropagatesServiceError(t *testing.T) {
	originalFactory := newNasWebdavCreateServiceFacade
	defer func() {
		newNasWebdavCreateServiceFacade = originalFactory
	}()

	serviceErr := errors.New("webdav create failed")
	newNasWebdavCreateServiceFacade = func() nasWebdavCreateFacade {
		return &fakeNasWebdavCreateFacade{err: serviceErr}
	}

	req := httptest.NewRequest("POST", "/nas/webdav", strings.NewReader(`{"rootPath":"/mnt/data","username":"user","password":"pw"}`))
	if _, err := NasServiceWebdavCreate(context.Background(), req); !errors.Is(err, serviceErr) {
		t.Fatalf("expected wrapper to propagate service error, got %v", err)
	}
}

func TestNasServiceWebdavStatusCompatibilityDelegatesToService(t *testing.T) {
	originalFactory := newNasWebdavStatusServiceFacade
	defer func() {
		newNasWebdavStatusServiceFacade = originalFactory
	}()

	facade := &fakeNasWebdavStatusFacade{
		result: &models.NasWebdavStatusResponseResult{
			Path:     "/mnt/data",
			Port:     "5244",
			Username: "user",
			Password: "pw",
		},
	}
	newNasWebdavStatusServiceFacade = func() nasWebdavStatusFacade {
		return facade
	}

	resp, err := NasServiceWebdavStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if resp == nil || resp.Result == nil || resp.Result.Path != "/mnt/data" || resp.Result.Port != "5244" || resp.Result.Username != "user" || resp.Result.Password != "pw" {
		t.Fatalf("unexpected wrapper response: %#v", resp)
	}
	if facade.calls != 1 {
		t.Fatalf("expected one service call, got %d", facade.calls)
	}
}

func TestNasServiceWebdavStatusCompatibilityPropagatesServiceError(t *testing.T) {
	originalFactory := newNasWebdavStatusServiceFacade
	defer func() {
		newNasWebdavStatusServiceFacade = originalFactory
	}()

	serviceErr := errors.New("webdav status failed")
	newNasWebdavStatusServiceFacade = func() nasWebdavStatusFacade {
		return &fakeNasWebdavStatusFacade{err: serviceErr}
	}

	if _, err := NasServiceWebdavStatus(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected wrapper to propagate service error, got %v", err)
	}
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

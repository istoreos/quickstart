package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeGuideDockerRuntimeReader struct {
	snapshot *GuideDockerRuntimeSnapshot
	err      error
}

func (reader *fakeGuideDockerRuntimeReader) ReadDockerRuntime(ctx context.Context) (*GuideDockerRuntimeSnapshot, error) {
	return reader.snapshot, reader.err
}

type fakeGuideDockerRuntimeWriter struct {
	startCalls int
	stopCalls  int
	startErr   error
	stopErr    error
}

type fakeGuideDockerRuntimeFacade struct {
	statusResp  *models.GuideDockerStatusResponse
	statusErr   error
	statusCalls int

	switchResp  *models.SDKNormalResponse
	switchErr   error
	switchCalls []bool
}

func (facade *fakeGuideDockerRuntimeFacade) GetStatus(ctx context.Context) (*models.GuideDockerStatusResponse, error) {
	facade.statusCalls++
	return facade.statusResp, facade.statusErr
}

func (facade *fakeGuideDockerRuntimeFacade) Switch(ctx context.Context, enable bool) (*models.SDKNormalResponse, error) {
	facade.switchCalls = append(facade.switchCalls, enable)
	return facade.switchResp, facade.switchErr
}

func (writer *fakeGuideDockerRuntimeWriter) Start(ctx context.Context) error {
	writer.startCalls++
	return writer.startErr
}

func (writer *fakeGuideDockerRuntimeWriter) Stop(ctx context.Context) error {
	writer.stopCalls++
	return writer.stopErr
}

func TestGuideDockerRuntimeServiceGetStatusBuildsLegacyResponse(t *testing.T) {
	service := GuideDockerRuntimeService{
		reader: &fakeGuideDockerRuntimeReader{
			snapshot: &GuideDockerRuntimeSnapshot{
				Installed: true,
				Running:   true,
				Path:      "/mnt/docker",
				ErrorInfo: "warn",
			},
		},
		writer: &fakeGuideDockerRuntimeWriter{},
	}

	resp, err := service.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected docker status error: %v", err)
	}
	if resp == nil || resp.Result == nil {
		t.Fatalf("expected docker status result, got %#v", resp)
	}
	if resp.Result.Status != "running" || resp.Result.Path != "/mnt/docker" || resp.Result.ErrorInfo != "warn" {
		t.Fatalf("unexpected docker status result: %#v", resp.Result)
	}

	service.reader = &fakeGuideDockerRuntimeReader{
		snapshot: &GuideDockerRuntimeSnapshot{Installed: false},
	}
	resp, err = service.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected not-installed docker status error: %v", err)
	}
	if resp.Result.Status != "not installed" {
		t.Fatalf("expected not installed status, got %#v", resp.Result)
	}
}

func TestGuideDockerRuntimeServiceGetStatusPropagatesReaderError(t *testing.T) {
	readerErr := errors.New("reader failed")
	service := GuideDockerRuntimeService{
		reader: &fakeGuideDockerRuntimeReader{err: readerErr},
		writer: &fakeGuideDockerRuntimeWriter{},
	}

	if _, err := service.GetStatus(context.Background()); !errors.Is(err, readerErr) {
		t.Fatalf("expected reader error, got %v", err)
	}
}

func TestGuideDockerRuntimeServiceSwitchPreservesLegacyErrorWording(t *testing.T) {
	writer := &fakeGuideDockerRuntimeWriter{}
	service := GuideDockerRuntimeService{
		reader: &fakeGuideDockerRuntimeReader{},
		writer: writer,
	}

	resp, err := service.Switch(context.Background(), true)
	if err != nil {
		t.Fatalf("unexpected docker start success error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected docker switch success response: %#v", resp)
	}
	if writer.startCalls != 1 || writer.stopCalls != 0 {
		t.Fatalf("unexpected start/stop calls after enable: start=%d stop=%d", writer.startCalls, writer.stopCalls)
	}

	resp, err = service.Switch(context.Background(), false)
	if err != nil {
		t.Fatalf("unexpected docker stop success error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected docker switch stop response: %#v", resp)
	}
	if writer.startCalls != 1 || writer.stopCalls != 1 {
		t.Fatalf("unexpected start/stop calls after disable: start=%d stop=%d", writer.startCalls, writer.stopCalls)
	}

	writer.startErr = errors.New("start failed")
	if _, err := service.Switch(context.Background(), true); err == nil || err.Error() != "docker启动失败" {
		t.Fatalf("unexpected docker start error: %v", err)
	}

	writer.startErr = nil
	writer.stopErr = errors.New("stop failed")
	if _, err := service.Switch(context.Background(), false); err == nil || err.Error() != "docker停止失败" {
		t.Fatalf("unexpected docker stop error: %v", err)
	}
}

func TestServiceBackendGetGuideDockerStatusCompatibility(t *testing.T) {
	orig := newGuideDockerRuntimeFacade
	defer func() { newGuideDockerRuntimeFacade = orig }()

	facade := &fakeGuideDockerRuntimeFacade{
		statusResp: &models.GuideDockerStatusResponse{
			Result: &models.GuideDockerStatusResponseResult{
				Status:    "running",
				Path:      "/mnt/docker",
				ErrorInfo: "warn",
			},
		},
	}
	newGuideDockerRuntimeFacade = func() guideDockerRuntimeFacade { return facade }

	resp, err := (&ServiceBackend{}).GetGuideDockerStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected GetGuideDockerStatus error: %v", err)
	}
	if facade.statusCalls != 1 {
		t.Fatalf("expected one facade status call, got %d", facade.statusCalls)
	}
	if !reflect.DeepEqual(resp, facade.statusResp) {
		t.Fatalf("expected passthrough response, got %#v", resp)
	}
}

func TestServiceBackendGetGuideDockerStatusCompatibilityPropagatesErrors(t *testing.T) {
	orig := newGuideDockerRuntimeFacade
	defer func() { newGuideDockerRuntimeFacade = orig }()

	serviceErr := errors.New("docker status failed")
	newGuideDockerRuntimeFacade = func() guideDockerRuntimeFacade {
		return &fakeGuideDockerRuntimeFacade{statusErr: serviceErr}
	}

	if _, err := (&ServiceBackend{}).GetGuideDockerStatus(context.Background()); !errors.Is(err, serviceErr) {
		t.Fatalf("expected GetGuideDockerStatus error, got %v", err)
	}
}

func TestServiceBackendPostGuideDockerSwitchCompatibility(t *testing.T) {
	orig := newGuideDockerRuntimeFacade
	defer func() { newGuideDockerRuntimeFacade = orig }()

	success := models.ResponseSuccess(0)
	facade := &fakeGuideDockerRuntimeFacade{
		switchResp: &models.SDKNormalResponse{Success: &success},
	}
	newGuideDockerRuntimeFacade = func() guideDockerRuntimeFacade { return facade }

	enableReq := strings.NewReader(`{"enable":true}`)
	resp, err := (&ServiceBackend{}).PostGuideDockerSwitch(context.Background(), httptest.NewRequest("POST", "/guide/docker-switch", enableReq))
	if err != nil {
		t.Fatalf("unexpected PostGuideDockerSwitch enable error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected GuideDockerSwitch enable response: %#v", resp)
	}

	disableReq := strings.NewReader(`{"enable":false}`)
	resp, err = (&ServiceBackend{}).PostGuideDockerSwitch(context.Background(), httptest.NewRequest("POST", "/guide/docker-switch", disableReq))
	if err != nil {
		t.Fatalf("unexpected PostGuideDockerSwitch disable error: %v", err)
	}
	if resp == nil || resp.Success == nil || *resp.Success != 0 {
		t.Fatalf("unexpected GuideDockerSwitch disable response: %#v", resp)
	}
	if !reflect.DeepEqual(facade.switchCalls, []bool{true, false}) {
		t.Fatalf("unexpected switch delegation calls: %v", facade.switchCalls)
	}
}

func TestServiceBackendPostGuideDockerSwitchCompatibilityPropagatesErrors(t *testing.T) {
	orig := newGuideDockerRuntimeFacade
	defer func() { newGuideDockerRuntimeFacade = orig }()

	serviceErr := errors.New("switch failed")
	newGuideDockerRuntimeFacade = func() guideDockerRuntimeFacade {
		return &fakeGuideDockerRuntimeFacade{switchErr: serviceErr}
	}

	req := httptest.NewRequest("POST", "/guide/docker-switch", strings.NewReader(`{"enable":true}`))
	if _, err := (&ServiceBackend{}).PostGuideDockerSwitch(context.Background(), req); !errors.Is(err, serviceErr) {
		t.Fatalf("expected PostGuideDockerSwitch error, got %v", err)
	}
}

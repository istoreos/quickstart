package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func sandboxTestRequest(body string) *http.Request {
	req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	return req
}

type fakeNasSandboxFacade struct {
	listDisksResult []*models.NasDiskInfo
	listDisksErr    error
	statusResult    string
	statusErr       error
	formatPath      string
	formatErr       error
	commitCalls     int
	commitErr       error
	resetCalls      int
	resetErr        error
	exitCalls       int
	exitErr         error
}

func (svc *fakeNasSandboxFacade) ListDisks(ctx context.Context) ([]*models.NasDiskInfo, error) {
	return svc.listDisksResult, svc.listDisksErr
}

func (svc *fakeNasSandboxFacade) Status(ctx context.Context) (string, error) {
	return svc.statusResult, svc.statusErr
}

func (svc *fakeNasSandboxFacade) FormatPartition(ctx context.Context, path string) error {
	svc.formatPath = path
	return svc.formatErr
}

func (svc *fakeNasSandboxFacade) Commit(ctx context.Context) error {
	svc.commitCalls++
	return svc.commitErr
}

func (svc *fakeNasSandboxFacade) Reset(ctx context.Context) error {
	svc.resetCalls++
	return svc.resetErr
}

func (svc *fakeNasSandboxFacade) Exit(ctx context.Context) error {
	svc.exitCalls++
	return svc.exitErr
}

func TestNasSandboxCompatibilityDelegatesDisksAndStatus(t *testing.T) {
	original := newNasSandboxService
	defer func() { newNasSandboxService = original }()

	facade := &fakeNasSandboxFacade{
		listDisksResult: []*models.NasDiskInfo{{Name: "usb0"}},
		statusResult:    "running",
	}
	newNasSandboxService = func() nasSandboxFacade {
		return facade
	}

	disksResp, err := NasSanboxDisks(context.Background())
	if err != nil {
		t.Fatalf("unexpected disks error: %v", err)
	}
	if disksResp == nil || disksResp.Result == nil || len(disksResp.Result.Disks) != 1 || disksResp.Result.Disks[0].Name != "usb0" {
		t.Fatalf("unexpected disks response: %#v", disksResp)
	}

	statusResp, err := NasSanboxStatus(context.Background())
	if err != nil {
		t.Fatalf("unexpected status error: %v", err)
	}
	if statusResp == nil || statusResp.Result == nil || statusResp.Result.Status != "running" {
		t.Fatalf("unexpected status response: %#v", statusResp)
	}
}

func TestNasSandboxCompatibilityDelegatesActionsAndFormat(t *testing.T) {
	original := newNasSandboxService
	defer func() { newNasSandboxService = original }()

	facade := &fakeNasSandboxFacade{}
	newNasSandboxService = func() nasSandboxFacade {
		return facade
	}

	if _, err := NasSanboxSubmit(context.Background(), sandboxTestRequest(`{}`)); err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if _, err := NasSanboxReset(context.Background(), sandboxTestRequest(`{}`)); err != nil {
		t.Fatalf("unexpected reset error: %v", err)
	}
	if _, err := NasSanboxExit(context.Background(), sandboxTestRequest(`{}`)); err != nil {
		t.Fatalf("unexpected exit error: %v", err)
	}
	if facade.commitCalls != 1 || facade.resetCalls != 1 || facade.exitCalls != 1 {
		t.Fatalf("unexpected action calls: commit=%d reset=%d exit=%d", facade.commitCalls, facade.resetCalls, facade.exitCalls)
	}

	if _, err := NasSanboxPartitionFormat(context.Background(), sandboxTestRequest(`{"path":"/dev/sda1"}`)); err != nil {
		t.Fatalf("unexpected format error: %v", err)
	}
	if facade.formatPath != "/dev/sda1" {
		t.Fatalf("unexpected format path: %q", facade.formatPath)
	}
}

func TestNasSandboxCompatibilityPropagatesErrors(t *testing.T) {
	original := newNasSandboxService
	defer func() { newNasSandboxService = original }()

	expectedErr := errors.New("sandbox failed")
	newNasSandboxService = func() nasSandboxFacade {
		return &fakeNasSandboxFacade{
			listDisksErr: expectedErr,
			statusErr:    expectedErr,
			formatErr:    expectedErr,
			commitErr:    expectedErr,
			resetErr:     expectedErr,
			exitErr:      expectedErr,
		}
	}

	if _, err := NasSanboxDisks(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected disks error, got %v", err)
	}
	if _, err := NasSanboxStatus(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("expected status error, got %v", err)
	}
	if _, err := NasSanboxSubmit(context.Background(), sandboxTestRequest(`{}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("expected submit error, got %v", err)
	}
	if _, err := NasSanboxReset(context.Background(), sandboxTestRequest(`{}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("expected reset error, got %v", err)
	}
	if _, err := NasSanboxExit(context.Background(), sandboxTestRequest(`{}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("expected exit error, got %v", err)
	}
	if _, err := NasSanboxPartitionFormat(context.Background(), sandboxTestRequest(`{"path":"/dev/sda1"}`)); !errors.Is(err, expectedErr) {
		t.Fatalf("expected format error, got %v", err)
	}
}

package homebox

import (
	"context"
	"errors"
	"testing"
)

type fakeHomeBoxRuntimeChecker struct {
	isRunning bool
}

func (checker *fakeHomeBoxRuntimeChecker) IsRunning() bool {
	return checker.isRunning
}

type fakeHomeBoxStarter struct {
	calls int
	err   error
}

func (starter *fakeHomeBoxStarter) Start(ctx context.Context) error {
	starter.calls++
	return starter.err
}

func TestHomeBoxEnableServiceReturnsFixedPortWithoutRestartWhenAlreadyRunning(t *testing.T) {
	t.Parallel()

	starter := &fakeHomeBoxStarter{}
	svc := &HomeBoxEnableService{
		runtimeChecker: &fakeHomeBoxRuntimeChecker{isRunning: true},
		starter:        starter,
	}

	resp, err := svc.Enable(context.Background())
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if starter.calls != 0 {
		t.Fatalf("expected no starter calls when already running, got %d", starter.calls)
	}
	if resp == nil || resp.Result == nil || resp.Result.Port != "3300" {
		t.Fatalf("expected fixed homebox port response, got %#v", resp)
	}
}

func TestHomeBoxEnableServiceStartsHomeBoxWhenNotRunning(t *testing.T) {
	t.Parallel()

	starter := &fakeHomeBoxStarter{}
	svc := &HomeBoxEnableService{
		runtimeChecker: &fakeHomeBoxRuntimeChecker{isRunning: false},
		starter:        starter,
	}

	resp, err := svc.Enable(context.Background())
	if err != nil {
		t.Fatalf("unexpected service error: %v", err)
	}
	if starter.calls != 1 {
		t.Fatalf("expected starter to be called once, got %d", starter.calls)
	}
	if resp == nil || resp.Result == nil || resp.Result.Port != "3300" {
		t.Fatalf("expected fixed homebox port response, got %#v", resp)
	}
}

func TestHomeBoxEnableServiceMapsStarterFailureToLegacyError(t *testing.T) {
	t.Parallel()

	starter := &fakeHomeBoxStarter{err: errors.New("restart failed")}
	svc := &HomeBoxEnableService{
		runtimeChecker: &fakeHomeBoxRuntimeChecker{isRunning: false},
		starter:        starter,
	}

	if _, err := svc.Enable(context.Background()); err == nil || err.Error() != "homebox 启动失败" {
		t.Fatalf("expected legacy startup error, got %v", err)
	}
	if starter.calls != 1 {
		t.Fatalf("expected starter to be called once, got %d", starter.calls)
	}
}

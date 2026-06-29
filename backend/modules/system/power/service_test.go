package power

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	commands []string
	err      error
}

func (store *fakeStore) Run(ctx context.Context, commands []string) error {
	store.commands = append([]string(nil), commands...)
	return store.err
}

func TestRebootRunsCommandsAndReturnsSuccess(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)

	resp, err := svc.Reboot(context.Background())
	if err != nil {
		t.Fatalf("Reboot returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want 0", resp.Success)
	}
	want := []string{"echo 'trigger reboot'", "reboot"}
	if !reflect.DeepEqual(store.commands, want) {
		t.Fatalf("commands = %#v, want %#v", store.commands, want)
	}
}

func TestRebootMapsStoreError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{err: errors.New("permission denied")})

	if _, err := svc.Reboot(context.Background()); err == nil || err.Error() != "重启失败permission denied" {
		t.Fatalf("Reboot error = %v, want 重启失败permission denied", err)
	}
}

func TestPowerOffRunsCommandsAndReturnsSuccess(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)

	resp, err := svc.PowerOff(context.Background())
	if err != nil {
		t.Fatalf("PowerOff returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want 0", resp.Success)
	}
	want := []string{"echo 'trigger poweroff'", "poweroff"}
	if !reflect.DeepEqual(store.commands, want) {
		t.Fatalf("commands = %#v, want %#v", store.commands, want)
	}
}

func TestPowerOffMapsStoreError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{err: errors.New("permission denied")})

	if _, err := svc.PowerOff(context.Background()); err == nil || err.Error() != "关机失败permission denied" {
		t.Fatalf("PowerOff error = %v, want 关机失败permission denied", err)
	}
}

package updatecheck

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	output   string
	exitCode int
	checkErr error
	commands []string
	runErr   error
}

func (store *fakeStore) RunUpdateCheck(ctx context.Context) (string, int, error) {
	return store.output, store.exitCode, store.checkErr
}

func (store *fakeStore) ApplyAutoCheckCommands(ctx context.Context, commands []string) error {
	store.commands = append([]string(nil), commands...)
	return store.runErr
}

func TestCheckReportsNeedUpdateWhenCommandSucceeds(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{output: "new firmware\n"})

	result, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if !result.NeedUpdate {
		t.Fatal("NeedUpdate = false, want true")
	}
	if result.Msg != "new firmware\n" {
		t.Fatalf("Msg = %q", result.Msg)
	}
}

func TestCheckTreatsExitCodeOneAsAlreadyLatest(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{
		output:   "ignored output",
		exitCode: 1,
		checkErr: errors.New("exit status 1"),
	})

	result, err := svc.Check(context.Background())
	if err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
	if result.NeedUpdate {
		t.Fatal("NeedUpdate = true, want false")
	}
	if result.Msg != "Already the latest firmware" {
		t.Fatalf("Msg = %q", result.Msg)
	}
}

func TestCheckPropagatesUnexpectedErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("ota failed")
	svc := NewService(&fakeStore{
		output:   "failure details",
		exitCode: 2,
		checkErr: expectedErr,
	})

	if _, err := svc.Check(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("Check error = %v, want expectedErr", err)
	}
}

func TestSetAutoCheckBuildsEnableCommands(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)

	resp, err := svc.SetAutoCheck(context.Background(), models.SystemAutoCheckUpdateRequest{Enable: true})
	if err != nil {
		t.Fatalf("SetAutoCheck returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want 0", resp.Success)
	}
	want := []string{
		"uci delete quickstart.main.disable_update_check",
		"uci commit quickstart",
	}
	if !reflect.DeepEqual(store.commands, want) {
		t.Fatalf("commands = %#v, want %#v", store.commands, want)
	}
}

func TestSetAutoCheckBuildsDisableCommands(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)

	_, err := svc.SetAutoCheck(context.Background(), models.SystemAutoCheckUpdateRequest{Enable: false})
	if err != nil {
		t.Fatalf("SetAutoCheck returned error: %v", err)
	}
	want := []string{
		"uci set quickstart.main.disable_update_check=1",
		"uci commit quickstart",
	}
	if !reflect.DeepEqual(store.commands, want) {
		t.Fatalf("commands = %#v, want %#v", store.commands, want)
	}
}

func TestSetAutoCheckMapsStoreError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{runErr: errors.New("batch failed")})

	if _, err := svc.SetAutoCheck(context.Background(), models.SystemAutoCheckUpdateRequest{}); err == nil || err.Error() != "设置失败" {
		t.Fatalf("SetAutoCheck error = %v, want 设置失败", err)
	}
}

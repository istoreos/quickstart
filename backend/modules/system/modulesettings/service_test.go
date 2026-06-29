package modulesettings

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	modules    []string
	readErr    error
	hasSection bool
	commands   []string
	applyErr   error
}

func (store *fakeStore) ReadDisabledDisplayModules(ctx context.Context) ([]string, error) {
	return store.modules, store.readErr
}

func (store *fakeStore) HasDisabledDisplaySection(ctx context.Context) bool {
	return store.hasSection
}

func (store *fakeStore) ApplyCommands(ctx context.Context, commands []string) error {
	store.commands = append([]string(nil), commands...)
	return store.applyErr
}

func TestGetReturnsDisabledDisplayModules(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{modules: []string{"smart", "raid"}})

	result, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !reflect.DeepEqual(result.DiableDisplay, []string{"smart", "raid"}) {
		t.Fatalf("DiableDisplay = %#v", result.DiableDisplay)
	}
}

func TestGetReturnsEmptySliceWhenStoreHasNoModules(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{})

	result, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if result.DiableDisplay == nil {
		t.Fatal("DiableDisplay = nil, want empty slice")
	}
	if len(result.DiableDisplay) != 0 {
		t.Fatalf("DiableDisplay = %#v, want empty", result.DiableDisplay)
	}
}

func TestGetPropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("read failed")
	svc := NewService(&fakeStore{readErr: expectedErr})

	if _, err := svc.Get(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("Get error = %v, want expectedErr", err)
	}
}

func TestSetBuildsCommandsForExistingSection(t *testing.T) {
	t.Parallel()

	store := &fakeStore{hasSection: true}
	svc := NewService(store)

	resp, err := svc.Set(context.Background(), models.SystemModuleSettingsRequest{
		DiableDisplay: []string{"smart", "raid"},
	})
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want code 0", resp.Success)
	}
	want := []string{
		"delete quickstart.modules",
		"set quickstart.modules=disabledisplay",
		"add_list quickstart.modules.module='smart'",
		"add_list quickstart.modules.module='raid'",
		"commit quickstart",
	}
	if !reflect.DeepEqual(store.commands, want) {
		t.Fatalf("commands = %#v, want %#v", store.commands, want)
	}
}

func TestSetEmptyModulesWithoutExistingSectionSkipsApply(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)

	resp, err := svc.Set(context.Background(), models.SystemModuleSettingsRequest{
		DiableDisplay: []string{},
	})
	if err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want code 0", resp.Success)
	}
	if store.commands != nil {
		t.Fatalf("commands = %#v, want nil", store.commands)
	}
}

func TestSetRejectsMissingDisabledDisplay(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{})

	if _, err := svc.Set(context.Background(), models.SystemModuleSettingsRequest{}); err == nil || err.Error() != "invalid params" {
		t.Fatalf("Set error = %v, want invalid params", err)
	}
}

func TestSetPropagatesApplyErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("apply failed")
	store := &fakeStore{hasSection: true, applyErr: expectedErr}
	svc := NewService(store)

	if _, err := svc.Set(context.Background(), models.SystemModuleSettingsRequest{
		DiableDisplay: []string{"smart"},
	}); !errors.Is(err, expectedErr) {
		t.Fatalf("Set error = %v, want expectedErr", err)
	}
}

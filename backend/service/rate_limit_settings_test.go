package service

import (
	"context"
	"errors"
	"testing"

	"github.com/istoreos/quickstart/backend/models"
)

type fakeRateLimitSettingsStore struct {
	current      models.RateLimitSettings
	applyErr     error
	readErr      error
	restoreErr   error
	applyCalls   int
	restoreCalls int
	hasRules     bool
}

func (store *fakeRateLimitSettingsStore) HasRules(context.Context, string) (bool, error) {
	return store.hasRules, nil
}

func (store *fakeRateLimitSettingsStore) Read(context.Context) (models.RateLimitSettings, error) {
	if store.readErr != nil {
		return models.RateLimitSettings{}, store.readErr
	}
	return normalizeRateLimitSettings(store.current), nil
}
func (*fakeRateLimitSettingsStore) Snapshot(context.Context) (rateLimitSettingsSnapshot, error) {
	return rateLimitSettingsSnapshot{Files: []devicePolicyFileSnapshot{{Path: "/tmp/eqos", Exists: true}}}, nil
}
func (store *fakeRateLimitSettingsStore) Apply(_ context.Context, value models.RateLimitSettings) error {
	store.applyCalls++
	if store.applyErr != nil {
		return store.applyErr
	}
	store.current = value
	return nil
}
func (store *fakeRateLimitSettingsStore) Restore(context.Context, rateLimitSettingsSnapshot) error {
	store.restoreCalls++
	return store.restoreErr
}

func TestRateLimitSettingsPlanDoesNotWrite(t *testing.T) {
	store := &fakeRateLimitSettingsStore{current: models.RateLimitSettings{Enabled: false, UploadSpeed: 100, DownloadSpeed: 1000}}
	module := NewRateLimitSettingsModule(store)
	response, err := module.Plan(context.Background(), &models.RateLimitSettingsRequest{Settings: &models.RateLimitSettings{Enabled: true, UploadSpeed: 200, DownloadSpeed: 2000}})
	if err != nil || response.Result.Error != nil || !response.Result.CanApply || len(response.Result.Changes) != 1 {
		t.Fatalf("unexpected plan: %#v, %v", response, err)
	}
	if store.applyCalls != 0 {
		t.Fatal("plan must not write")
	}
}

func TestRateLimitSettingsApplyRejectsStaleVersion(t *testing.T) {
	store := &fakeRateLimitSettingsStore{current: models.RateLimitSettings{UploadSpeed: 100, DownloadSpeed: 1000}}
	module := NewRateLimitSettingsModule(store)
	response, err := module.Apply(context.Background(), &models.RateLimitSettingsRequest{ExpectedVersion: "stale", Settings: &models.RateLimitSettings{Enabled: true, UploadSpeed: 200, DownloadSpeed: 2000}})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "conflict" {
		t.Fatalf("expected conflict: %#v, %v", response, err)
	}
	if store.applyCalls != 0 {
		t.Fatal("conflict must not write")
	}
}

func TestRateLimitSettingsApplyRollsBackFailure(t *testing.T) {
	store := &fakeRateLimitSettingsStore{current: models.RateLimitSettings{UploadSpeed: 100, DownloadSpeed: 1000}, applyErr: errors.New("reload failed")}
	module := NewRateLimitSettingsModule(store)
	response, err := module.Apply(context.Background(), &models.RateLimitSettingsRequest{IdempotencyKey: "rollback", Settings: &models.RateLimitSettings{Enabled: true, UploadSpeed: 200, DownloadSpeed: 2000}})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "rolled_back" {
		t.Fatalf("expected rollback: %#v, %v", response, err)
	}
	if store.restoreCalls != 1 {
		t.Fatalf("restore calls = %d", store.restoreCalls)
	}
}

func TestRateLimitSettingsApplyVerifiesCommittedValue(t *testing.T) {
	store := &fakeRateLimitSettingsStore{current: models.RateLimitSettings{UploadSpeed: 100, DownloadSpeed: 1000}}
	module := NewRateLimitSettingsModule(store)
	response, err := module.Apply(context.Background(), &models.RateLimitSettingsRequest{IdempotencyKey: "commit", Settings: &models.RateLimitSettings{Enabled: true, UploadSpeed: 200, DownloadSpeed: 2000}})
	if err != nil || response.Result.Error != nil || !response.Result.Changed || response.Result.Current.UploadSpeed != 200 {
		t.Fatalf("unexpected result: %#v, %v", response, err)
	}
}

func TestRateLimitSettingsBlocksProviderSwitchWithExistingRules(t *testing.T) {
	store := &fakeRateLimitSettingsStore{current: models.RateLimitSettings{Enabled: true, UploadSpeed: 100, DownloadSpeed: 1000, Provider: "bandix"}, hasRules: true}
	module := NewRateLimitSettingsModule(store)
	response, err := module.Plan(context.Background(), &models.RateLimitSettingsRequest{Settings: &models.RateLimitSettings{Enabled: true, UploadSpeed: 100, DownloadSpeed: 1000, Provider: "eqos"}})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "provider_has_rules" {
		t.Fatalf("expected provider_has_rules, got %#v, %v", response, err)
	}
	if store.applyCalls != 0 {
		t.Fatal("provider switch must not write")
	}
}

func TestNormalizeExternalRateLimitSettingsKeepsInactiveEQoSTotals(t *testing.T) {
	current := models.RateLimitSettings{Enabled: false, UploadSpeed: 20, DownloadSpeed: 50, Provider: "eqos"}
	desired := models.RateLimitSettings{Enabled: false, UploadSpeed: 200, DownloadSpeed: 2000, Provider: "bandix"}

	got := normalizeExternalRateLimitSettings(desired, current)
	if !got.Enabled || got.Provider != "bandix" {
		t.Fatalf("unexpected Bandix state: %#v", got)
	}
	if got.UploadSpeed != current.UploadSpeed || got.DownloadSpeed != current.DownloadSpeed {
		t.Fatalf("Bandix changed inactive eQoS totals: %#v", got)
	}
}

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

type rateLimitSettingsStore interface {
	Read(context.Context) (models.RateLimitSettings, error)
	HasRules(context.Context, string) (bool, error)
	Snapshot(context.Context) (rateLimitSettingsSnapshot, error)
	Apply(context.Context, models.RateLimitSettings) error
	Restore(context.Context, rateLimitSettingsSnapshot) error
}

type rateLimitSettingsSnapshot struct{ Files []devicePolicyFileSnapshot }

type RateLimitSettingsModule struct {
	mu           sync.Mutex
	store        rateLimitSettingsStore
	transactions *TaskTransactionJournal
}

func NewRateLimitSettingsModule(store rateLimitSettingsStore) *RateLimitSettingsModule {
	return &RateLimitSettingsModule{store: store, transactions: newMemoryTaskTransactionJournal()}
}

func NewDefaultRateLimitSettingsModule() *RateLimitSettingsModule {
	return NewRateLimitSettingsModule(&eqosRateLimitSettingsStore{configDir: deviceRestrictionConfigDir()})
}

func (module *RateLimitSettingsModule) Plan(ctx context.Context, request *models.RateLimitSettingsRequest) (*models.RateLimitSettingsResponse, error) {
	result := &models.RateLimitSettingsResult{Changes: []*models.PolicyPlanChange{}, ReloadServices: []string{"eqos"}}
	current, err := module.store.Read(ctx)
	if err != nil {
		result.Error = &models.DevicePolicyError{Code: "status_unavailable", Message: err.Error()}
		return &models.RateLimitSettingsResponse{Result: result}, nil
	}
	result.Current = &current
	result.Version = bindLANVersion(ctx, rateLimitSettingsVersion(current))
	result.RollbackPoint = result.Version
	if request == nil || request.Settings == nil {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "settings are required"}
		return &models.RateLimitSettingsResponse{Result: result}, nil
	}
	desired := normalizeRateLimitSettings(*request.Settings)
	if selectedLAN(ctx) != defaultLAN && desired.Provider != nativePolicyProviderName {
		result.Error = &models.DevicePolicyError{Code: "provider_scope_unsupported", Message: "此网络需要可独立接入的策略服务"}
		return &models.RateLimitSettingsResponse{Result: result}, nil
	}
	if desired.Provider == "bandix" || desired.Provider == nativePolicyProviderName {
		result.ReloadServices = []string{}
		installed, supported, reason := false, false, "dependency_not_installed"
		if desired.Provider == nativePolicyProviderName {
			installed, supported, reason = nativeProviderCapability(ctx)
		} else {
			installed, supported, reason = bandixProviderAvailability(ctx)
		}
		if !installed || !supported {
			result.Desired = &desired
			result.Error = &models.DevicePolicyError{Code: "dependency_not_installed", Message: reason}
			return &models.RateLimitSettingsResponse{Result: result}, nil
		}
		desired = normalizeExternalRateLimitSettings(desired, current)
	}
	result.Desired = &desired
	if desired.Enabled && (desired.UploadSpeed <= 0 || desired.DownloadSpeed <= 0) {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "upload and download bandwidth must be greater than zero"}
		return &models.RateLimitSettingsResponse{Result: result}, nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != result.Version {
		result.Error = &models.DevicePolicyError{Code: "conflict", Message: "rate limit settings changed; plan again"}
		return &models.RateLimitSettingsResponse{Result: result}, nil
	}
	if current.Provider != desired.Provider {
		hasRules, rulesErr := module.store.HasRules(ctx, current.Provider)
		if rulesErr != nil {
			result.Error = &models.DevicePolicyError{Code: "status_unavailable", Message: "could not inspect existing device limits"}
			return &models.RateLimitSettingsResponse{Result: result}, nil
		}
		if hasRules {
			result.Error = &models.DevicePolicyError{Code: "provider_has_rules", Message: "请先移除现有限速规则，再切换执行方式"}
			return &models.RateLimitSettingsResponse{Result: result}, nil
		}
	}
	if current != desired {
		result.Changes = append(result.Changes, &models.PolicyPlanChange{Kind: "rate_limit_service", Description: "change rate limit service and total bandwidth"})
	}
	result.CanApply = true
	return &models.RateLimitSettingsResponse{Result: result}, nil
}

func (module *RateLimitSettingsModule) Apply(ctx context.Context, request *models.RateLimitSettingsRequest) (*models.RateLimitSettingsResponse, error) {
	locked, release, lockErr := lockLANWrite(ctx)
	if lockErr != nil {
		return nil, lockErr
	}
	defer release()
	ctx = locked

	module.mu.Lock()
	defer module.mu.Unlock()
	plan, err := module.Plan(ctx, request)
	if err != nil || plan.Result == nil || plan.Result.Error != nil || !plan.Result.CanApply {
		return plan, err
	}
	result := plan.Result
	if len(result.Changes) == 0 {
		return plan, nil
	}
	begin, beginErr := module.transactions.Begin(ctx, "rate_limit_settings", request.IdempotencyKey, result.Version+rateLimitSettingsVersion(*result.Desired))
	if beginErr != nil {
		result.Error = &models.DevicePolicyError{Code: "transaction_unavailable", Message: beginErr.Error()}
		return plan, nil
	}
	result.Transaction = begin.Transaction
	if begin.Conflict {
		result.Error = &models.DevicePolicyError{Code: "conflict", Message: "idempotency key was already used for another change"}
		return plan, nil
	}
	if begin.Replay {
		result.Error = failedTaskReplay(begin.Transaction)
		if result.Error != nil {
			result.CanApply = false
		}
		return plan, nil
	}
	snapshot, snapshotErr := module.store.Snapshot(ctx)
	if snapshotErr != nil {
		result.Error = &models.DevicePolicyError{Code: "apply_failed", Message: "could not create rollback snapshot"}
		return plan, nil
	}
	if txErr := advanceTaskTransaction(ctx, module.transactions, begin.Transaction, "apply", "in_progress", "restore_task_snapshot"); txErr != nil {
		result.Error = txErr
		return plan, nil
	}
	if applyErr := module.store.Apply(ctx, *result.Desired); applyErr != nil {
		return module.rollback(ctx, plan, snapshot, begin.Transaction, "rate limit settings were not applied")
	}
	updated, verifyErr := module.store.Read(ctx)
	if verifyErr != nil || updated != *result.Desired {
		return module.rollback(ctx, plan, snapshot, begin.Transaction, "rate limit settings could not be verified")
	}
	result.Current = &updated
	result.Changed = true
	result.Version = bindLANVersion(ctx, rateLimitSettingsVersion(updated))
	if txErr := advanceTaskTransaction(ctx, module.transactions, begin.Transaction, "verify", "committed", ""); txErr != nil {
		result.Error = txErr
		return plan, nil
	}
	return plan, nil
}

func (module *RateLimitSettingsModule) rollback(ctx context.Context, response *models.RateLimitSettingsResponse, snapshot rateLimitSettingsSnapshot, transaction *models.TaskTransaction, message string) (*models.RateLimitSettingsResponse, error) {
	if err := module.store.Restore(ctx, snapshot); err != nil {
		response.Result.Error = &models.DevicePolicyError{Code: "recovery_required", Message: message + "; rollback failed"}
		if txErr := advanceTaskTransaction(ctx, module.transactions, transaction, "recover", "recovery_required", "restore_task_snapshot"); txErr != nil {
			response.Result.Error = txErr
			return response, nil
		}
		return response, nil
	}
	response.Result.Error = &models.DevicePolicyError{Code: "rolled_back", Message: message + "; original settings were restored"}
	if txErr := advanceTaskTransaction(ctx, module.transactions, transaction, "rollback", "rolled_back", "retry"); txErr != nil {
		response.Result.Error = txErr
		return response, nil
	}
	return response, nil
}

func normalizeRateLimitSettings(value models.RateLimitSettings) models.RateLimitSettings {
	if value.Provider != "bandix" && value.Provider != nativePolicyProviderName {
		value.Provider = "eqos"
	}
	if value.UploadSpeed == 0 {
		value.UploadSpeed = 200
	}
	if value.DownloadSpeed == 0 {
		value.DownloadSpeed = 2000
	}
	return value
}

// Bandix enforces absolute per-device rates and has no global bandwidth
// setting. Keep the eQoS totals unchanged while Bandix is selected so hidden,
// semantically inactive form fields cannot create an unverifiable change.
func normalizeExternalRateLimitSettings(desired, current models.RateLimitSettings) models.RateLimitSettings {
	desired.Enabled = true
	desired.UploadSpeed = current.UploadSpeed
	desired.DownloadSpeed = current.DownloadSpeed
	return desired
}

func rateLimitSettingsVersion(value models.RateLimitSettings) string {
	sum := sha256.Sum256([]byte(value.Provider + ":" + boolString(value.Enabled) + ":" + int64String(value.UploadSpeed) + ":" + int64String(value.DownloadSpeed)))
	return hex.EncodeToString(sum[:8])
}

func boolString(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func int64String(value int64) string { return fmt.Sprintf("%d", value) }

type eqosRateLimitSettingsStore struct{ configDir string }

func (store *eqosRateLimitSettingsStore) HasRules(ctx context.Context, provider string) (bool, error) {
	if selectedLAN(ctx) != defaultLAN && provider != nativePolicyProviderName {
		return false, errors.New("provider does not support this network")
	}
	if provider == nativePolicyProviderName {
		native := newScopedNativeRateLimitProvider()
		snapshot, err := native.snapshot(ctx)
		if err != nil {
			return false, err
		}
		for _, policy := range snapshot.Policies {
			if policy.Owner == native.owner && policy.RateLimit != nil && policy.RateLimit.Enabled {
				return true, nil
			}
		}
		return false, nil
	}
	if provider == "bandix" {
		rules, err := NewBandixRateLimitProvider(defaultBandixBaseURL(), nil).(*bandixRateLimitProvider).rules(ctx)
		return len(rules) > 0, err
	}
	tree := uci.NewTree(store.configDir)
	if err := tree.LoadConfig("eqos", true); err != nil {
		return false, err
	}
	sections, _ := tree.GetSections("eqos", "device")
	return len(sections) > 0, nil
}

func (store *eqosRateLimitSettingsStore) Read(ctx context.Context) (models.RateLimitSettings, error) {
	if selectedLAN(ctx) != defaultLAN {
		_, available, _ := nativeProviderCapability(ctx)
		return normalizeRateLimitSettings(models.RateLimitSettings{Provider: scopedRateLimitProvider(ctx), Enabled: available}), nil
	}
	state, err := NewDefaultSpeedLimitReader().ReadSpeedLimitStatus(ctx)
	if err != nil {
		return models.RateLimitSettings{}, err
	}
	result := normalizeRateLimitSettings(models.RateLimitSettings{Enabled: state.Enabled, UploadSpeed: state.UploadSpeed, DownloadSpeed: state.DownloadSpeed, Provider: effectiveRateLimitProvider(defaultRateLimitProviderPath)})
	if result.Provider == "bandix" || result.Provider == nativePolicyProviderName {
		result.Enabled = true
	}
	return result, nil
}

func (store *eqosRateLimitSettingsStore) Snapshot(ctx context.Context) (rateLimitSettingsSnapshot, error) {
	result := rateLimitSettingsSnapshot{Files: make([]devicePolicyFileSnapshot, 0, 2)}
	paths := []string{lanStatePath(defaultRateLimitProviderPath, selectedLAN(ctx))}
	if selectedLAN(ctx) == defaultLAN {
		paths = append(paths, filepath.Join(store.configDir, "eqos"))
	}
	for _, path := range paths {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			result.Files = append(result.Files, devicePolicyFileSnapshot{Path: path})
			continue
		}
		if err != nil {
			return rateLimitSettingsSnapshot{}, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return rateLimitSettingsSnapshot{}, err
		}
		result.Files = append(result.Files, devicePolicyFileSnapshot{Path: path, Data: data, Mode: info.Mode(), Exists: true})
	}
	return result, nil
}

func (*eqosRateLimitSettingsStore) Apply(ctx context.Context, value models.RateLimitSettings) error {
	if selectedLAN(ctx) != defaultLAN && value.Provider != nativePolicyProviderName {
		return errors.New("provider does not support this network")
	}
	if value.Provider == "eqos" {
		if err := NewDefaultLanSpeedLimitWriteService().SetSpeedLimitModule(ctx, SpeedLimitModuleInput{Enabled: value.Enabled, UploadSpeed: value.UploadSpeed, DownloadSpeed: value.DownloadSpeed}); err != nil {
			return err
		}
	}
	return writeRateLimitProviderPreference(lanStatePath(defaultRateLimitProviderPath, selectedLAN(ctx)), value.Provider)
}

func (store *eqosRateLimitSettingsStore) Restore(ctx context.Context, snapshot rateLimitSettingsSnapshot) error {
	kind := "speed"
	if selectedLAN(ctx) != defaultLAN {
		kind = "provider_preference"
	}
	backup := &systemDevicePolicyBackup{Kind: kind, Files: snapshot.Files}
	return (&systemDevicePolicyStore{}).Restore(ctx, backup)
}

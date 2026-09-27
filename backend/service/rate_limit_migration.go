package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const defaultRateLimitMigrationReceipt = "/etc/quickstart/rate-limit-migration-v1.json"

type rateLimitMigrationReceipt struct {
	SchemaVersion     int                            `json:"schemaVersion"`
	MigrationID       string                         `json:"migrationId"`
	CreatedAt         string                         `json:"createdAt"`
	SourcePlan        *models.RateLimitMigrationPlan `json:"sourcePlan"`
	PreviousProvider  string                         `json:"previousProvider"`
	PreviousSnapshot  nativePolicySnapshot           `json:"previousSnapshot"`
	CommittedSnapshot nativePolicySnapshot           `json:"committedSnapshot"`
	RolledBack        bool                           `json:"rolledBack"`
}

type RateLimitMigrationModule struct {
	mu             sync.Mutex
	bandix         *bandixRateLimitProvider
	native         *nativeRateLimitProvider
	preferencePath string
	receiptPath    string
	now            func() time.Time
}

func NewDefaultRateLimitMigrationModule() *RateLimitMigrationModule {
	return &RateLimitMigrationModule{
		bandix:         NewBandixRateLimitProvider(defaultBandixBaseURL(), nil).(*bandixRateLimitProvider),
		native:         NewNativeRateLimitProvider(defaultNativePolicyBaseURL(), nil).(*nativeRateLimitProvider),
		preferencePath: defaultRateLimitProviderPath,
		receiptPath:    defaultRateLimitMigrationReceipt,
		now:            time.Now,
	}
}

func (module *RateLimitMigrationModule) Plan(ctx context.Context) (*models.RateLimitMigrationResponse, error) {
	if module == nil || module.bandix == nil {
		return nil, errors.New("rate limit migration module is unavailable")
	}
	rules, err := module.bandix.rules(ctx)
	if err != nil {
		return migrationResult(nil, false, false, "", &models.DevicePolicyError{Code: "source_unavailable", Message: err.Error()}), nil
	}
	plan := buildRateLimitMigrationPlan(rules)
	return migrationResult(plan, false, false, "", nil), nil
}

func (module *RateLimitMigrationModule) Apply(ctx context.Context, request *models.RateLimitMigrationRequest) (*models.RateLimitMigrationResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	if request == nil || request.Plan == nil || request.ExpectedVersion == "" {
		return migrationResult(nil, false, false, "", &models.DevicePolicyError{Code: "validation_failed", Message: "migration plan and expectedVersion are required"}), nil
	}
	plan := request.Plan
	if request.ExpectedVersion != plan.Version || migrationPlanVersion(plan.Items) != plan.Version {
		return migrationResult(plan, false, false, "", &models.DevicePolicyError{Code: "conflict", Message: "migration preview changed; scan Bandix again"}), nil
	}
	if err := validateRateLimitMigrationPlan(plan); err != nil {
		return migrationResult(plan, false, false, "", &models.DevicePolicyError{Code: "needs_attention", Message: "resolve unsupported or conflicting Bandix rules first"}), nil
	}
	if err := module.native.probe(ctx); err != nil {
		return migrationResult(plan, false, false, "", &models.DevicePolicyError{
			Code: "target_unavailable", Message: "stop Bandix, start Quickstart NetPolicy, then retry: " + err.Error(),
		}), nil
	}
	before, err := module.native.snapshot(ctx)
	if err != nil {
		return migrationResult(plan, false, false, "", &models.DevicePolicyError{Code: "snapshot_failed", Message: err.Error()}), nil
	}
	previousProvider := readRateLimitProviderPreference(module.preferencePath)
	migrationID := fmt.Sprintf("migration-%d", module.now().UnixNano())
	desired := make([]nativePolicyInput, 0, plan.ConvertibleCount)
	for _, item := range plan.Items {
		if item.Disposition != "convert" || item.Desired == nil {
			continue
		}
		desired = append(desired, nativePolicyInput{
			Device: nativeDeviceIdentity{Kind: "mac", Value: strings.ToUpper(item.MAC)},
			RateLimit: &nativeRateLimit{
				Enabled: true, UploadBitsPerSecond: item.UploadBitsPerSecond,
				DownloadBitsPerSecond: item.DownloadBitsPerSecond,
			},
		})
	}
	committed, err := module.native.replaceOwned(ctx, before, desired, migrationID)
	if err != nil {
		return migrationResult(plan, false, false, "", &models.DevicePolicyError{Code: "target_apply_failed", Message: err.Error()}), nil
	}
	rollbackNative := func() error {
		_, restoreErr := module.native.replaceOwned(ctx, committed, module.native.ownedInputs(before), migrationID+"-rollback")
		return restoreErr
	}
	if err := writeRateLimitProviderPreference(module.preferencePath, nativePolicyProviderName); err != nil {
		return migrationResult(plan, false, false, "", migrationRollbackError("provider switch failed", err, rollbackNative())), nil
	}
	receipt := rateLimitMigrationReceipt{
		SchemaVersion: 1, MigrationID: migrationID, CreatedAt: module.now().UTC().Format(time.RFC3339),
		SourcePlan: plan, PreviousProvider: previousProvider, PreviousSnapshot: before, CommittedSnapshot: committed,
	}
	if err := writeMigrationReceipt(module.receiptPath, receipt); err != nil {
		providerErr := writeRateLimitProviderPreference(module.preferencePath, previousProvider)
		restoreErr := rollbackNative()
		if providerErr != nil && restoreErr == nil {
			restoreErr = providerErr
		}
		return migrationResult(plan, false, false, "", migrationRollbackError("audit receipt failed", err, restoreErr)), nil
	}
	return migrationResult(plan, true, false, migrationID, nil), nil
}

func (module *RateLimitMigrationModule) Rollback(ctx context.Context, request *models.RateLimitMigrationRollbackRequest) (*models.RateLimitMigrationResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	receipt, err := readRateLimitMigrationReceipt(module.receiptPath)
	if err != nil {
		return migrationResult(nil, false, false, "", &models.DevicePolicyError{Code: "snapshot_failed", Message: err.Error()}), nil
	}
	if request == nil || request.MigrationID == "" || request.MigrationID != receipt.MigrationID {
		return migrationResult(receipt.SourcePlan, false, false, receipt.MigrationID, &models.DevicePolicyError{Code: "conflict", Message: "migration receipt does not match"}), nil
	}
	if receipt.RolledBack {
		return migrationResult(receipt.SourcePlan, false, true, receipt.MigrationID, nil), nil
	}
	current, err := module.native.snapshot(ctx)
	if err != nil {
		return migrationResult(receipt.SourcePlan, false, false, receipt.MigrationID, &models.DevicePolicyError{Code: "target_unavailable", Message: err.Error()}), nil
	}
	if _, err = module.native.replaceOwned(ctx, current, module.native.ownedInputs(receipt.PreviousSnapshot), receipt.MigrationID+"-rollback"); err != nil {
		return migrationResult(receipt.SourcePlan, false, false, receipt.MigrationID, &models.DevicePolicyError{Code: "recovery_required", Message: err.Error()}), nil
	}
	if err = writeRateLimitProviderPreference(module.preferencePath, receipt.PreviousProvider); err != nil {
		return migrationResult(receipt.SourcePlan, false, false, receipt.MigrationID, &models.DevicePolicyError{Code: "recovery_required", Message: err.Error()}), nil
	}
	receipt.RolledBack = true
	if err = writeMigrationReceipt(module.receiptPath, receipt); err != nil {
		return migrationResult(receipt.SourcePlan, false, true, receipt.MigrationID, &models.DevicePolicyError{Code: "audit_failed", Message: err.Error()}), nil
	}
	return migrationResult(receipt.SourcePlan, false, true, receipt.MigrationID, nil), nil
}

func buildRateLimitMigrationPlan(rules []bandixScheduleRule) *models.RateLimitMigrationPlan {
	items := make([]*models.RateLimitMigrationItem, 0, len(rules))
	counts := map[string]int{}
	for _, rule := range rules {
		counts[normalizeInventoryMAC(rule.MAC)]++
	}
	for _, rule := range rules {
		mac := strings.ToUpper(normalizeInventoryMAC(rule.MAC))
		item := &models.RateLimitMigrationItem{
			ID: fmt.Sprint(rule.ID), MAC: mac, Disposition: "convert",
			UploadBytesPerSecond: rule.UploadBytes, DownloadBytesPerSecond: rule.DownloadBytes,
		}
		switch {
		case !validMigrationMAC(mac):
			item.Disposition, item.Reason = "unsupported", "invalid_mac"
		case counts[normalizeInventoryMAC(rule.MAC)] > 1:
			item.Disposition, item.Reason = "conflict", "duplicate_device_rule"
		case len(rule.UnknownFields) > 0:
			item.Disposition, item.Reason = "unsupported", "unknown_fields:"+strings.Join(rule.UnknownFields, ",")
		case !isFullWeekRule(rule):
			item.Disposition, item.Reason = "unsupported", "scheduled_rule_not_supported"
		case rule.UploadBytes < 0 || rule.DownloadBytes < 0:
			item.Disposition, item.Reason = "unsupported", "invalid_rate_limit"
		case rule.UploadBytes > math.MaxInt64/8 || rule.DownloadBytes > math.MaxInt64/8:
			item.Disposition, item.Reason = "unsupported", "rate_limit_overflow"
		case rule.UploadBytes == 0 && rule.DownloadBytes == 0:
			item.Disposition, item.Reason = "unsupported", "empty_rate_limit"
		default:
			item.UploadBitsPerSecond = rule.UploadBytes * 8
			item.DownloadBitsPerSecond = rule.DownloadBytes * 8
			item.Desired = &models.DeviceSpeedPolicy{
				Enabled: true, UploadSpeed: bytesPerSecondToMbit(rule.UploadBytes), DownloadSpeed: bytesPerSecondToMbit(rule.DownloadBytes),
			}
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].MAC < items[j].MAC || (items[i].MAC == items[j].MAC && items[i].ID < items[j].ID)
	})
	plan := &models.RateLimitMigrationPlan{
		Source: "bandix", Target: nativePolicyProviderName, Items: items,
		RequiredActions: []string{"stop_bandix_on_target_interface", "start_quickstart_netpolicy", "confirm_apply"},
	}
	for _, item := range items {
		switch item.Disposition {
		case "convert":
			plan.ConvertibleCount++
		case "conflict":
			plan.ConflictCount++
		default:
			plan.UnsupportedCount++
		}
	}
	plan.CanApply = len(items) > 0 && plan.UnsupportedCount == 0 && plan.ConflictCount == 0
	plan.Version = migrationPlanVersion(items)
	return plan
}

func validateRateLimitMigrationPlan(plan *models.RateLimitMigrationPlan) error {
	if plan == nil || plan.Source != "bandix" || plan.Target != nativePolicyProviderName || len(plan.Items) == 0 {
		return errors.New("invalid migration source, target, or empty plan")
	}
	seen := make(map[string]struct{}, len(plan.Items))
	convertible := 0
	for _, item := range plan.Items {
		if item == nil || item.Disposition != "convert" || item.Desired == nil || !item.Desired.Enabled {
			return errors.New("migration contains an unresolved item")
		}
		mac := strings.ToUpper(normalizeInventoryMAC(item.MAC))
		if !validMigrationMAC(mac) {
			return errors.New("migration contains an invalid device identity")
		}
		if _, duplicate := seen[mac]; duplicate {
			return errors.New("migration contains duplicate device rules")
		}
		seen[mac] = struct{}{}
		if item.UploadBytesPerSecond < 0 || item.DownloadBytesPerSecond < 0 ||
			item.UploadBytesPerSecond > math.MaxInt64/8 || item.DownloadBytesPerSecond > math.MaxInt64/8 ||
			(item.UploadBytesPerSecond == 0 && item.DownloadBytesPerSecond == 0) ||
			item.UploadBitsPerSecond != item.UploadBytesPerSecond*8 ||
			item.DownloadBitsPerSecond != item.DownloadBytesPerSecond*8 {
			return errors.New("migration contains an invalid rate limit")
		}
		convertible++
	}
	if !plan.CanApply || plan.ConvertibleCount != convertible || plan.UnsupportedCount != 0 || plan.ConflictCount != 0 {
		return errors.New("migration summary does not match its rules")
	}
	return nil
}

func validMigrationMAC(value string) bool {
	_, err := net.ParseMAC(value)
	return err == nil
}

func isFullWeekRule(rule bandixScheduleRule) bool {
	if rule.TimeSlot.Start != "00:00" || rule.TimeSlot.End != "23:59" || len(rule.TimeSlot.Days) != 7 {
		return false
	}
	days := append([]int(nil), rule.TimeSlot.Days...)
	sort.Ints(days)
	for index, day := range days {
		if day != index+1 {
			return false
		}
	}
	return true
}

func migrationPlanVersion(items []*models.RateLimitMigrationItem) string {
	payload, _ := json.Marshal(items)
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:12])
}

func migrationResult(plan *models.RateLimitMigrationPlan, changed, rolledBack bool, id string, policyErr *models.DevicePolicyError) *models.RateLimitMigrationResponse {
	return &models.RateLimitMigrationResponse{Result: &models.RateLimitMigrationResult{
		Plan: plan, MigrationID: id, Changed: changed, RolledBack: rolledBack, Error: policyErr,
	}}
}

func migrationRollbackError(message string, cause, rollback error) *models.DevicePolicyError {
	if rollback != nil {
		return &models.DevicePolicyError{Code: "recovery_required", Message: fmt.Sprintf("%s: %v; rollback failed: %v", message, cause, rollback)}
	}
	return &models.DevicePolicyError{Code: "rolled_back", Message: fmt.Sprintf("%s: %v; original native state restored", message, cause)}
}

func writeMigrationReceipt(path string, receipt rateLimitMigrationReceipt) error {
	payload, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err = os.WriteFile(temporary, payload, 0600); err != nil {
		return err
	}
	if err = os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func readRateLimitMigrationReceipt(path string) (rateLimitMigrationReceipt, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return rateLimitMigrationReceipt{}, err
	}
	var receipt rateLimitMigrationReceipt
	if err = json.Unmarshal(payload, &receipt); err != nil {
		return rateLimitMigrationReceipt{}, err
	}
	if receipt.SchemaVersion != 1 {
		return rateLimitMigrationReceipt{}, errors.New("unsupported migration receipt version")
	}
	return receipt, nil
}

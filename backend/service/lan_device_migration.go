package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/istoreos/quickstart/backend/models"
)

const lanDeviceModelVersion = 1

type lanDeviceMigrationSnapshot struct {
	Items          []*models.LanDeviceMigrationItem
	SourceVersions map[string]string
	AppliedVersion string
}

type lanDeviceMigrationReceipt struct {
	ModelVersion  int               `json:"modelVersion"`
	Mode          string            `json:"mode"`
	SourceVersion string            `json:"sourceVersion"`
	Mappings      map[string]string `json:"mappings"`
}

type lanDeviceMigrationStore interface {
	Read(context.Context) (lanDeviceMigrationSnapshot, error)
	SnapshotReceipt(context.Context) ([]byte, bool, error)
	Commit(context.Context, lanDeviceMigrationReceipt) error
	RestoreReceipt(context.Context, []byte, bool) error
}

type LanDeviceMigrationModule struct {
	mu    sync.Mutex
	store lanDeviceMigrationStore
}

func NewLanDeviceMigrationModule(store lanDeviceMigrationStore) *LanDeviceMigrationModule {
	return &LanDeviceMigrationModule{store: store}
}

func NewDefaultLanDeviceMigrationModule(rules *NetworkRulesModule) *LanDeviceMigrationModule {
	return NewLanDeviceMigrationModule(&systemLanDeviceMigrationStore{rules: rules, receiptPath: defaultLanDeviceMigrationReceiptPath})
}

func (module *LanDeviceMigrationModule) Plan(ctx context.Context) (*models.LanDeviceMigrationResponse, error) {
	_, plan, err := module.plan(ctx)
	if err != nil {
		return nil, err
	}
	return migrationResponse(plan, false, nil), nil
}

func (module *LanDeviceMigrationModule) Apply(ctx context.Context, request *models.LanDeviceMigrationRequest) (*models.LanDeviceMigrationResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	if module == nil || module.store == nil {
		return nil, errors.New("LAN device migration module is unavailable")
	}
	snapshot, plan, err := module.plan(ctx)
	if err != nil {
		return nil, err
	}
	if request == nil || request.ExpectedVersion == "" {
		return migrationResponse(plan, false, &models.DevicePolicyError{Code: "validation_failed", Message: "expectedVersion is required"}), nil
	}
	if request.ExpectedVersion != plan.Version {
		return migrationResponse(plan, false, &models.DevicePolicyError{Code: "conflict", Message: "source configuration changed; run migration preview again"}), nil
	}
	if plan.AlreadyApplied {
		return migrationResponse(plan, false, nil), nil
	}
	if !plan.CanApply {
		return migrationResponse(plan, false, plan.Error), nil
	}
	receiptData, receiptExisted, err := module.store.SnapshotReceipt(ctx)
	if err != nil {
		return migrationResponse(plan, false, &models.DevicePolicyError{Code: "snapshot_failed", Message: err.Error()}), nil
	}
	mappings := make(map[string]string, len(snapshot.Items))
	for _, item := range snapshot.Items {
		mappings[item.ID] = item.Target
	}
	receipt := lanDeviceMigrationReceipt{ModelVersion: lanDeviceModelVersion, Mode: "adopt_in_place", SourceVersion: plan.Version, Mappings: mappings}
	if err := module.store.Commit(ctx, receipt); err != nil {
		restoreErr := module.store.RestoreReceipt(ctx, receiptData, receiptExisted)
		if restoreErr != nil {
			return migrationResponse(plan, false, &models.DevicePolicyError{Code: "recovery_required", Message: combineMigrationErrors("migration apply failed", err, restoreErr)}), nil
		}
		return migrationResponse(plan, false, &models.DevicePolicyError{Code: "rolled_back", Message: combineMigrationErrors("migration apply failed", err, nil)}), nil
	}
	verified, verifyErr := module.store.Read(ctx)
	if verifyErr != nil || verified.AppliedVersion != plan.Version {
		restoreErr := module.store.RestoreReceipt(ctx, receiptData, receiptExisted)
		if restoreErr != nil {
			return migrationResponse(plan, false, &models.DevicePolicyError{Code: "recovery_required", Message: combineMigrationErrors("migration verification failed", verifyErr, restoreErr)}), nil
		}
		return migrationResponse(plan, false, &models.DevicePolicyError{Code: "rolled_back", Message: combineMigrationErrors("migration verification failed", verifyErr, nil)}), nil
	}
	plan.AlreadyApplied = true
	return migrationResponse(plan, true, nil), nil
}

func (module *LanDeviceMigrationModule) plan(ctx context.Context) (lanDeviceMigrationSnapshot, *models.LanDeviceMigrationPlan, error) {
	if module == nil || module.store == nil {
		return lanDeviceMigrationSnapshot{}, nil, errors.New("LAN device migration module is unavailable")
	}
	snapshot, err := module.store.Read(ctx)
	if err != nil {
		return lanDeviceMigrationSnapshot{}, nil, err
	}
	items := cloneMigrationItems(snapshot.Items)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	version := versionLanDeviceMigration(items, snapshot.SourceVersions)
	plan := &models.LanDeviceMigrationPlan{
		ModelVersion:   lanDeviceModelVersion,
		Mode:           "adopt_in_place",
		Version:        version,
		AlreadyApplied: snapshot.AppliedVersion == version,
		Items:          items,
		SnapshotPaths:  []string{"/etc/config/dhcp", "/etc/config/eqos", "/etc/config/firewall", "/etc/config/floatip", "/etc/quickstart/device-profiles.json"},
	}
	seenTargets := map[string]string{}
	for _, item := range items {
		switch item.Disposition {
		case "adopt", "normalize":
			if previous, found := seenTargets[item.Target]; found && previous != item.ID {
				item.Disposition = "conflict"
				item.Reason = "multiple legacy entries resolve to the same target"
			}
			seenTargets[item.Target] = item.ID
		case "conflict":
		case "unresolved":
		default:
			item.Disposition = "unresolved"
			item.Reason = "unknown migration disposition"
		}
		if item.Disposition == "conflict" {
			plan.ConflictCount++
		}
		if item.Disposition == "unresolved" {
			plan.UnresolvedCount++
		}
	}
	plan.CanApply = plan.ConflictCount == 0 && plan.UnresolvedCount == 0
	if !plan.CanApply {
		plan.Error = &models.DevicePolicyError{Code: "needs_attention", Message: "resolve migration conflicts and unknown items before applying"}
	}
	return snapshot, plan, nil
}

func versionLanDeviceMigration(items []*models.LanDeviceMigrationItem, versions map[string]string) string {
	keys := make([]string, 0, len(versions))
	for key := range versions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	payload := struct {
		Items    []*models.LanDeviceMigrationItem `json:"items"`
		Versions [][2]string                      `json:"versions"`
	}{Items: items}
	for _, key := range keys {
		payload.Versions = append(payload.Versions, [2]string{key, versions[key]})
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func cloneMigrationItems(source []*models.LanDeviceMigrationItem) []*models.LanDeviceMigrationItem {
	result := make([]*models.LanDeviceMigrationItem, 0, len(source))
	for _, item := range source {
		if item == nil {
			continue
		}
		copy := *item
		result = append(result, &copy)
	}
	return result
}

func migrationResponse(plan *models.LanDeviceMigrationPlan, changed bool, policyErr *models.DevicePolicyError) *models.LanDeviceMigrationResponse {
	return &models.LanDeviceMigrationResponse{Result: &models.LanDeviceMigrationResult{Plan: plan, Changed: changed, Error: policyErr}}
}

func combineMigrationErrors(message string, cause, recovery error) string {
	parts := []string{message}
	if cause != nil {
		parts = append(parts, cause.Error())
	}
	if recovery != nil {
		parts = append(parts, "restore failed: "+recovery.Error())
	}
	return strings.Join(parts, "; ")
}

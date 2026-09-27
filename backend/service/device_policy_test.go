package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/istoreos/quickstart/backend/models"
)

type fakeDevicePolicyStore struct {
	policy       *models.DevicePolicy
	rules        *models.DevicePolicyRulesResult
	applyCalls   int
	restoreCalls int
	failApply    bool
	failRestore  bool
}

func (store *fakeDevicePolicyStore) Get(context.Context, string) (*models.DevicePolicy, error) {
	copyPolicy := *store.policy
	copyStatic, copySpeed, copyAccess := *store.policy.Static, *store.policy.Speed, *store.policy.Access
	copyPolicy.Static, copyPolicy.Speed, copyPolicy.Access = &copyStatic, &copySpeed, &copyAccess
	copyPolicy.Capabilities = map[string]*models.DevicePolicyCapability{}
	for key, value := range store.policy.Capabilities {
		copyValue := *value
		copyPolicy.Capabilities[key] = &copyValue
	}
	return &copyPolicy, nil
}

func (store *fakeDevicePolicyStore) ListRules(context.Context) (*models.DevicePolicyRulesResult, error) {
	if store.rules == nil {
		return &models.DevicePolicyRulesResult{Static: []*models.LANStaticAssigned{}, Speed: []*models.LANCtrlSpeedLimitItem{}}, nil
	}
	return store.rules, nil
}

func (store *fakeDevicePolicyStore) Backup(context.Context, string, ...*models.DevicePolicy) (devicePolicyBackup, error) {
	return store.Get(context.Background(), store.policy.DeviceID)
}

func (store *fakeDevicePolicyStore) Apply(_ context.Context, request *models.DevicePolicyApplyRequest, _ *models.DevicePolicy) error {
	store.applyCalls++
	switch request.Kind {
	case "static":
		copyValue := *request.Static
		store.policy.Static = &copyValue
	case "speed":
		copyValue := *request.Speed
		store.policy.Speed = &copyValue
	case "access":
		copyValue := *request.Access
		store.policy.Access = &copyValue
	}
	if store.failApply {
		return errors.New("apply interrupted")
	}
	return nil
}

func (store *fakeDevicePolicyStore) Restore(_ context.Context, raw devicePolicyBackup) error {
	store.restoreCalls++
	if store.failRestore {
		return errors.New("restore interrupted")
	}
	policy := raw.(*models.DevicePolicy)
	store.policy = policy
	return nil
}

func availablePolicyStore() *fakeDevicePolicyStore {
	return &fakeDevicePolicyStore{policy: &models.DevicePolicy{
		DeviceID: "mac:aa:bb:cc:dd:ee:ff", DisplayName: "television", MAC: "AA:BB:CC:DD:EE:FF", CurrentIPv4: "192.168.100.20",
		Static: &models.DeviceStaticPolicy{}, Speed: &models.DeviceSpeedPolicy{}, Access: &models.DeviceAccessPolicy{NetworkAccess: true},
		Capabilities: map[string]*models.DevicePolicyCapability{
			"static": {State: "available"}, "speed": {State: "available"}, "access": {State: "available"},
		},
	}}
}

func TestDevicePolicyAppliesAndReturnsAuthoritativePolicy(t *testing.T) {
	t.Parallel()
	store := availablePolicyStore()
	module := newDevicePolicyModuleForTest(store)
	response, err := module.Apply(context.Background(), &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "static", Static: &models.DeviceStaticPolicy{Enabled: true, AssignedIP: "192.168.100.50", BindIP: true},
	})
	if err != nil || response.Result.Error != nil || !response.Result.Changed || response.Result.Policy.Static.AssignedIP != "192.168.100.50" {
		t.Fatalf("response = %#v, err = %v", response, err)
	}
	if store.applyCalls != 1 || store.restoreCalls != 0 {
		t.Fatalf("apply=%d restore=%d", store.applyCalls, store.restoreCalls)
	}
}

func TestDeviceRestrictionsPlanIsReadOnlyAndVersionGuardsApply(t *testing.T) {
	store := availablePolicyStore()
	module := newDevicePolicyModuleForTest(store)
	request := &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "access", Access: &models.DeviceAccessPolicy{NetworkAccess: false},
	}
	planned, err := module.Plan(context.Background(), request)
	if err != nil || planned.Result.Error != nil || !planned.Result.CanApply || store.applyCalls != 0 || planned.Result.Version == "" {
		t.Fatalf("plan=%#v apply=%d err=%v", planned, store.applyCalls, err)
	}
	request.ExpectedVersion = "stale"
	applied, err := module.Apply(context.Background(), request)
	if err != nil || applied.Result.Error == nil || applied.Result.Error.Code != "conflict" || store.applyCalls != 0 {
		t.Fatalf("apply=%#v calls=%d err=%v", applied, store.applyCalls, err)
	}
}

func TestDevicePolicyVersionIgnoresRuntimeObservations(t *testing.T) {
	policy := availablePolicyStore().policy
	policy.RateLimit = &models.RateLimitEnforcement{Provider: "bandix", State: "ready", ObservedAt: "2026-09-26T11:00:00Z"}
	before := devicePolicyVersion(policy)

	policy.CurrentIPv4 = "192.168.100.99"
	policy.RateLimit.ObservedAt = "2026-09-26T11:00:01Z"
	policy.RateLimit.State = "loaded_unverified"
	policy.Capabilities["speed"].Reason = "runtime_changed"
	if after := devicePolicyVersion(policy); after != before {
		t.Fatalf("runtime observation changed policy version: before=%s after=%s", before, after)
	}

	policy.Speed = &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 100}
	if after := devicePolicyVersion(policy); after == before {
		t.Fatal("configured speed change did not change policy version")
	}
}

func TestBandixRestrictionPlanDoesNotClaimEQoSReload(t *testing.T) {
	store := availablePolicyStore()
	store.policy.RateLimit = &models.RateLimitEnforcement{Provider: "bandix", CanApply: true}
	module := newDevicePolicyModuleForTest(store)
	planned, err := module.Plan(context.Background(), &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID,
		Kind:     "speed",
		Speed:    &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 100},
	})
	if err != nil || planned.Result.Error != nil {
		t.Fatalf("plan=%#v err=%v", planned, err)
	}
	if len(planned.Result.ReloadServices) != 0 {
		t.Fatalf("Bandix plan reloads = %v", planned.Result.ReloadServices)
	}
}

func TestDevicePolicyRejectsStaticAddressConflictBeforeWrite(t *testing.T) {
	t.Parallel()
	store := availablePolicyStore()
	store.rules = &models.DevicePolicyRulesResult{Static: []*models.LANStaticAssigned{{AssignedIP: "192.168.100.50", AssignedMac: "AA:BB:CC:DD:EE:01"}}}
	response, err := newDevicePolicyModuleForTest(store).Apply(context.Background(), &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "static", Static: &models.DeviceStaticPolicy{Enabled: true, AssignedIP: "192.168.100.50"},
	})
	if err != nil || response.Result.Error.Code != "conflict" || store.applyCalls != 0 {
		t.Fatalf("response = %#v, apply=%d, err=%v", response, store.applyCalls, err)
	}
}

func TestDevicePolicyRejectsMissingDependencyWithoutWrite(t *testing.T) {
	t.Parallel()
	store := availablePolicyStore()
	store.policy.Capabilities["speed"] = &models.DevicePolicyCapability{State: "not_installed", Reason: "eqos is not installed"}
	response, err := newDevicePolicyModuleForTest(store).Apply(context.Background(), &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "speed", Speed: &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 100, DownloadSpeed: 200},
	})
	if err != nil || response.Result.Error.Code != "dependency_not_installed" || store.applyCalls != 0 {
		t.Fatalf("response = %#v, apply=%d, err=%v", response, store.applyCalls, err)
	}
}

func TestDevicePolicyRollsBackInterruptedApply(t *testing.T) {
	t.Parallel()
	store := availablePolicyStore()
	store.failApply = true
	response, err := newDevicePolicyModuleForTest(store).Apply(context.Background(), &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "access", Access: &models.DeviceAccessPolicy{NetworkAccess: false},
	})
	if err != nil || response.Result.Error.Code != "rolled_back" || response.Result.Transaction.Status != "rolled_back" || store.restoreCalls != 1 || !store.policy.Access.NetworkAccess {
		t.Fatalf("response=%#v restore=%d policy=%#v err=%v", response, store.restoreCalls, store.policy, err)
	}
}

func TestDevicePolicyReportsRecoveryRequiredWhenRollbackFails(t *testing.T) {
	store := availablePolicyStore()
	store.failApply, store.failRestore = true, true
	response, err := newDevicePolicyModuleForTest(store).Apply(context.Background(), &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "speed", IdempotencyKey: "rollback-failure",
		Speed: &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 20},
	})
	if err != nil || response.Result.Error.Code != "recovery_required" || response.Result.Transaction.Status != "recovery_required" || response.Result.Transaction.RecoveryAction != "restore_task_snapshot" {
		t.Fatalf("response = %#v, err = %v", response, err)
	}
}

func TestDevicePolicyIdempotencyPreventsDuplicateWrite(t *testing.T) {
	t.Parallel()
	store := availablePolicyStore()
	module := newDevicePolicyModuleForTest(store)
	request := &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "access", IdempotencyKey: "request-one", Access: &models.DeviceAccessPolicy{NetworkAccess: false},
	}
	first, _ := module.Apply(context.Background(), request)
	second, _ := module.Apply(context.Background(), request)
	if !first.Result.Changed || second.Result.Changed || store.applyCalls != 1 {
		t.Fatalf("first=%#v second=%#v apply=%d", first.Result, second.Result, store.applyCalls)
	}
	conflicting := *request
	conflicting.Access = &models.DeviceAccessPolicy{NetworkAccess: true}
	third, _ := module.Apply(context.Background(), &conflicting)
	if third.Result.Error.Code != "conflict" || store.applyCalls != 1 {
		t.Fatalf("third=%#v apply=%d", third.Result, store.applyCalls)
	}
}

func TestDevicePolicyIdempotencySurvivesModuleRestart(t *testing.T) {
	store := availablePolicyStore()
	path := filepath.Join(t.TempDir(), "transactions.json")
	firstModule := newDevicePolicyModuleForTest(store)
	firstModule.transactions = NewDefaultTaskTransactionJournal()
	firstModule.transactions.path = path
	request := &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "access", IdempotencyKey: "persistent-request",
		Access: &models.DeviceAccessPolicy{NetworkAccess: false},
	}
	first, _ := firstModule.Apply(context.Background(), request)
	restarted := newDevicePolicyModuleForTest(store)
	restarted.transactions = NewDefaultTaskTransactionJournal()
	restarted.transactions.path = path
	replayed, _ := restarted.Apply(context.Background(), request)
	if first.Result.Transaction.Status != "committed" || !replayed.Result.Transaction.Replayed || replayed.Result.Changed || store.applyCalls != 1 {
		t.Fatalf("first=%#v replay=%#v calls=%d", first.Result, replayed.Result, store.applyCalls)
	}
}

func TestDevicePolicyValidationRejectsInvalidSpeed(t *testing.T) {
	t.Parallel()
	store := availablePolicyStore()
	response, err := newDevicePolicyModuleForTest(store).Apply(context.Background(), &models.DevicePolicyApplyRequest{
		DeviceID: store.policy.DeviceID, Kind: "speed", Speed: &models.DeviceSpeedPolicy{Enabled: true},
	})
	if err != nil || response.Result.Error.Code != "validation_failed" || store.applyCalls != 0 {
		t.Fatalf("response=%#v apply=%d err=%v", response, store.applyCalls, err)
	}
}

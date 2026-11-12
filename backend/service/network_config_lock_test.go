package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func TestMultiLANTransactionKeysAndLegacyReplay(t *testing.T) {
	journal := newMemoryTaskTransactionJournal()
	legacy, err := journal.Begin(context.Background(), "profile", "same-key", "payload")
	if err != nil {
		t.Fatal(err)
	}
	if err := journal.Advance(context.Background(), legacy.Transaction, "verify", "committed", ""); err != nil {
		t.Fatal(err)
	}
	replayed, err := journal.Begin(withLANScope(context.Background(), &models.LANScope{NetworkID: "lan", Version: "new-scope-version"}), "profile", "same-key", "payload")
	if err != nil || !replayed.Replay || replayed.Conflict {
		t.Fatalf("legacy replay: %#v %v", replayed, err)
	}
	for _, network := range []string{"office", "guest"} {
		ctx := inventoryLANContext(network)
		begin, err := journal.Begin(ctx, "profile", "same-key", network)
		if err != nil || begin.Replay || begin.Conflict || begin.Transaction.NetworkID != network {
			t.Fatalf("transaction namespace: %#v %v", begin, err)
		}
		if err := journal.Advance(ctx, begin.Transaction, "verify", "committed", ""); err != nil {
			t.Fatal(err)
		}
		if err := journal.Advance(inventoryLANContext("other"), begin.Transaction, "verify", "committed", ""); err == nil {
			t.Fatal("transaction advanced under another network")
		}
		anonymous, err := journal.Begin(ctx, "profile", "", "payload")
		if err != nil {
			t.Fatal(err)
		}
		if err := journal.Advance(ctx, anonymous.Transaction, "verify", "committed", ""); err != nil || anonymous.Transaction.Status != "committed" {
			t.Fatalf("anonymous transaction lost scope: %#v %v", anonymous, err)
		}
	}
}

func TestMultiLANPlansRejectAnotherNetworkOrBinding(t *testing.T) {
	store := &fakeLanDHCPSettingsStore{snapshot: testLanDHCPSettingsSnapshot()}
	module := NewLanDHCPSettingsModule(store)
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "office", Version: "binding-a"})
	desired := *store.snapshot.Settings
	desired.LeaseTime = "24h"
	plan, err := module.Plan(ctx, &models.LanDHCPSettingsApplyRequest{Settings: &desired})
	if err != nil || !plan.Result.CanApply {
		t.Fatalf("plan: %#v %v", plan, err)
	}
	for _, scope := range []*models.LANScope{{NetworkID: "guest", Version: "binding-a"}, {NetworkID: "office", Version: "binding-b"}} {
		response, err := module.Apply(withLANScope(context.Background(), scope), &models.LanDHCPSettingsApplyRequest{Settings: &desired, ExpectedVersion: plan.Result.Version})
		if err != nil || response.Result.Error == nil || response.Result.Error.Code != "conflict" || store.applies != 0 {
			t.Fatalf("cross-scope plan accepted: %#v %v applies=%d", response, err, store.applies)
		}
	}
}

func TestMultiLANJournalFailureStopsARestrictionBeforeWriting(t *testing.T) {
	store := &fakeDevicePolicyStore{policy: &models.DevicePolicy{
		DeviceID: "mac:02:00:00:00:04:01", MAC: "02:00:00:00:04:01",
		Static: &models.DeviceStaticPolicy{}, Speed: &models.DeviceSpeedPolicy{}, Access: &models.DeviceAccessPolicy{NetworkAccess: true},
		Capabilities: map[string]*models.DevicePolicyCapability{"access": {State: models.CapabilityAvailable}},
	}}
	module := newDevicePolicyModuleForTest(store)
	journal := NewDefaultTaskTransactionJournal()
	journal.path = filepath.Join(t.TempDir(), "transactions.json")
	writes := 0
	journal.persist = func(path string, raw []byte) error {
		writes++
		if writes > 1 {
			return errors.New("journal storage unavailable")
		}
		return persistClassificationOverrides(path, raw)
	}
	module.transactions = journal
	response, err := module.Apply(inventoryLANContext("office"), &models.DevicePolicyApplyRequest{DeviceID: store.policy.DeviceID, Kind: "access", IdempotencyKey: "journal-failure", Access: &models.DeviceAccessPolicy{NetworkAccess: false}})
	if err != nil || response.Result.Error == nil || store.applyCalls != 0 {
		t.Fatalf("configuration changed despite an unavailable transaction journal: %#v %v writes=%d", response, err, store.applyCalls)
	}
}

func TestMultiLANJournalFailureAfterEffectsRequiresRecovery(t *testing.T) {
	for _, failAt := range []int{4, 5} {
		t.Run(strconv.Itoa(failAt), func(t *testing.T) {
			store := &fakeDevicePolicyStore{policy: &models.DevicePolicy{DeviceID: "mac:02:00:00:00:04:01", MAC: "02:00:00:00:04:01", Static: &models.DeviceStaticPolicy{}, Speed: &models.DeviceSpeedPolicy{}, Access: &models.DeviceAccessPolicy{NetworkAccess: true}, Capabilities: map[string]*models.DevicePolicyCapability{"access": {State: models.CapabilityAvailable}}}}
			module := newDevicePolicyModuleForTest(store)
			journal := NewDefaultTaskTransactionJournal()
			journal.path = filepath.Join(t.TempDir(), "transactions.json")
			writes := 0
			journal.persist = func(path string, raw []byte) error {
				writes++
				if writes >= failAt {
					return errors.New("journal storage unavailable")
				}
				return persistClassificationOverrides(path, raw)
			}
			module.transactions = journal
			response, err := module.Apply(inventoryLANContext("office"), &models.DevicePolicyApplyRequest{DeviceID: store.policy.DeviceID, Kind: "access", IdempotencyKey: "post-apply-journal", Access: &models.DeviceAccessPolicy{NetworkAccess: false}})
			if err != nil || response.Result.Error == nil || response.Result.Error.Code != "recovery_required" || response.Result.Transaction.Status != "recovery_required" || store.applyCalls != 1 || store.policy.Access.NetworkAccess {
				t.Fatalf("post-apply journal failure misreported: %#v %v applies=%d", response, err, store.applyCalls)
			}
		})
	}
}

type recoveryRequiredDHCPStore struct {
	snapshot lanDHCPSettingsSnapshot
	applies  int
}

func (store *recoveryRequiredDHCPStore) Read(context.Context) (lanDHCPSettingsSnapshot, error) {
	return store.snapshot, nil
}
func (store *recoveryRequiredDHCPStore) Apply(context.Context, lanDHCPSettingsExecutionPlan) error {
	store.applies++
	return errLanDHCPRecoveryRequired
}

func TestMultiLANFailedTransactionReplayDoesNotReportSuccess(t *testing.T) {
	store := &recoveryRequiredDHCPStore{snapshot: testLanDHCPSettingsSnapshot()}
	module := NewLanDHCPSettingsModule(store)
	desired := *store.snapshot.Settings
	desired.LeaseTime = "24h"
	request := &models.LanDHCPSettingsApplyRequest{Settings: &desired, IdempotencyKey: "recovering-task"}
	first, err := module.Apply(context.Background(), request)
	if err != nil || first.Result.Error == nil || first.Result.Transaction.Status != "recovery_required" {
		t.Fatalf("initial failure not recorded: %#v %v", first, err)
	}
	retry, err := module.Apply(context.Background(), request)
	if err != nil || retry.Result.Error == nil || retry.Result.Error.Code != "recovery_required" || retry.Result.CanApply || store.applies != 1 {
		t.Fatalf("failed transaction replay appeared successful: %#v %v applies=%d", retry, err, store.applies)
	}
}

func TestMultiLANWriteLockRevalidatesScopeAfterWaiting(t *testing.T) {
	module := testLANScopeModule()
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "office", Version: "old-binding"})
	ctx = context.WithValue(ctx, lanScopeResolverKey{}, module)
	if _, release, err := lockLANWrite(ctx); err == nil {
		release()
		t.Fatal("write accepted a changed binding")
	}
}

func TestNetworkConfigurationLockIsReentrantAndCancelable(t *testing.T) {
	dir := t.TempDir()
	ctx, release, err := lockNetworkConfiguration(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	_, nested, err := lockNetworkConfiguration(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	nested()
	deadline, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, unlock, err := lockNetworkConfiguration(deadline, dir); err == nil {
		unlock()
		t.Fatal("nested release unlocked the owner's lease")
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	release()
	_, unlock, err := lockNetworkConfiguration(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}

func TestNetworkConfigurationLockWorksAcrossProcesses(t *testing.T) {
	dir := t.TempDir()
	_, release, err := lockNetworkConfiguration(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, "-test.run=^TestNetworkConfigurationLockProcessHelper$")
	command.Env = append(os.Environ(), "QUICKSTART_LOCK_HELPER_DIR="+dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("cross-process lock failed: %s %v", output, err)
	}
}

func TestNetworkConfigurationLockProcessHelper(t *testing.T) {
	dir := os.Getenv("QUICKSTART_LOCK_HELPER_DIR")
	if dir == "" {
		t.Skip("subprocess-only lock test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, release, err := lockNetworkConfiguration(ctx, dir); err == nil {
		release()
		t.Fatal("another process entered the held configuration lock")
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
}

func TestMultiLANFailedRollbackCannotEraseAnotherSuccessfulWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dhcp")
	if err := os.WriteFile(path, []byte(`{"lan":0,"office":0}`), 0600); err != nil {
		t.Fatal(err)
	}
	firstWrote, secondStarted := make(chan struct{}), make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		_, release, err := lockNetworkConfiguration(context.Background(), dir)
		if err != nil {
			t.Error(err)
			return
		}
		defer release()
		before, err := os.ReadFile(path)
		if err != nil {
			t.Error(err)
			return
		}
		if err := persistClassificationOverrides(path, []byte(`{"lan":1,"office":0}`)); err != nil {
			t.Error(err)
			return
		}
		close(firstWrote)
		<-secondStarted
		time.Sleep(30 * time.Millisecond)
		// Simulate a failed service verification, then restore while owning the lock.
		if err := persistClassificationOverrides(path, before); err != nil {
			t.Error(err)
		}
	}()
	go func() {
		defer workers.Done()
		<-firstWrote
		close(secondStarted)
		_, release, err := lockNetworkConfiguration(context.Background(), dir)
		if err != nil {
			t.Error(err)
			return
		}
		defer release()
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Error(err)
			return
		}
		var values map[string]int
		if err := json.Unmarshal(raw, &values); err != nil {
			t.Error(err)
			return
		}
		values["office"] = 1
		raw, _ = json.Marshal(values)
		if err := persistClassificationOverrides(path, raw); err != nil {
			t.Error(err)
		}
	}()
	workers.Wait()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]int
	if err := json.Unmarshal(raw, &values); err != nil {
		t.Fatal(err)
	}
	if values["lan"] != 0 || values["office"] != 1 {
		t.Fatalf("successful change lost during another LAN's rollback: %s", raw)
	}
}

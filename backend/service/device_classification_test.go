package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func TestDeviceClassificationStorePersistentAndBootScopes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device-classifications.json")
	store := NewDeviceClassificationOverrideStore(path)

	changed, err := store.Set("mac:aa:bb:cc:dd:ee:01", "persistent", "tv")
	if err != nil || !changed {
		t.Fatalf("persistent set = %v, %v", changed, err)
	}
	changed, err = store.Set("duid:one:one", "boot", "gaming")
	if err != nil || !changed {
		t.Fatalf("boot set = %v, %v", changed, err)
	}
	changed, err = store.Set("mac:aa:bb:cc:dd:ee:01", "persistent", "tv")
	if err != nil || changed {
		t.Fatalf("idempotent set = %v, %v", changed, err)
	}

	reloaded := NewDeviceClassificationOverrideStore(path)
	if category, ok, err := reloaded.Get("mac:aa:bb:cc:dd:ee:01", "persistent"); err != nil || !ok || category != "tv" {
		t.Fatalf("reloaded persistent = %q, %v, %v", category, ok, err)
	}
	if _, ok, err := reloaded.Get("duid:one:one", "boot"); err != nil || ok {
		t.Fatalf("reloaded boot override = %v, %v", ok, err)
	}
	changed, err = reloaded.Reset("mac:aa:bb:cc:dd:ee:01", "persistent")
	if err != nil || !changed {
		t.Fatalf("reset = %v, %v", changed, err)
	}
	changed, err = reloaded.Reset("mac:aa:bb:cc:dd:ee:01", "persistent")
	if err != nil || changed {
		t.Fatalf("idempotent reset = %v, %v", changed, err)
	}
}

func TestDeviceClassificationStoreWriteFailureDoesNotMutateMemory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device-classifications.json")
	store := NewDeviceClassificationOverrideStore(path)
	if _, _, err := store.Get("mac:one", "persistent"); err != nil {
		t.Fatal(err)
	}
	store.persist = func(string, []byte) error { return errors.New("disk full") }

	if changed, err := store.Set("mac:one", "persistent", "tv"); err == nil || changed {
		t.Fatalf("failed set = %v, %v", changed, err)
	}
	if _, ok, err := store.Get("mac:one", "persistent"); err != nil || ok {
		t.Fatalf("failed set mutated state = %v, %v", ok, err)
	}
}

func TestDeviceClassificationStoreRejectsCorruptFileWithoutOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "device-classifications.json")
	original := []byte("not-json")
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewDeviceClassificationOverrideStore(path)
	if _, _, err := store.Get("mac:one", "persistent"); err == nil {
		t.Fatal("expected corrupt store error")
	}
	if changed, err := store.Set("mac:one", "persistent", "tv"); err == nil || changed {
		t.Fatalf("corrupt set = %v, %v", changed, err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(original) {
		t.Fatalf("corrupt file was changed: %q, %v", got, err)
	}
}

func TestDeviceClassificationModuleSetResetAndValidation(t *testing.T) {
	now := time.Date(2026, 9, 24, 4, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.1.10", Hostname: "client", Online: true}},
	}}}
	inventory := newTestDeviceInventory(t, source, &now, 100)
	module := NewDeviceClassificationModule(inventory)
	deviceID := "mac:aa:bb:cc:dd:ee:01"

	response, err := module.Apply(context.Background(), &models.DeviceClassificationApplyRequest{DeviceID: deviceID, Action: "set", Category: "tv"})
	requireClassificationResult(t, response, err, "tv", "manual", true, true)
	response, err = module.Apply(context.Background(), &models.DeviceClassificationApplyRequest{DeviceID: deviceID, Action: "set", Category: "tv"})
	requireClassificationResult(t, response, err, "tv", "manual", true, false)

	response, err = module.Get(context.Background(), deviceID)
	requireClassificationResult(t, response, err, "tv", "manual", true, false)
	response, err = module.Apply(context.Background(), &models.DeviceClassificationApplyRequest{DeviceID: deviceID, Action: "reset"})
	requireClassificationResult(t, response, err, "computer", "fallback", false, true)

	response, err = module.Apply(context.Background(), &models.DeviceClassificationApplyRequest{DeviceID: deviceID, Action: "set", Category: "spaceship"})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "validation_failed" {
		t.Fatalf("invalid category response = %#v, %v", response, err)
	}
	response, err = module.Get(context.Background(), "mac:missing")
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "not_found" {
		t.Fatalf("missing device response = %#v, %v", response, err)
	}
}

func TestDeviceClassificationModuleBootOverrideIsNotPersisted(t *testing.T) {
	now := time.Date(2026, 9, 24, 4, 0, 0, 0, time.UTC)
	snapshot := deviceInventorySourceSnapshot{DHCPv6: []deviceInventoryObservation{{
		DUID: "00020000ab11abcd", IAID: "1234abcd", IPv6: "fd00::10", Hostname: "client-v6",
	}}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	module := NewDeviceClassificationModule(inventory)
	deviceID := "duid:00020000ab11abcd:1234abcd"
	response, err := module.Apply(context.Background(), &models.DeviceClassificationApplyRequest{DeviceID: deviceID, Action: "set", Category: "gaming"})
	requireClassificationResult(t, response, err, "gaming", "manual", true, true)
	if response.Result.Override.Scope != "boot" {
		t.Fatalf("override scope = %q", response.Result.Override.Scope)
	}

	restarted := newDeviceInventoryModuleForTest(
		&fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}},
		inventory.historyPath, "boot-test", func() time.Time { return now }, 100,
	)
	response, err = NewDeviceClassificationModule(restarted).Get(context.Background(), deviceID)
	requireClassificationResult(t, response, err, "computer", "fallback", false, false)
}

func requireClassificationResult(t *testing.T, response *models.DeviceClassificationResponse, err error, category, source string, active, changed bool) {
	t.Helper()
	if err != nil || response == nil || response.Result == nil || response.Result.Error != nil {
		t.Fatalf("classification response = %#v, %v", response, err)
	}
	result := response.Result
	if result.Classification == nil || result.Override == nil || result.Classification.Category != category || result.Classification.Source != source || result.Override.Active != active || result.Changed != changed {
		t.Fatalf("classification result = %#v", result)
	}
}

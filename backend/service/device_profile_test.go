package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func TestDeviceProfileStoreMigratesLegacyClassification(t *testing.T) {
	directory := t.TempDir()
	legacyPath := filepath.Join(directory, "device-classifications.json")
	profilePath := filepath.Join(directory, "device-profiles.json")
	legacy, _ := json.Marshal(classificationOverrideFile{Version: 1, Records: map[string]string{"mac:one": "network"}})
	if err := os.WriteFile(legacyPath, legacy, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewDeviceProfileStore(profilePath, legacyPath)
	profile, ok, err := store.GetProfile("mac:one", "persistent")
	if err != nil || !ok || profile.Category != "network" {
		t.Fatalf("migrated profile = %#v, %v, %v", profile, ok, err)
	}
	data, err := os.ReadFile(profilePath)
	if err != nil || !strings.Contains(string(data), `"version":3`) {
		t.Fatalf("profile migration file = %s, %v", data, err)
	}
}

func TestDeviceIconRegistryHasThirtyStableUniqueKeys(t *testing.T) {
	if len(deviceIconRegistry) != 30 {
		t.Fatalf("icon count = %d", len(deviceIconRegistry))
	}
	seen := map[string]bool{}
	for _, icon := range deviceIconRegistry {
		if icon.Key == "" || icon.Label == "" || icon.Category == "" || seen[icon.Key] || !validDeviceIconKey(icon.Key) {
			t.Fatalf("invalid icon descriptor: %#v", icon)
		}
		seen[icon.Key] = true
	}
}

func TestDeviceProfileBrandCategoryAndIconPreferencesAreIndependent(t *testing.T) {
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	snapshot := deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{{
		MAC: "02:00:00:00:00:24", IPv4: "192.168.1.24", Online: true,
	}}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	module := NewDeviceProfileModule(inventory)
	deviceID, brand, network, manual, icon := "mac:02:00:00:00:00:24", "ASUS", "network", "manual", "laptop"
	response, err := module.Apply(context.Background(), &models.DeviceProfileApplyRequest{
		DeviceID: deviceID, Action: "patch", Brand: &brand, Category: &network,
	})
	if err != nil || response.Result.Error != nil || response.Result.Profile.Icon.ResolvedKey != "brand:asus:network" || response.Result.Profile.Icon.AssetKey != "network" {
		t.Fatalf("ASUS network visual = %#v, %v", response, err)
	}
	response, err = module.Apply(context.Background(), &models.DeviceProfileApplyRequest{
		DeviceID: deviceID, Action: "patch", IconMode: &manual, IconKey: &icon,
	})
	if err != nil || response.Result.Error != nil || response.Result.Profile.Icon.ResolvedKey != "manual:laptop" {
		t.Fatalf("manual visual = %#v, %v", response, err)
	}
	computer := "computer"
	response, err = module.Apply(context.Background(), &models.DeviceProfileApplyRequest{DeviceID: deviceID, Action: "patch", Category: &computer})
	if err != nil || response.Result.Profile.Icon.ResolvedKey != "manual:laptop" {
		t.Fatalf("category overwrote manual icon: %#v, %v", response, err)
	}
	response, err = module.Apply(context.Background(), &models.DeviceProfileApplyRequest{DeviceID: deviceID, Action: "reset_icon"})
	if err != nil || response.Result.Profile.Icon.ResolvedKey != "brand:asus:computer" || response.Result.Profile.Icon.AssetKey != "computer" {
		t.Fatalf("automatic ASUS computer visual = %#v, %v", response, err)
	}
	response, err = module.Apply(context.Background(), &models.DeviceProfileApplyRequest{DeviceID: deviceID, Action: "reset_brand"})
	if err != nil || response.Result.Profile.ManualCategory != "computer" || response.Result.Profile.ManualBrand != "" || response.Result.Profile.Icon.ResolvedKey != "category:computer" {
		t.Fatalf("brand reset changed another preference: %#v, %v", response, err)
	}
}

func TestDeviceProfileManualIconPersistsAcrossRestart(t *testing.T) {
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	snapshot := deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{{MAC: "02:00:00:00:00:25", IPv4: "192.168.1.25", Online: true}}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	manual, icon := "manual", "smart-speaker"
	response, err := NewDeviceProfileModule(inventory).Apply(context.Background(), &models.DeviceProfileApplyRequest{
		DeviceID: "mac:02:00:00:00:00:25", Action: "patch", IconMode: &manual, IconKey: &icon,
	})
	if err != nil || response.Result.Error != nil {
		t.Fatalf("patch = %#v, %v", response, err)
	}
	restarted := newDeviceInventoryModuleForTest(&fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, inventory.historyPath, "boot-test", func() time.Time { return now }, 100)
	response, err = NewDeviceProfileModule(restarted).Get(context.Background(), "mac:02:00:00:00:00:25")
	if err != nil || response.Result.Profile.Icon.PreferenceKey != icon || response.Result.Profile.Icon.Mode != "manual" {
		t.Fatalf("restarted icon = %#v, %v", response, err)
	}
}

func TestDeviceProfileRejectsInvalidBrandAndIcon(t *testing.T) {
	store := NewDeviceClassificationOverrideStore(filepath.Join(t.TempDir(), "profiles.json"))
	badBrand, manual, badIcon := "bad\nbrand", "manual", "not-an-icon"
	if changed, err := store.PatchProfileFields("mac:one", "persistent", deviceProfilePatch{Brand: &badBrand}); err == nil || changed {
		t.Fatalf("invalid brand = %v, %v", changed, err)
	}
	if changed, err := store.PatchProfileFields("mac:one", "persistent", deviceProfilePatch{IconMode: &manual, IconKey: &badIcon}); err == nil || changed {
		t.Fatalf("invalid icon = %v, %v", changed, err)
	}
}

func TestDeviceProfileStoreValidatesAliasAndCapacity(t *testing.T) {
	store := NewDeviceClassificationOverrideStore(filepath.Join(t.TempDir(), "profiles.json"))
	store.limit = 1
	alias := " 客厅电视 "
	changed, err := store.PatchProfile("mac:one", "persistent", &alias, nil)
	if err != nil || !changed {
		t.Fatalf("patch alias = %v, %v", changed, err)
	}
	profile, ok, err := store.GetProfile("mac:one", "persistent")
	if err != nil || !ok || profile.Alias != "客厅电视" {
		t.Fatalf("profile = %#v, %v, %v", profile, ok, err)
	}
	invalid := "bad\nname"
	if changed, err := store.PatchProfile("mac:one", "persistent", &invalid, nil); err == nil || changed {
		t.Fatalf("control alias = %v, %v", changed, err)
	}
	tooLong := strings.Repeat("设", deviceAliasMaxRunes+1)
	if changed, err := store.PatchProfile("mac:one", "persistent", &tooLong, nil); err == nil || changed {
		t.Fatalf("long alias = %v, %v", changed, err)
	}
	second := "书房电脑"
	if changed, err := store.PatchProfile("mac:two", "persistent", &second, nil); err == nil || changed {
		t.Fatalf("capacity patch = %v, %v", changed, err)
	}
}

func TestDeviceProfileModulePatchResetAndRestart(t *testing.T) {
	now := time.Date(2026, 9, 24, 11, 0, 0, 0, time.UTC)
	snapshot := deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{{
		MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.1.10", Hostname: "client-host", Online: true,
	}}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	inventory.cacheTTL = time.Minute
	module := NewDeviceProfileModule(inventory)
	alias, category := "客厅电视", "tv"
	response, err := module.Apply(context.Background(), &models.DeviceProfileApplyRequest{
		DeviceID: "mac:aa:bb:cc:dd:ee:01", Action: "patch", Alias: &alias, Category: &category,
	})
	requireDeviceProfile(t, response, err, alias, "client-host", "tv", true)

	restarted := newDeviceInventoryModuleForTest(
		&fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}},
		inventory.historyPath, "boot-test", func() time.Time { return now }, 100,
	)
	response, err = NewDeviceProfileModule(restarted).Get(context.Background(), "mac:aa:bb:cc:dd:ee:01")
	requireDeviceProfile(t, response, err, alias, "client-host", "tv", false)
	response, err = NewDeviceProfileModule(restarted).Apply(context.Background(), &models.DeviceProfileApplyRequest{
		DeviceID: "mac:aa:bb:cc:dd:ee:01", Action: "reset",
	})
	requireDeviceProfile(t, response, err, "", "client-host", "computer", true)
}

func TestDeviceProfileTransactionIsIdempotentAndKeepsUnicodeOutOfDHCP(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	snapshot := deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:28", IPv4: "192.168.1.28", Hostname: "safe-host", Online: true}}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	module := NewDeviceProfileModule(inventory)
	alias := "书房电脑"
	request := &models.DeviceProfileApplyRequest{DeviceID: "mac:aa:bb:cc:dd:ee:28", IdempotencyKey: "profile-request-1", Action: "patch", Alias: &alias}
	first, firstErr := module.Apply(context.Background(), request)
	second, secondErr := module.Apply(context.Background(), request)
	if firstErr != nil || secondErr != nil || first.Result.Transaction.Status != "committed" || !second.Result.Transaction.Replayed || second.Result.Changed {
		t.Fatalf("first=%#v second=%#v errors=%v/%v", first.Result, second.Result, firstErr, secondErr)
	}
	if second.Result.Profile.Alias != alias || second.Result.Profile.OriginalHostname != "safe-host" {
		t.Fatalf("profile and DHCP hostname mixed: %#v", second.Result.Profile)
	}
	conflicting := *request
	otherAlias := "客厅电脑"
	conflicting.Alias = &otherAlias
	third, _ := module.Apply(context.Background(), &conflicting)
	if third.Result.Error == nil || third.Result.Error.Code != "conflict" {
		t.Fatalf("conflict = %#v", third.Result)
	}
}

func TestDeviceProfileAtomicWriteFailureReportsRolledBack(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 30, 0, 0, time.UTC)
	snapshot := deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:29", IPv4: "192.168.1.29", Hostname: "safe-host", Online: true}}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	// Load the store before replacing its atomic persistence seam.
	if _, err := inventory.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	inventory.classificationOverrides.persist = func(string, []byte) error { return errors.New("disk full") }
	alias := "卧室电脑"
	response, err := NewDeviceProfileModule(inventory).Apply(context.Background(), &models.DeviceProfileApplyRequest{
		DeviceID: "mac:aa:bb:cc:dd:ee:29", IdempotencyKey: "profile-disk-full", Action: "patch", Alias: &alias,
	})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "rolled_back" || response.Result.Transaction.Status != "rolled_back" {
		t.Fatalf("response = %#v, err = %v", response, err)
	}
	record, ok, readErr := inventory.classificationOverrides.GetProfile("mac:aa:bb:cc:dd:ee:29", "persistent")
	if readErr != nil || ok || record.Alias != "" {
		t.Fatalf("profile changed after atomic write failure: %#v, %v, %v", record, ok, readErr)
	}
}

func TestDeviceProfileBootAliasIsNotPersisted(t *testing.T) {
	now := time.Date(2026, 9, 24, 11, 0, 0, 0, time.UTC)
	snapshot := deviceInventorySourceSnapshot{DHCPv6: []deviceInventoryObservation{{
		DUID: "00020000ab11abcd", IAID: "1234abcd", IPv6: "fd00::10", Hostname: "client-v6",
	}}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	alias := "临时设备"
	response, err := NewDeviceProfileModule(inventory).Apply(context.Background(), &models.DeviceProfileApplyRequest{
		DeviceID: "duid:00020000ab11abcd:1234abcd", Action: "patch", Alias: &alias,
	})
	requireDeviceProfile(t, response, err, alias, "client-v6", "computer", true)
	if response.Result.Profile.Scope != "boot" {
		t.Fatalf("scope = %q", response.Result.Profile.Scope)
	}
	restarted := newDeviceInventoryModuleForTest(
		&fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}},
		inventory.historyPath, "boot-test", func() time.Time { return now }, 100,
	)
	response, err = NewDeviceProfileModule(restarted).Get(context.Background(), "duid:00020000ab11abcd:1234abcd")
	requireDeviceProfile(t, response, err, "", "client-v6", "computer", false)
}

func TestDeviceProfileConcurrentUpdatesRemainReadable(t *testing.T) {
	store := NewDeviceClassificationOverrideStore(filepath.Join(t.TempDir(), "profiles.json"))
	var group sync.WaitGroup
	for index := 0; index < 32; index++ {
		group.Add(1)
		go func(value string) {
			defer group.Done()
			_, _ = store.PatchProfile("mac:one", "persistent", &value, nil)
		}(strings.Repeat("设备", index%8+1))
	}
	group.Wait()
	if _, ok, err := NewDeviceClassificationOverrideStore(store.path).GetProfile("mac:one", "persistent"); err != nil || !ok {
		t.Fatalf("concurrent profile is unreadable: %v, %v", ok, err)
	}
}

func TestCorruptDeviceProfileStoreDegradesInventoryWithoutHidingDevices(t *testing.T) {
	now := time.Date(2026, 9, 24, 11, 0, 0, 0, time.UTC)
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.1.10", Online: true}},
	}}}, &now, 100)
	if err := os.WriteFile(inventory.classificationOverrides.path, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	response, err := inventory.Snapshot(context.Background())
	if err != nil || response == nil || response.Result == nil || len(response.Result.Devices) != 1 {
		t.Fatalf("degraded inventory = %#v, %v", response, err)
	}
	if response.Result.Health.State != "partial" || !containsString(response.Result.Health.Reasons, "device_profile_store_invalid") {
		t.Fatalf("profile health = %#v", response.Result.Health)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func requireDeviceProfile(t *testing.T, response *models.DeviceProfileResponse, err error, alias, hostname, category string, changed bool) {
	t.Helper()
	if err != nil || response == nil || response.Result == nil || response.Result.Error != nil || response.Result.Profile == nil {
		t.Fatalf("profile response = %#v, %v", response, err)
	}
	profile := response.Result.Profile
	if profile.Alias != alias || profile.OriginalHostname != hostname || profile.Classification == nil || profile.Classification.Category != category || response.Result.Changed != changed {
		t.Fatalf("profile result = %#v", response.Result)
	}
}

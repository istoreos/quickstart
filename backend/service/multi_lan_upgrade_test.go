package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

// These files are written by the pinned pre-multi-LAN backend, not recreated
// from today's structures. The old reader runs again after this test.
func TestMultiLANReadsPinnedOldDataWithoutCopyingToAdditionalLAN(t *testing.T) {
	base := os.Getenv("QUICKSTART_MULTI_LAN_UPGRADE_BASE")
	if base == "" {
		t.Skip("requires fixture exported by pinned old backend")
	}
	raw, err := os.ReadFile(filepath.Join(base, "seed-time"))
	if err != nil {
		t.Fatal(err)
	}
	now, err := time.Parse(time.RFC3339Nano, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	id := "mac:02:00:00:00:00:55"
	snapshot := deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{{NetworkID: "lan", MAC: "02:00:00:00:00:55", IPv4: "192.168.100.55", Online: true}, {NetworkID: "office", MAC: "02:00:00:00:00:55", IPv4: "192.0.2.55", Online: true}}, NetworkDevices: map[string]string{"lan": "br-lan", "office": "br-office"}}
	inventory := newDeviceInventoryModuleForTest(&fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, filepath.Join(base, "inventory.json"), "old-boot", func() time.Time { return now }, 2048)
	ctx := inventoryLANContext("lan")
	other := inventoryLANContext("office")
	profile := NewDeviceProfileModule(inventory)
	existing, err := profile.Get(ctx, id)
	if err != nil || existing.Result.Profile.Alias != "旧版设备资料" || existing.Result.Profile.Icon.PreferenceKey != "laptop" {
		t.Fatal("default profile changed", err, existing)
	}
	empty, err := profile.Get(other, id)
	if err != nil || empty.Result.Profile.Alias != "" {
		t.Fatal("default profile copied to office", err, empty)
	}
	alias := "office 独立资料"
	changed, err := profile.Apply(other, &models.DeviceProfileApplyRequest{DeviceID: id, Action: "patch", Alias: &alias})
	if err != nil || changed.Result.Error != nil {
		t.Fatal(err, changed)
	}
	repeated, err := profile.Get(ctx, id)
	if err != nil || repeated.Result.Profile.Alias != "旧版设备资料" {
		t.Fatal("office overwrote default profile", err, repeated)
	}
	groups := newJSONDeviceGroupStore(filepath.Join(base, "groups.json"))
	existingGroups, err := groups.Read(ctx)
	if err != nil || len(existingGroups.Groups) != 1 || existingGroups.Groups[0].Policy.Speed.UploadSpeed != 3 || len(existingGroups.Groups[0].Policy.Schedules) != 1 {
		t.Fatal("old group or schedule lost", err, existingGroups)
	}
	otherGroups, err := groups.Read(other)
	if err != nil || len(otherGroups.Groups) != 0 {
		t.Fatal("groups copied to another LAN", err)
	}
	insights := NewTrafficInsightsModule(newTrafficInsightsFileStore(filepath.Join(base, "traffic.json")), nil)
	insights.now = func() time.Time { return now }
	usage, err := insights.Get(ctx, id, "today")
	if err != nil || usage.Result.UploadBytes != 1250 || usage.Result.DownloadBytes != 3750 || usage.Result.Quota == nil || usage.Result.Quota.Quota.Period != "weekly" {
		t.Fatal("old quota/history lost", err, usage)
	}
	isolated, err := insights.Get(other, id, "today")
	if err != nil || isolated.Result.UploadBytes != 0 || isolated.Result.DownloadBytes != 0 || isolated.Result.Quota != nil && isolated.Result.Quota.Quota != nil && isolated.Result.Quota.Quota.Enabled {
		t.Fatal("default quota/history copied", err, isolated)
	}
}

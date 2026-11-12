package service

import (
	"context"
	"github.com/istoreos/quickstart/backend/models"
	"path/filepath"
	"testing"
	"time"
)

type scopedGroupRecorder struct {
	networks []string
	access   []bool
}

func (recorder *scopedGroupRecorder) Apply(ctx context.Context, policy *models.EffectiveGroupPolicy) error {
	recorder.networks = append(recorder.networks, selectedLAN(ctx))
	recorder.access = append(recorder.access, policy.NetworkAccess)
	return nil
}
func TestMultiLANGroupsKeepRuntimeAndRejectForeignMembers(t *testing.T) {
	now := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	const mac = "02:00:00:00:00:55"
	const id = "mac:02:00:00:00:00:55"
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{NetworkID: "lan", MAC: mac, IPv4: "192.0.2.55"}, {NetworkID: "office", MAC: mac, IPv4: "198.51.100.55"}, {NetworkID: "office", MAC: "02:00:00:00:00:66", IPv4: "198.51.100.66"}}}}}, &now, 100)
	recorder := &scopedGroupRecorder{}
	module := NewDeviceGroupModule(newJSONDeviceGroupStore(filepath.Join(t.TempDir(), "groups.json")), recorder)
	module.inventory = inventory
	module.now = func() time.Time { return now }
	resolver := testLANScopeModule()
	module.AttachLANScopes(resolver)
	contexts := map[string]context.Context{}
	for _, network := range []string{"lan", "office"} {
		scope, err := resolver.Resolve(context.Background(), network)
		if err != nil {
			t.Fatal(err)
		}
		contexts[network] = context.WithValue(withLANScope(context.Background(), scope), lanScopeResolverKey{}, resolver)
	}
	for _, network := range []string{"lan", "office"} {
		allow := network == "office"
		response, err := module.Mutate(contexts[network], &models.DeviceGroupMutationRequest{Action: "upsert_group", Group: &models.DeviceGroup{ID: "same", Name: "same", Members: []string{id}, Policy: &models.GroupPolicy{Access: &allow}}})
		if err != nil || response.Result.Error != nil {
			t.Fatalf("group %s: %+v %v", network, response, err)
		}
	}
	if len(recorder.networks) != 2 || recorder.networks[0] != "lan" || recorder.access[0] || recorder.networks[1] != "office" || !recorder.access[1] {
		t.Fatalf("cross-network execution: %+v", recorder)
	}
	rejected, err := module.Mutate(contexts["lan"], &models.DeviceGroupMutationRequest{Action: "upsert_group", Group: &models.DeviceGroup{ID: "foreign", Name: "foreign", Members: []string{"mac:02:00:00:00:00:66"}}})
	if err != nil || rejected.Result.Error == nil || rejected.Result.Error.Code != "device_network_mismatch" {
		t.Fatalf("foreign membership accepted: %+v %v", rejected, err)
	}
	for _, network := range []string{"lan", "office"} {
		response, err := module.List(contexts[network])
		if err != nil || len(response.Result.Groups) != 1 || response.Result.NetworkID != network || response.Result.LastBatch.NetworkID != network {
			t.Fatalf("runtime mixed: %+v %v", response, err)
		}
	}
	// Same desired policy must be re-evaluated after a binding change, without a
	// page visit and without replaying the unchanged LAN's batch.
	recorder.networks = nil
	recorder.access = nil
	lanBinding := *selectedLANScope(contexts["lan"])
	officeBinding := *selectedLANScope(contexts["office"])
	officeBinding.Device = "br-recreated"
	officeBinding.Version += "-recreated"
	resolver.reader = fakeLANScopeReader{scopes: []*models.LANScope{&lanBinding, &officeBinding}}
	module.runScheduledNetworks(now)
	if len(recorder.networks) != 1 || recorder.networks[0] != "office" {
		t.Fatalf("scheduler reused stale binding: %v", recorder.networks)
	}
}

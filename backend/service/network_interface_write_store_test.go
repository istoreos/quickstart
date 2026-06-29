package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var networkInterfaceWriteStoreTestMu sync.Mutex

func TestReadNetworkInterfaceConfigSnapshot(t *testing.T) {
	networkInterfaceWriteStoreTestMu.Lock()
	defer networkInterfaceWriteStoreTestMu.Unlock()

	oldLoad := networkInterfaceWriteLoadConfig
	oldGetSections := networkInterfaceWriteGetSections
	oldGetLast := networkInterfaceWriteGetLast
	oldGet := networkInterfaceWriteGet
	oldGetConfig := networkInterfaceWriteGetConfig
	defer func() {
		networkInterfaceWriteLoadConfig = oldLoad
		networkInterfaceWriteGetSections = oldGetSections
		networkInterfaceWriteGetLast = oldGetLast
		networkInterfaceWriteGet = oldGet
		networkInterfaceWriteGetConfig = oldGetConfig
	}()

	var loaded []string
	networkInterfaceWriteLoadConfig = func(name string, forceReload bool) error {
		loaded = append(loaded, name)
		return nil
	}
	networkInterfaceWriteGetSections = func(config, sectionType string) ([]string, bool) {
		switch {
		case config == "network" && sectionType == "device":
			return []string{"cfg01"}, true
		case config == "firewall" && sectionType == "zone":
			return []string{"cfg10"}, true
		default:
			return nil, false
		}
	}
	networkInterfaceWriteGetLast = func(config, section, option string) (string, bool) {
		switch {
		case config == "network" && section == "cfg01" && option == "name":
			return "br-lan", true
		case config == "firewall" && section == "cfg10" && option == "name":
			return "lan", true
		default:
			return "", false
		}
	}
	networkInterfaceWriteGet = func(config, section, option string) ([]string, bool) {
		if config == "firewall" && section == "cfg10" && option == "network" {
			return []string{"lan"}, true
		}
		return nil, false
	}
	networkInterfaceWriteGetConfig = func(ctx context.Context) (*models.NetworkInterfaceGetConfigResponse, error) {
		return &models.NetworkInterfaceGetConfigResponse{
			Result: &models.NetworkInterfaceGetConfigResponseResult{
				Interfaces: []*models.NetworkInterfaceInfo{{Name: "lan"}, {Name: "wan"}},
			},
		}, nil
	}

	snapshot, err := NewDefaultNetworkInterfaceConfigStore().ReadSnapshot(context.Background())
	if err != nil {
		t.Fatalf("unexpected snapshot error: %v", err)
	}
	if !reflect.DeepEqual(loaded, []string{"network", "firewall"}) {
		t.Fatalf("unexpected load order: %#v", loaded)
	}
	if len(snapshot.Interfaces) != 2 || snapshot.Interfaces[1].Name != "wan" {
		t.Fatalf("unexpected interfaces: %#v", snapshot.Interfaces)
	}
	if len(snapshot.DeviceSections) != 1 || snapshot.DeviceSections[0].Name != "br-lan" {
		t.Fatalf("unexpected device sections: %#v", snapshot.DeviceSections)
	}
	if len(snapshot.FirewallZones) != 1 || snapshot.FirewallZones[0].Name != "lan" || !reflect.DeepEqual(snapshot.FirewallZones[0].Networks, []string{"lan"}) {
		t.Fatalf("unexpected firewall zones: %#v", snapshot.FirewallZones)
	}
}

func TestDefaultNetworkInterfaceConfigStoreExecutesPlannedCommandBatches(t *testing.T) {
	networkInterfaceWriteStoreTestMu.Lock()
	defer networkInterfaceWriteStoreTestMu.Unlock()

	oldExec := networkInterfaceWriteBatchRun
	defer func() {
		networkInterfaceWriteBatchRun = oldExec
	}()

	var got [][]string
	networkInterfaceWriteBatchRun = func(ctx context.Context, commands []string, timeout int) error {
		_ = ctx
		_ = timeout
		got = append(got, append([]string(nil), commands...))
		return nil
	}

	store := NewDefaultNetworkInterfaceConfigStore()
	err := store.ApplyPlan(context.Background(), NetworkInterfaceCommandPlan{
		DeleteCommands:    []string{"uci del network.wan6"},
		BridgeCommands:    [][]string{{"uci add network device"}, {"uci add_list network.cfg01.ports=eth0"}},
		InterfaceCommands: [][]string{{"uci set network.lan=interface", "uci set network.lan.device=br-lan"}},
		FirewallCommands:  [][]string{{"uci add_list firewall.cfg10.network=guest"}},
	})
	if err != nil {
		t.Fatalf("ApplyPlan returned error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 command batches, got %#v", got)
	}
	if !reflect.DeepEqual(got[0], []string{"uci del network.wan6"}) {
		t.Fatalf("unexpected delete batch: %#v", got[0])
	}
	if !reflect.DeepEqual(got[4], []string{"uci add_list firewall.cfg10.network=guest"}) {
		t.Fatalf("unexpected firewall batch: %#v", got[4])
	}
}

func TestDefaultNetworkInterfaceConfigStorePropagatesBatchError(t *testing.T) {
	networkInterfaceWriteStoreTestMu.Lock()
	defer networkInterfaceWriteStoreTestMu.Unlock()

	oldExec := networkInterfaceWriteBatchRun
	defer func() {
		networkInterfaceWriteBatchRun = oldExec
	}()

	networkInterfaceWriteBatchRun = func(ctx context.Context, commands []string, timeout int) error {
		_ = ctx
		_ = commands
		_ = timeout
		return errors.New("batch failed")
	}

	err := NewDefaultNetworkInterfaceConfigStore().ApplyPlan(context.Background(), NetworkInterfaceCommandPlan{
		DeleteCommands: []string{"uci del network.wan6"},
	})
	if err == nil || err.Error() != "batch failed" {
		t.Fatalf("expected batch failure, got %v", err)
	}
}

func TestDefaultNetworkInterfaceConfigStoreUsesExpansionPortSeam(t *testing.T) {
	networkInterfaceWriteStoreTestMu.Lock()
	defer networkInterfaceWriteStoreTestMu.Unlock()

	oldHasPort := networkInterfaceWriteHasLink
	defer func() {
		networkInterfaceWriteHasLink = oldHasPort
	}()

	networkInterfaceWriteHasLink = func(name string) bool {
		return name == "dsm-ext" || name == "vm-ext"
	}

	got := NewDefaultNetworkInterfaceConfigStore().NormalizeDevices("lan", []string{"eth0"})
	want := []string{"eth0", "dsm-ext", "vm-ext"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized devices = %#v, want %#v", got, want)
	}
}

func TestDefaultNetworkInterfaceConfigApplyDelegatesExpectedConfigs(t *testing.T) {
	networkInterfaceWriteStoreTestMu.Lock()
	defer networkInterfaceWriteStoreTestMu.Unlock()

	oldApply := networkInterfaceWriteCommitAndApply
	defer func() {
		networkInterfaceWriteCommitAndApply = oldApply
	}()

	var got []string
	networkInterfaceWriteCommitAndApply = func(ctx context.Context, configs []string) error {
		_ = ctx
		got = append(got, configs...)
		return nil
	}

	err := NewDefaultNetworkInterfaceConfigApply().Apply(context.Background(), []string{"network", "firewall"})
	if err != nil {
		t.Fatalf("unexpected apply error: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"network", "firewall"}) {
		t.Fatalf("unexpected apply configs: %#v", got)
	}
}

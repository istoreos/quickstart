package service

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func inventoryLANContext(network string) context.Context {
	return withLANScope(context.Background(), &models.LANScope{NetworkID: network})
}

func TestMultiLANInventoryPreservesIDsAndScopesSameMAC(t *testing.T) {
	now := time.Now()
	const mac = "02:00:00:00:03:01"
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		ARP: []deviceInventoryObservation{
			{NetworkID: "lan", MAC: mac, IPv4: "192.0.2.10", Online: true},
			{NetworkID: "office", MAC: mac, IPv4: "198.51.100.10", Online: true},
		},
		DHCPv4:         []deviceInventoryObservation{{NetworkID: unassignedLAN, MAC: "02:00:00:00:03:02", IPv4: "203.0.113.10"}},
		NetworkDevices: map[string]string{"lan": "br-lan", "office": "br-office"},
	}}}
	inventory := newTestDeviceInventory(t, source, &now, 100)
	inventory.cacheTTL = time.Minute
	all, err := inventory.Snapshot(inventoryLANContext("all"))
	if err != nil || len(all.Result.Devices) != 3 {
		t.Fatalf("aggregate inventory: %#v, %v", all, err)
	}
	for _, network := range []string{"lan", "office"} {
		response, err := inventory.Snapshot(inventoryLANContext(network))
		if err != nil || len(response.Result.Devices) != 1 || response.Result.Devices[0].DeviceID != "mac:02:00:00:00:03:01" || response.Result.Devices[0].NetworkID != network {
			t.Fatalf("network %s: %#v, %v", network, response, err)
		}
	}
	unknown := 0
	for _, device := range all.Result.Devices {
		if device.Membership == "unconfirmed" {
			unknown++
		}
	}
	if unknown != 1 || source.index != 1 {
		t.Fatalf("unknown=%d source reads=%d", unknown, source.index)
	}
}

func TestMultiLANProfileCorruptionDegradesOnlyItsOwnNetwork(t *testing.T) {
	now := time.Now()
	snapshot := deviceInventorySourceSnapshot{NetworkReasons: map[string][]string{"lan": {}, "office": {}}, ARP: []deviceInventoryObservation{
		{NetworkID: "lan", MAC: "02:00:00:00:03:01", IPv4: "192.0.2.10"},
		{NetworkID: "office", MAC: "02:00:00:00:03:01", IPv4: "198.51.100.10"},
	}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	path := lanStatePath(inventory.classificationOverrides.path, "office")
	if err := persistClassificationOverrides(path, []byte("invalid json")); err != nil {
		t.Fatal(err)
	}
	lan, err := inventory.Snapshot(inventoryLANContext("lan"))
	if err != nil || lan.Result.Health.State != "ready" {
		t.Fatalf("healthy LAN degraded: %#v %v", lan, err)
	}
	office, err := inventory.Snapshot(inventoryLANContext("office"))
	if err != nil || office.Result.Health.State != "partial" || !containsString(office.Result.Health.Reasons, "device_profile_store_invalid") {
		t.Fatalf("corrupt profile health was hidden: %#v %v", office, err)
	}
}

func TestMultiLANProfileAndHistoryRemainIsolatedAfterRestart(t *testing.T) {
	now := time.Now()
	const id = "mac:02:00:00:00:03:01"
	snapshot := deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{
		{NetworkID: "lan", MAC: "02:00:00:00:03:01", IPv4: "192.0.2.10", Online: true},
		{NetworkID: "office", MAC: "02:00:00:00:03:01", IPv4: "198.51.100.10", Online: true},
	}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	profile := NewDeviceProfileModule(inventory)
	for network, alias := range map[string]string{"lan": "家庭终端", "office": "办公终端"} {
		response, err := profile.Apply(inventoryLANContext(network), &models.DeviceProfileApplyRequest{DeviceID: id, Action: "patch", Alias: &alias})
		if err != nil || response.Result.Error != nil {
			t.Fatalf("profile %s: %#v, %v", network, response, err)
		}
	}
	for _, network := range []string{"lan", "office"} {
		raw, err := os.ReadFile(lanStatePath(inventory.historyPath, network))
		if err != nil {
			t.Fatal(err)
		}
		var history deviceInventoryHistory
		if err := json.Unmarshal(raw, &history); err != nil || len(history.Devices) != 1 || history.Devices[id] == nil {
			t.Fatalf("history %s: %s %v", network, raw, err)
		}
	}
	restarted := newDeviceInventoryModuleForTest(&fakeDeviceInventorySource{}, inventory.historyPath, "boot-test", func() time.Time { return now }, 100)
	for network, alias := range map[string]string{"lan": "家庭终端", "office": "办公终端"} {
		response, err := NewDeviceProfileModule(restarted).Get(inventoryLANContext(network), id)
		if err != nil || response.Result.Error != nil || response.Result.Profile.Alias != alias {
			t.Fatalf("restarted profile %s: %#v, %v", network, response, err)
		}
		devices, err := restarted.Snapshot(inventoryLANContext(network))
		if err != nil || len(devices.Result.Devices) != 1 || devices.Result.Devices[0].Online {
			t.Fatalf("offline history %s: %#v, %v", network, devices, err)
		}
		for _, address := range devices.Result.Devices[0].Addresses.Historical {
			if network == "lan" && address.Address != "192.0.2.10" {
				t.Fatal("office address leaked into LAN history")
			}
			if network == "office" && address.Address != "198.51.100.10" {
				t.Fatal("LAN address leaked into office history")
			}
		}
	}
}

func TestMultiLANManualDevicesAndProfilesUseSeparateFiles(t *testing.T) {
	now := time.Now()
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{}, &now, 100)
	for network, alias := range map[string]string{"lan": "手动家庭设备", "office": "手动办公设备"} {
		response, err := inventory.AddManual(inventoryLANContext(network), &models.DeviceInventoryAddRequest{MAC: "02:00:00:00:03:01", Alias: alias})
		if err != nil || len(response.Result.Devices) != 1 || response.Result.Devices[0].DisplayName != alias {
			t.Fatalf("manual %s: %#v, %v", network, response, err)
		}
	}
	restarted := newDeviceInventoryModuleForTest(&fakeDeviceInventorySource{}, inventory.historyPath, "boot-test", func() time.Time { return now }, 100)
	for network, alias := range map[string]string{"lan": "手动家庭设备", "office": "手动办公设备"} {
		response, err := restarted.Snapshot(inventoryLANContext(network))
		if err != nil || len(response.Result.Devices) != 1 || response.Result.Devices[0].DisplayName != alias || response.Result.Devices[0].PresenceState != "never_seen" {
			t.Fatalf("manual restart %s: %#v, %v", network, response, err)
		}
	}
}

func TestMultiLANIPv6IdentityAndLinkLocalZone(t *testing.T) {
	now := time.Now()
	snapshot := deviceInventorySourceSnapshot{NetworkDevices: map[string]string{"lan": "br-lan", "office": "br-office"}}
	for _, network := range []string{"lan", "office"} {
		snapshot.NDP = append(snapshot.NDP, deviceInventoryObservation{NetworkID: network, MAC: "02:00:00:00:03:01", IPv6: "fe80::3", Online: true})
		snapshot.DHCPv6 = append(snapshot.DHCPv6, deviceInventoryObservation{NetworkID: network, DUID: "0001000300010203", IAID: "0001", IPv6: "2001:db8::3"})
	}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	for network, zone := range snapshot.NetworkDevices {
		response, err := inventory.Snapshot(inventoryLANContext(network))
		if err != nil || len(response.Result.Devices) != 2 {
			t.Fatalf("IPv6 %s: %#v, %v", network, response, err)
		}
		for _, device := range response.Result.Devices {
			if device.Mac != "" && device.Addresses.Current[0].Zone != zone {
				t.Fatalf("link-local zone: %#v", device.Addresses.Current)
			}
		}
	}
}

func TestMultiLANLeaseAndGlobalHintsRequireUnambiguousEvidence(t *testing.T) {
	scopes := []*models.LANScope{
		{NetworkID: "lan", Device: "br-lan", IPv4Prefixes: []string{"192.0.2.0/24"}},
		{NetworkID: "office", Device: "br-office", IPv4Prefixes: []string{"198.51.100.0/24"}},
	}
	if leaseLANOwner("192.0.2.10", "/tmp/dhcp.leases", scopes) != "lan" || leaseLANOwner("203.0.113.1", "/tmp/dhcp.leases", scopes) != unassignedLAN {
		t.Fatal("lease network was guessed")
	}
	scopes[1].IPv4Prefixes = []string{"192.0.2.0/24"}
	if leaseLANOwner("192.0.2.10", "/tmp/dhcp.leases", scopes) != unassignedLAN {
		t.Fatal("overlapping lease was assigned to a network")
	}
	items := parseLANInventoryARP(strings.NewReader("192.0.2.10 0x1 0x2 02:00:00:00:03:01 * br-office\n"), scopes[0])
	if len(items) != 0 {
		t.Fatal("ARP from another network leaked")
	}
	mac := "02:00:00:00:03:01"
	snapshot := deviceInventorySourceSnapshot{
		ARP: []deviceInventoryObservation{{NetworkID: "lan", MAC: mac, IPv4: "192.0.2.10"}, {NetworkID: "office", MAC: mac, IPv4: "192.0.2.11"}},
		NetworkHints: map[string]map[string]HostHintSnapshot{
			"lan":    {mac: {Hostname: "global-name", IPAddrs: []string{"192.0.2.10", "192.0.2.11"}}},
			"office": {mac: {Hostname: "global-name", IPAddrs: []string{"192.0.2.10", "192.0.2.11"}}},
		},
	}
	scopeInventoryHints(&snapshot, scopes)
	for _, network := range []string{"lan", "office"} {
		hint := snapshot.NetworkHints[network][mac]
		if hint.Hostname != "" || len(hint.IPAddrs) != 1 {
			t.Fatalf("ambiguous hints: %#v", hint)
		}
	}
}

func TestMultiLANOldManualRecordBecomesUnconfirmedWithoutLegacyNetwork(t *testing.T) {
	now := time.Now()
	snapshot := deviceInventorySourceSnapshot{NetworkDevices: map[string]string{"office": "br-office"}}
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{snapshot}}, &now, 100)
	_, err := inventory.AddManual(context.Background(), &models.DeviceInventoryAddRequest{MAC: "02:00:00:00:03:01", Alias: "旧记录"})
	if err != nil {
		t.Fatal(err)
	}
	response, err := inventory.Snapshot(inventoryLANContext("all"))
	if err != nil || len(response.Result.Devices) != 1 || response.Result.Devices[0].Membership != "unconfirmed" {
		t.Fatalf("orphaned manual device: %#v, %v", response, err)
	}
	office, _ := inventory.Snapshot(inventoryLANContext("office"))
	if len(office.Result.Devices) != 0 {
		t.Fatal("legacy manual data silently migrated to office")
	}
}

type invalidatedLANInventorySource struct {
	started chan struct{}
	release chan struct{}
	reads   atomic.Int32
}

func (source *invalidatedLANInventorySource) Read(ctx context.Context) deviceInventorySourceSnapshot {
	mac := "02:00:00:00:03:02"
	if source.reads.Add(1) == 1 {
		close(source.started)
		select {
		case <-ctx.Done():
			return deviceInventorySourceSnapshot{}
		case <-source.release:
		}
		mac = "02:00:00:00:03:01"
	}
	return deviceInventorySourceSnapshot{ARP: []deviceInventoryObservation{{NetworkID: "office", MAC: mac, IPv4: "198.51.100.10", Online: true}}}
}

func TestMultiLANInvalidationCannotPublishAnOlderRefresh(t *testing.T) {
	now := time.Now()
	source := &invalidatedLANInventorySource{started: make(chan struct{}), release: make(chan struct{})}
	inventory := newTestDeviceInventory(t, source, &now, 100)
	inventory.cacheTTL = time.Minute
	ctx, cancel := context.WithTimeout(inventoryLANContext("office"), 3*time.Second)
	defer cancel()
	firstDone := make(chan struct{})
	go func() { defer close(firstDone); _, _ = inventory.Snapshot(ctx) }()
	<-source.started
	inventory.Invalidate()
	close(source.release)
	<-firstDone
	result, err := inventory.Snapshot(ctx)
	if err != nil || len(result.Result.Devices) != 2 || result.Result.Devices[0].Mac != "02:00:00:00:03:02" || source.reads.Load() != 2 {
		t.Fatalf("invalidated result was cached: result=%#v err=%v reads=%d", result, err, source.reads.Load())
	}
}

func TestMultiLANSystemSourceReadOnly(t *testing.T) {
	if os.Getenv("QUICKSTART_MULTI_LAN_TEST_DEVICE") != "1" {
		t.Skip("requires an authorized OpenWrt test device")
	}
	now := time.Now()
	reader := systemDeviceInventorySource{scopes: NewLANScopeModule()}
	inventory := newTestDeviceInventory(t, reader, &now, 2048)
	response, err := inventory.Snapshot(inventoryLANContext("all"))
	if err != nil || response.Result == nil {
		t.Fatalf("system inventory: %v", err)
	}
	scopes, err := reader.scopes.List(context.Background())
	if err != nil || len(scopes) == 0 {
		t.Fatalf("system scopes: %v", err)
	}
	seen := map[string]bool{}
	for _, device := range response.Result.Devices {
		key := inventoryLANKey(device.NetworkID, device.DeviceID)
		if seen[key] {
			t.Fatalf("duplicate network/device identity")
		}
		seen[key] = true
		if device.Membership == "observed" && device.NetworkID == "" {
			t.Fatal("online observation lacks its network")
		}
	}
	t.Logf("system scopes=%d inventory_records=%d health=%s state_root=task_owned_temp", len(scopes), len(response.Result.Devices), response.Result.Health.State)
}

func TestMultiLANIsolatedKernelNeighbors(t *testing.T) {
	if os.Getenv("QUICKSTART_MULTI_LAN_TEST_DEVICE") != "1" {
		t.Skip("requires an authorized OpenWrt test device")
	}
	var wan ubusNetworkInterface
	if err := UbusCallWithObject(context.Background(), "network.interface.wan status", &wan); err != nil {
		t.Fatal(err)
	}
	authorized := false
	for _, address := range wan.Ipv4 {
		authorized = authorized || address.Address == "192.168.30.93" || address.Address == "192.168.30.7"
	}
	if !authorized {
		t.Fatal("isolated kernel test refused outside the authorized test hosts")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	created := []string{}
	t.Cleanup(func() {
		for _, device := range created {
			command := exec.Command("ip", "link", "delete", device)
			if output, err := command.CombinedOutput(); err != nil {
				t.Errorf("cleanup %s: %s %v", device, output, err)
			}
		}
	})
	run := func(args ...string) {
		t.Helper()
		if output, err := exec.CommandContext(ctx, "ip", args...).CombinedOutput(); err != nil {
			t.Fatalf("ip %v: %s %v", args, output, err)
		}
	}
	scopes := []*models.LANScope{}
	for index, suffix := range []string{"a", "b"} {
		device := "qsm" + strconv.Itoa(os.Getpid()) + suffix
		if exec.CommandContext(ctx, "ip", "link", "show", device).Run() == nil {
			t.Fatal("task-owned link name already exists")
		}
		run("link", "add", device, "type", "bridge")
		created = append(created, device)
		v4, v6 := "192.0.2.", "fd00:31::"
		if index == 1 {
			v4, v6 = "198.51.100.", "fd00:32::"
		}
		run("addr", "add", v4+"1/24", "dev", device)
		run("-6", "addr", "add", v6+"1/64", "dev", device)
		run("link", "set", device, "up")
		run("neigh", "replace", v4+"10", "lladdr", "02:00:00:00:03:01", "nud", "permanent", "dev", device)
		run("-6", "neigh", "replace", v6+"10", "lladdr", "02:00:00:00:03:01", "nud", "permanent", "dev", device)
		scopes = append(scopes, &models.LANScope{NetworkID: "lab_" + suffix, Device: device, State: "up", IPv4Prefixes: []string{v4 + "0/24"}, IPv6Prefixes: []string{v6 + "/64"}})
	}
	now := time.Now()
	reader := systemDeviceInventorySource{scopes: &LANScopeModule{reader: fakeLANScopeReader{scopes: scopes}, now: time.Now}}
	inventory := newTestDeviceInventory(t, reader, &now, 2048)
	for _, scope := range scopes {
		response, err := inventory.Snapshot(inventoryLANContext(scope.NetworkID))
		if err != nil || len(response.Result.Devices) == 0 {
			t.Fatalf("kernel scope %s: %#v %v", scope.NetworkID, response, err)
		}
		var device *models.DeviceInventoryItem
		for _, item := range response.Result.Devices {
			if item.DeviceID == "mac:02:00:00:00:03:01" {
				device = item
			}
			for _, address := range item.Addresses.Current {
				parsed, parseErr := netip.ParseAddr(address.Address)
				if parseErr != nil || (!addressInPrefixes(parsed, scopeLANPrefixes(scope)) && !(parsed.IsLinkLocalUnicast() && address.Zone == scope.Device)) {
					t.Fatalf("kernel address crossed LAN: network=%s address=%s zone=%s", scope.NetworkID, address.Address, address.Zone)
				}
			}
		}
		if device == nil {
			t.Fatal("controlled neighbor was not discovered")
		}
		if !device.Online || device.NetworkID != scope.NetworkID || len(device.Addresses.Current) != 2 {
			t.Fatalf("ARP/NDP network ownership: %#v", device)
		}
		for _, address := range device.Addresses.Current {
			if strings.HasPrefix(address.Address, "198.51.100.") && scope.NetworkID == "lab_a" {
				t.Fatal("kernel address crossed LAN")
			}
		}
	}
	t.Log("two isolated kernel bridges: same MAC; IPv4 ARP and IPv6 NDP remain network-scoped")
}

func TestInactiveLANRetainsEvidenceWithoutReportingOnline(t *testing.T) {
	source := deviceInventorySourceSnapshot{NetworkReasons: map[string][]string{}, ARP: []deviceInventoryObservation{{NetworkID: "office", MAC: "02:00:00:00:00:55", IPv4: "192.0.2.55", Online: true}, {NetworkID: "lan", MAC: "02:00:00:00:00:55", IPv4: "198.51.100.55", Online: true}}}
	demoteInactiveLANObservations(&source, []*models.LANScope{{NetworkID: "office", State: "down"}, {NetworkID: "lan", State: "up"}})
	if source.ARP[0].Online || !source.ARP[1].Online || source.ARP[0].IPv4 != "192.0.2.55" || len(source.NetworkReasons["office"]) != 1 {
		t.Fatal("link state leaked or erased history", source)
	}
}

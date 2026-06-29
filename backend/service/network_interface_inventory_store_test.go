package service

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"testing"

	simplejson "github.com/bitly/go-simplejson"
	"github.com/linkease/quick-start/istore-backend/models"
)

func TestResolveNetworkInterfaceAttachmentsPrefersSlavePortsThenFallsBackToDirectPort(t *testing.T) {
	t.Parallel()

	allPorts := map[string]*models.NetworkPortInfo{
		"eth0": {Name: "eth0"},
		"lan1": {Name: "lan1", Master: "br-lan"},
		"lan2": {Name: "lan2", Master: "br-lan"},
	}

	bridgePorts, bridgeDeviceNames := resolveNetworkInterfaceAttachments(allPorts, "br-lan")
	if len(bridgePorts) != 2 {
		t.Fatalf("expected slave ports for br-lan, got %#v", bridgePorts)
	}
	bridgePortNames := []string{bridgePorts[0].Name, bridgePorts[1].Name}
	sort.Strings(bridgePortNames)
	if bridgePortNames[0] != "lan1" || bridgePortNames[1] != "lan2" {
		t.Fatalf("expected slave ports for br-lan, got %#v", bridgePorts)
	}
	if len(bridgeDeviceNames) != 2 {
		t.Fatalf("expected device names derived from slave ports, got %#v", bridgeDeviceNames)
	}
	sort.Strings(bridgeDeviceNames)
	if bridgeDeviceNames[0] != "lan1" || bridgeDeviceNames[1] != "lan2" {
		t.Fatalf("expected device names derived from slave ports, got %#v", bridgeDeviceNames)
	}

	directPorts, directDeviceNames := resolveNetworkInterfaceAttachments(allPorts, "eth0")
	if len(directPorts) != 1 || directPorts[0].Name != "eth0" {
		t.Fatalf("expected direct port fallback for eth0, got %#v", directPorts)
	}
	if len(directDeviceNames) != 1 || directDeviceNames[0] != "eth0" {
		t.Fatalf("expected direct device name fallback, got %#v", directDeviceNames)
	}
}

func TestFilterNetworkInterfaceGetConfigDropsDhcpv6Only(t *testing.T) {
	t.Parallel()

	interfaces := []*models.NetworkInterfaceInfo{
		{Name: "wan", Proto: "dhcp"},
		{Name: "wan6", Proto: "dhcpv6"},
		{Name: "lan", Proto: "static"},
	}

	filtered := filterNetworkInterfaceGetConfig(interfaces)
	if len(filtered) != 2 {
		t.Fatalf("expected dhcpv6-only filter, got %#v", filtered)
	}
	if filtered[0].Name != "wan" || filtered[1].Name != "lan" {
		t.Fatalf("unexpected filtered interfaces: %#v", filtered)
	}
}

func TestBuildNetworkInterfaceStatusResult(t *testing.T) {
	t.Parallel()

	interfaces := []*models.NetworkInterfaceInfo{{Name: "lan"}}

	resp := buildNetworkInterfaceStatusResult(interfaces)
	if resp == nil || resp.Result == nil || len(resp.Result.Interfaces) != 1 || resp.Result.Interfaces[0].Name != "lan" {
		t.Fatalf("unexpected status response: %#v", resp)
	}
}

func TestBuildNetworkInterfaceGetConfigResult(t *testing.T) {
	t.Parallel()

	devices := []*models.NetworkPortInfo{{Name: "eth0"}}
	interfaces := []*models.NetworkInterfaceInfo{{Name: "wan"}}

	resp := buildNetworkInterfaceGetConfigResult(devices, interfaces)
	if resp == nil || resp.Result == nil {
		t.Fatalf("expected non-nil get-config response, got %#v", resp)
	}
	if len(resp.Result.Devices) != 1 || resp.Result.Devices[0].Name != "eth0" {
		t.Fatalf("unexpected devices payload: %#v", resp.Result.Devices)
	}
	if len(resp.Result.Interfaces) != 1 || resp.Result.Interfaces[0].Name != "wan" {
		t.Fatalf("unexpected interfaces payload: %#v", resp.Result.Interfaces)
	}
}

func TestDefaultNetworkInterfaceInventoryReaderFiltersDockerAndLoopback(t *testing.T) {
	t.Parallel()

	original := readNetworkInterfaceInventorySnapshots
	readNetworkInterfaceInventorySnapshots = func(ctx context.Context) ([]NetworkInterfaceInventorySnapshot, error) {
		return []NetworkInterfaceInventorySnapshot{
			{Name: "wan", Proto: "dhcp", PortName: "eth0"},
			{Name: "docker", Proto: "static", PortName: "docker0"},
			{Name: "loopback", Proto: "static", PortName: "lo"},
			{Name: "lan", Proto: "static", PortName: "br-lan"},
		}, nil
	}
	defer func() { readNetworkInterfaceInventorySnapshots = original }()

	reader := newDefaultNetworkInterfaceInventoryReader()
	snapshots, err := reader.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected inventory reader error: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("expected docker/loopback to be filtered, got %#v", snapshots)
	}
	if snapshots[0].Name != "wan" || snapshots[1].Name != "lan" {
		t.Fatalf("unexpected inventory snapshots: %#v", snapshots)
	}
}

func TestDefaultNetworkInterfaceInventoryReaderMapsLegacyError(t *testing.T) {
	t.Parallel()

	original := readNetworkInterfaceInventorySnapshots
	readNetworkInterfaceInventorySnapshots = func(ctx context.Context) ([]NetworkInterfaceInventorySnapshot, error) {
		return nil, errors.New("ubus failed")
	}
	defer func() { readNetworkInterfaceInventorySnapshots = original }()

	reader := newDefaultNetworkInterfaceInventoryReader()
	if _, err := reader.Read(context.Background()); err == nil || err.Error() != "获取网络接口失败" {
		t.Fatalf("expected legacy inventory error, got %v", err)
	}
}

func TestDefaultNetworkInterfaceFirewallBindingReaderBuildsMapping(t *testing.T) {
	t.Parallel()

	original := readNetworkInterfaceFirewallBindings
	readNetworkInterfaceFirewallBindings = func(ctx context.Context) (map[string]string, error) {
		return map[string]string{"lan": "lan", "wan": "wan"}, nil
	}
	defer func() { readNetworkInterfaceFirewallBindings = original }()

	reader := newDefaultNetworkInterfaceFirewallBindingReader()
	bindings, err := reader.Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected firewall binding reader error: %v", err)
	}
	if bindings["lan"] != "lan" || bindings["wan"] != "wan" {
		t.Fatalf("unexpected firewall bindings: %#v", bindings)
	}
}

func TestDefaultNetworkInterfacePortAttachmentResolverUsesCurrentFallbackSemantics(t *testing.T) {
	t.Parallel()

	allPorts := map[string]*models.NetworkPortInfo{
		"eth0": {Name: "eth0"},
		"lan1": {Name: "lan1", Master: "br-lan"},
	}

	resolver := newDefaultNetworkInterfacePortAttachmentResolver()

	bridgePorts, bridgeDeviceNames := resolver.Resolve(allPorts, "br-lan")
	if len(bridgePorts) != 1 || bridgePorts[0].Name != "lan1" || len(bridgeDeviceNames) != 1 || bridgeDeviceNames[0] != "lan1" {
		t.Fatalf("expected slave fallback for bridge device, got ports=%#v devices=%#v", bridgePorts, bridgeDeviceNames)
	}

	directPorts, directDeviceNames := resolver.Resolve(allPorts, "eth0")
	if len(directPorts) != 1 || directPorts[0].Name != "eth0" || len(directDeviceNames) != 1 || directDeviceNames[0] != "eth0" {
		t.Fatalf("expected direct fallback semantics, got ports=%#v devices=%#v", directPorts, directDeviceNames)
	}
}

func TestReadNetworkInterfaceInventorySnapshotsReadsUbusInterfaces(t *testing.T) {
	original := readNetworkInterfaceInventoryUbusCall
	readNetworkInterfaceInventoryUbusCall = func(ctx context.Context, arg string) (*simplejson.Json, error) {
		if arg != "network.interface dump" {
			t.Fatalf("unexpected ubus arg: %s", arg)
		}
		return simplejson.NewJson([]byte(`{
			"interface": [
				{
					"interface": "lan",
					"device": "br-lan",
					"proto": "static",
					"ipv4-address": [{"address": "192.168.1.1"}],
					"ipv6-address": [{"address": "fd00::1"}]
				},
				{
					"interface": "wan",
					"device": "eth0",
					"proto": "dhcp",
					"ipv4-address": [],
					"ipv6-address": []
				}
			]
		}`))
	}
	defer func() { readNetworkInterfaceInventoryUbusCall = original }()

	snapshots, err := readNetworkInterfaceInventorySnapshots(context.Background())
	if err != nil {
		t.Fatalf("unexpected inventory snapshot error: %v", err)
	}
	want := []NetworkInterfaceInventorySnapshot{
		{Name: "lan", Proto: "static", PortName: "br-lan", IPV4Addr: "192.168.1.1", IPV6Addr: "fd00::1"},
		{Name: "wan", Proto: "dhcp", PortName: "eth0"},
	}
	if !reflect.DeepEqual(snapshots, want) {
		t.Fatalf("snapshots = %#v, want %#v", snapshots, want)
	}
}

func TestReadNetworkInterfaceFirewallBindingsReadsUciZones(t *testing.T) {
	oldLoadConfig := readNetworkInterfaceFirewallLoadConfig
	oldGetSections := readNetworkInterfaceFirewallGetSections
	oldGetLast := readNetworkInterfaceFirewallGetLast
	oldGet := readNetworkInterfaceFirewallGet
	defer func() {
		readNetworkInterfaceFirewallLoadConfig = oldLoadConfig
		readNetworkInterfaceFirewallGetSections = oldGetSections
		readNetworkInterfaceFirewallGetLast = oldGetLast
		readNetworkInterfaceFirewallGet = oldGet
	}()

	var loaded []string
	readNetworkInterfaceFirewallLoadConfig = func(name string, forceReload bool) error {
		if !forceReload {
			t.Fatal("expected forced firewall reload")
		}
		loaded = append(loaded, name)
		return nil
	}
	readNetworkInterfaceFirewallGetSections = func(config, sectionType string) ([]string, bool) {
		if config != "firewall" || sectionType != "zone" {
			t.Fatalf("unexpected section lookup: %s %s", config, sectionType)
		}
		return []string{"cfg_lan", "cfg_wan"}, true
	}
	readNetworkInterfaceFirewallGetLast = func(config, section, option string) (string, bool) {
		if config != "firewall" || option != "name" {
			t.Fatalf("unexpected get-last lookup: %s %s %s", config, section, option)
		}
		return map[string]string{"cfg_lan": "lan", "cfg_wan": "wan"}[section], true
	}
	readNetworkInterfaceFirewallGet = func(config, section, option string) ([]string, bool) {
		if config != "firewall" || option != "network" {
			t.Fatalf("unexpected get lookup: %s %s %s", config, section, option)
		}
		switch section {
		case "cfg_lan":
			return []string{"lan"}, true
		case "cfg_wan":
			return []string{"wan", "wan6"}, true
		default:
			return nil, false
		}
	}

	bindings, err := readNetworkInterfaceFirewallBindings(context.Background())
	if err != nil {
		t.Fatalf("unexpected firewall binding error: %v", err)
	}
	if !reflect.DeepEqual(loaded, []string{"firewall"}) {
		t.Fatalf("loaded configs = %#v, want firewall", loaded)
	}
	want := map[string]string{"lan": "lan", "wan": "wan", "wan6": "wan"}
	if !reflect.DeepEqual(bindings, want) {
		t.Fatalf("bindings = %#v, want %#v", bindings, want)
	}
}

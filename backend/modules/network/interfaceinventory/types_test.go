package interfaceinventory

import (
	"sort"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestResolveAttachmentsPrefersSlavePortsThenFallsBackToDirectPort(t *testing.T) {
	t.Parallel()

	allPorts := map[string]*models.NetworkPortInfo{
		"eth0": {Name: "eth0"},
		"lan1": {Name: "lan1", Master: "br-lan"},
		"lan2": {Name: "lan2", Master: "br-lan"},
	}

	bridgePorts, bridgeDeviceNames := ResolveAttachments(allPorts, "br-lan")
	if len(bridgePorts) != 2 {
		t.Fatalf("expected slave ports for br-lan, got %#v", bridgePorts)
	}
	bridgePortNames := []string{bridgePorts[0].Name, bridgePorts[1].Name}
	sort.Strings(bridgePortNames)
	if bridgePortNames[0] != "lan1" || bridgePortNames[1] != "lan2" {
		t.Fatalf("expected slave ports for br-lan, got %#v", bridgePorts)
	}
	sort.Strings(bridgeDeviceNames)
	if len(bridgeDeviceNames) != 2 || bridgeDeviceNames[0] != "lan1" || bridgeDeviceNames[1] != "lan2" {
		t.Fatalf("expected device names derived from slave ports, got %#v", bridgeDeviceNames)
	}

	directPorts, directDeviceNames := ResolveAttachments(allPorts, "eth0")
	if len(directPorts) != 1 || directPorts[0].Name != "eth0" {
		t.Fatalf("expected direct port fallback for eth0, got %#v", directPorts)
	}
	if len(directDeviceNames) != 1 || directDeviceNames[0] != "eth0" {
		t.Fatalf("expected direct device name fallback, got %#v", directDeviceNames)
	}
}

func TestFilterGetConfigDropsDhcpv6Only(t *testing.T) {
	t.Parallel()

	interfaces := []*models.NetworkInterfaceInfo{
		{Name: "wan", Proto: "dhcp"},
		{Name: "wan6", Proto: "dhcpv6"},
		{Name: "lan", Proto: "static"},
	}

	filtered := FilterGetConfig(interfaces)
	if len(filtered) != 2 {
		t.Fatalf("expected dhcpv6-only filter, got %#v", filtered)
	}
	if filtered[0].Name != "wan" || filtered[1].Name != "lan" {
		t.Fatalf("unexpected filtered interfaces: %#v", filtered)
	}
}

func TestBuildStatusResult(t *testing.T) {
	t.Parallel()

	interfaces := []*models.NetworkInterfaceInfo{{Name: "lan"}}

	resp := BuildStatusResult(interfaces)
	if resp == nil || resp.Result == nil || len(resp.Result.Interfaces) != 1 || resp.Result.Interfaces[0].Name != "lan" {
		t.Fatalf("unexpected status response: %#v", resp)
	}
}

func TestBuildGetConfigResult(t *testing.T) {
	t.Parallel()

	devices := []*models.NetworkPortInfo{{Name: "eth0"}}
	interfaces := []*models.NetworkInterfaceInfo{{Name: "wan"}}

	resp := BuildGetConfigResult(devices, interfaces)
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

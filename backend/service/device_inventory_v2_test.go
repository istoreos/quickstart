package service

import (
	"context"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeDeviceInventorySource struct {
	snapshots []deviceInventorySourceSnapshot
	index     int
}

func (source *fakeDeviceInventorySource) Read(ctx context.Context) deviceInventorySourceSnapshot {
	_ = ctx
	if len(source.snapshots) == 0 {
		return deviceInventorySourceSnapshot{}
	}
	index := source.index
	if index >= len(source.snapshots) {
		index = len(source.snapshots) - 1
	}
	source.index++
	return source.snapshots[index]
}

func newTestDeviceInventory(t *testing.T, source deviceInventorySource, now *time.Time, limit int) *DeviceInventoryModule {
	t.Helper()
	return newDeviceInventoryModuleForTest(source, filepath.Join(t.TempDir(), "history.json"), "boot-test", func() time.Time {
		return *now
	}, limit)
}

func TestDeviceInventorySnapshotMergesSourcesByMAC(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		ARP:       []deviceInventoryObservation{{MAC: "aa-bb-cc-dd-ee-ff", IPv4: "192.168.100.20", Online: true}},
		DHCPv4:    []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:FF", IPv4: "192.168.100.20", Hostname: "living-room"}},
		HostHints: map[string]HostHintSnapshot{"AA:BB:CC:DD:EE:FF": {Hostname: "hint-name"}},
		WifiMACs:  map[string]struct{}{"AA:BB:CC:DD:EE:FF": {}},
	}}}

	response, err := newTestDeviceInventory(t, source, &now, 100).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Devices) != 1 {
		t.Fatalf("devices = %#v, want one merged device", response.Result.Devices)
	}
	device := response.Result.Devices[0]
	if device.DeviceID != "mac:aa:bb:cc:dd:ee:ff" || !device.Online || device.Hostname != "living-room" {
		t.Fatalf("unexpected merged device: %#v", device)
	}
	if device.Connection.Kind != "wifi" || len(device.Addresses.Current) != 1 || device.Addresses.Current[0].Address != "192.168.100.20" {
		t.Fatalf("unexpected connection/addresses: %#v", device)
	}
	if response.Result.Health.State != "ready" {
		t.Fatalf("health = %#v", response.Result.Health)
	}
}

func TestDeviceInventoryKeepsBootHistoryAndStableIdentity(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{
		{ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.100.20", Online: true}}},
		{},
		{ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.100.21", Online: true}}},
	}}
	module := newTestDeviceInventory(t, source, &now, 100)

	first, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	second, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Result.Devices) != 1 || second.Result.Devices[0].Online {
		t.Fatalf("expected one offline historical device: %#v", second.Result.Devices)
	}
	if second.Result.Devices[0].DeviceID != first.Result.Devices[0].DeviceID {
		t.Fatal("device identity changed while offline")
	}

	now = now.Add(time.Minute)
	third, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	device := third.Result.Devices[0]
	if device.DeviceID != first.Result.Devices[0].DeviceID || !device.Online {
		t.Fatalf("expected stable online identity: %#v", device)
	}
	if len(device.Addresses.Historical) != 1 || device.Addresses.Historical[0].Address != "192.168.100.20" {
		t.Fatalf("historical addresses = %#v", device.Addresses.Historical)
	}
}

func TestDeviceInventoryDoesNotTransferIdentityWhenIPIsReused(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{
		{ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.100.20", Hostname: "old", Online: true}}},
		{ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:02", IPv4: "192.168.100.20", Hostname: "new", Online: true}}},
	}}
	module := newTestDeviceInventory(t, source, &now, 100)
	if _, err := module.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	response, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Devices) != 2 {
		t.Fatalf("devices = %#v, want old and new identities", response.Result.Devices)
	}
	var oldDevice, newDeviceFound bool
	for _, device := range response.Result.Devices {
		switch device.DeviceID {
		case "mac:aa:bb:cc:dd:ee:01":
			oldDevice = !device.Online && device.Hostname == "old"
		case "mac:aa:bb:cc:dd:ee:02":
			newDeviceFound = device.Online && device.Hostname == "new"
		}
	}
	if !oldDevice || !newDeviceFound {
		t.Fatalf("IP reuse transferred identity: %#v", response.Result.Devices)
	}
}

func TestDeviceInventoryReportsPartialSourcesAndCapacity(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		ARP: []deviceInventoryObservation{
			{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.100.20", Online: true},
			{MAC: "AA:BB:CC:DD:EE:02", IPv4: "192.168.100.21", Online: true},
		},
		Reasons: []string{"wifi_assoc_unavailable"},
	}}}
	response, err := newTestDeviceInventory(t, source, &now, 1).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Devices) != 1 || response.Result.Health.State != "partial" {
		t.Fatalf("unexpected response: %#v", response.Result)
	}
	want := map[string]bool{"wifi_assoc_unavailable": true, "inventory_capacity_reached": true}
	for _, reason := range response.Result.Health.Reasons {
		delete(want, reason)
	}
	if len(want) != 0 {
		t.Fatalf("missing health reasons: %#v", want)
	}
}

func TestReadDeviceInventoryDHCPv4FiltersExpiredAndMalformedLeases(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "dhcp.leases")
	content := "200 AA:BB:CC:DD:EE:01 192.168.100.20 television client\n" +
		"50 AA:BB:CC:DD:EE:02 192.168.100.21 expired client\n" +
		"200 invalid 192.168.100.22 bad client\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	items, err := readDeviceInventoryDHCPv4(path, time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Hostname != "television" || items[0].IPv4 != "192.168.100.20" {
		t.Fatalf("unexpected leases: %#v", items)
	}
}

func TestBuildLANIPv4HostHintsRejectsNonLANAddresses(t *testing.T) {
	t.Parallel()
	prefix := netip.MustParsePrefix("192.168.100.0/24")
	hints := buildLANIPv4HostHints(ubusHostHintMap{
		"AA:BB:CC:DD:EE:01": {Name: "lan-device.lan", IPAddrs: []string{"192.168.100.20"}},
		"AA:BB:CC:DD:EE:02": {Name: "management.lan", IPAddrs: []string{"192.168.9.215"}},
	}, []netip.Prefix{prefix})
	if len(hints) != 1 || hints["AA:BB:CC:DD:EE:01"].Hostname != "lan-device" {
		t.Fatalf("unexpected LAN hints: %#v", hints)
	}
}

func TestDeviceInventoryAggregatesIPv4AndIPv6ByMAC(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.100.20", Online: true}},
		NDP: []deviceInventoryObservation{
			{MAC: "AA:BB:CC:DD:EE:01", IPv6: "fd5f:e357:7969::20", Online: true},
			{MAC: "AA:BB:CC:DD:EE:01", IPv6: "fe80::20", Online: true},
		},
	}}}
	response, err := newTestDeviceInventory(t, source, &now, 100).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Devices) != 1 {
		t.Fatalf("devices = %#v", response.Result.Devices)
	}
	device := response.Result.Devices[0]
	if len(device.Addresses.Current) != 3 || device.Addresses.Current[0].Address != "192.168.100.20" || !device.Addresses.Current[0].Primary {
		t.Fatalf("addresses = %#v", device.Addresses.Current)
	}
	if device.Identity.Kind != "mac" || device.Identity.Scope != "persistent" {
		t.Fatalf("identity = %#v", device.Identity)
	}
}

func TestDeviceInventoryIncludesIPv6OnlyNDPDevice(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		NDP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:06", IPv6: "fe80::6", Online: true}},
	}}}
	response, err := newTestDeviceInventory(t, source, &now, 100).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	device := response.Result.Devices[0]
	if !device.Online || device.Addresses.Current[0].Family != 6 || device.Addresses.Current[0].Address != "fe80::6" {
		t.Fatalf("IPv6-only device = %#v", device)
	}
}

func TestDeviceInventoryUsesBootScopedDUIDIdentityAndDeduplicates(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		DHCPv6: []deviceInventoryObservation{
			{DUID: "00020000ab11abcd", IAID: "1234abcd", IPv6: "fd5f:e357:7969::10", Hostname: "v6-client"},
			{DUID: "00020000AB11ABCD", IAID: "1234ABCD", IPv6: "fd5f:e357:7969::11", Hostname: "v6-client"},
		},
	}}}
	response, err := newTestDeviceInventory(t, source, &now, 100).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Devices) != 1 {
		t.Fatalf("devices = %#v", response.Result.Devices)
	}
	device := response.Result.Devices[0]
	if device.DeviceID != "duid:00020000ab11abcd:1234abcd" || device.Identity.Scope != "boot" || len(device.Addresses.Current) != 2 {
		t.Fatalf("DUID device = %#v", device)
	}
}

func TestDeviceInventoryLinksDHCPv6LeaseToNDPAddressOwner(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		NDP:    []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:06", IPv6: "fd5f:e357:7969::6", Online: true}},
		DHCPv6: []deviceInventoryObservation{{DUID: "00020000ab11abcd", IAID: "1234abcd", IPv6: "fd5f:e357:7969::6", Hostname: "v6-client"}},
	}}}
	response, err := newTestDeviceInventory(t, source, &now, 100).Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Devices) != 1 || response.Result.Devices[0].DeviceID != "mac:aa:bb:cc:dd:ee:06" || response.Result.Devices[0].Hostname != "v6-client" {
		t.Fatalf("linked lease = %#v", response.Result.Devices)
	}
}

func TestDeviceInventoryDoesNotTransferIPv6Identity(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{
		{NDP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv6: "fd5f:e357:7969::20", Online: true}}},
		{NDP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:02", IPv6: "fd5f:e357:7969::20", Online: true}}},
	}}
	module := newTestDeviceInventory(t, source, &now, 100)
	if _, err := module.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	response, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Devices) != 2 {
		t.Fatalf("devices = %#v", response.Result.Devices)
	}
	for _, device := range response.Result.Devices {
		if device.DeviceID == "mac:aa:bb:cc:dd:ee:01" && device.Online {
			t.Fatalf("old IPv6 owner remained online: %#v", device)
		}
		if device.DeviceID == "mac:aa:bb:cc:dd:ee:02" && (!device.Online || len(device.Addresses.Current) != 1) {
			t.Fatalf("new IPv6 owner is incorrect: %#v", device)
		}
	}
}

func TestParseDeviceInventoryNDPRejectsFailedEntries(t *testing.T) {
	t.Parallel()
	input := "fe80::1 lladdr aa:bb:cc:dd:ee:01 STALE\n" +
		"fe80::2 FAILED\n" +
		"fd5f:e357:7969::3 lladdr aa:bb:cc:dd:ee:03 REACHABLE\n"
	items, err := parseDeviceInventoryNDP(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].IPv6 != "fe80::1" || items[1].MAC != "AA:BB:CC:DD:EE:03" {
		t.Fatalf("NDP items = %#v", items)
	}
}

func TestParseDeviceInventoryDHCPv6UsesDUIDAndIAID(t *testing.T) {
	t.Parallel()
	input := "fd5f:e357:7969::128 client-one\n" +
		"# br-lan 00020000ab11b03a1ee6f241575b 66d17f74 client-one 200 128 128 fd5f:e357:7969::\n" +
		"fd5f:e357:7969::129 expired\n" +
		"# br-lan 00020000ab11b03a1ee6f241575c 66d17f75 expired 50 129 128 fd5f:e357:7969::\n"
	items, err := parseDeviceInventoryDHCPv6(strings.NewReader(input), time.Unix(100, 0), "br-lan")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].DUID != "00020000ab11b03a1ee6f241575b" || items[0].IAID != "66d17f74" {
		t.Fatalf("DHCPv6 items = %#v", items)
	}
}

func TestBuildLANHostHintsRequiresNDPProofForLinkLocalIPv6(t *testing.T) {
	t.Parallel()
	prefixes := []netip.Prefix{netip.MustParsePrefix("fd5f:e357:7969::/60"), netip.MustParsePrefix("fe80::/64")}
	hints := buildLANHostHints(ubusHostHintMap{
		"AA:BB:CC:DD:EE:01": {IPv6Addrs: []string{"fe80::1", "fd5f:e357:7969::1"}},
		"AA:BB:CC:DD:EE:02": {IPv6Addrs: []string{"fe80::2"}},
	}, prefixes, map[string]struct{}{"AA:BB:CC:DD:EE:01": {}})
	if len(hints) != 1 || len(hints["AA:BB:CC:DD:EE:01"].IPv6Addrs) != 2 {
		t.Fatalf("IPv6 hints = %#v", hints)
	}
}

package guestnetwork

import (
	"errors"
	"reflect"
	"testing"
)

func TestNetworkProbeAndPlanCommands(t *testing.T) {
	t.Parallel()

	if got := NetworkProbeCommands(); !reflect.DeepEqual(got, []string{"uci -q get network.guest_dev.name"}) {
		t.Fatalf("NetworkProbeCommands = %#v", got)
	}

	if got := PlanNetworkCommands([]byte("br-guest\n"), nil); len(got) != 0 {
		t.Fatalf("PlanNetworkCommands existing = %#v, want empty", got)
	}

	got := PlanNetworkCommands([]byte("missing"), nil)
	assertCommands(t, got, []string{
		`uci -q delete network.guest_dev`,
		`uci set network.guest_dev="device"`,
		`uci set network.guest_dev.type="bridge"`,
		`uci set network.guest_dev.name="br-guest"`,
		`uci -q delete network.guest`,
		`uci set network.guest="interface"`,
		`uci set network.guest.proto="static"`,
		`uci set network.guest.device="br-guest"`,
		`uci set network.guest.ipaddr="192.168.102.1/24"`,
		`uci commit network`,
		`service network restart`,
	})

	got = PlanNetworkCommands(nil, errors.New("not found"))
	if len(got) == 0 {
		t.Fatal("expected network commands when probe fails")
	}
}

func TestDHCPProbeAndPlanCommands(t *testing.T) {
	t.Parallel()

	if got := DHCPProbeCommands(); !reflect.DeepEqual(got, []string{"uci -q dhcp.guest"}) {
		t.Fatalf("DHCPProbeCommands = %#v", got)
	}

	if got := PlanDHCPCommands([]byte("dhcp\n"), nil); len(got) != 0 {
		t.Fatalf("PlanDHCPCommands existing = %#v, want empty", got)
	}

	assertCommands(t, PlanDHCPCommands([]byte("missing"), nil), []string{
		`uci -q delete dhcp.guest`,
		`uci set dhcp.guest="dhcp"`,
		`uci set dhcp.guest.interface="guest"`,
		`uci set dhcp.guest.start="100"`,
		`uci set dhcp.guest.limit="150"`,
		`uci set dhcp.guest.leasetime="1h"`,
		`uci commit dhcp`,
		`service dnsmasq restart`,
	})

	if got := PlanDHCPCommands(nil, errors.New("not found")); len(got) == 0 {
		t.Fatal("expected DHCP commands when probe fails")
	}
}

func TestFirewallProbeAndPlanCommands(t *testing.T) {
	t.Parallel()

	if got := FirewallProbeCommands(); !reflect.DeepEqual(got, []string{"uci -q get firewall.guest"}) {
		t.Fatalf("FirewallProbeCommands = %#v", got)
	}

	if got := PlanFirewallCommands([]byte("zone\n"), nil); len(got) != 0 {
		t.Fatalf("PlanFirewallCommands existing = %#v, want empty", got)
	}

	assertCommands(t, PlanFirewallCommands([]byte("missing"), nil), []string{
		`uci -q delete firewall.guest`,
		`uci set firewall.guest="zone"`,
		`uci set firewall.guest.name="guest"`,
		`uci set firewall.guest.network="guest"`,
		`uci set firewall.guest.input="REJECT"`,
		`uci set firewall.guest.output="ACCEPT"`,
		`uci set firewall.guest.forward="REJECT"`,
		`uci -q delete firewall.guest_wan`,
		`uci set firewall.guest_wan="forwarding"`,
		`uci set firewall.guest_wan.src="guest"`,
		`uci set firewall.guest_wan.dest="wan"`,
		`uci -q delete firewall.guest_dns`,
		`uci set firewall.guest_dns="rule"`,
		`uci set firewall.guest_dns.name="Allow-DNS-Guest"`,
		`uci set firewall.guest_dns.src="guest"`,
		`uci set firewall.guest_dns.dest_port="53"`,
		`uci set firewall.guest_dns.proto="tcp udp"`,
		`uci set firewall.guest_dns.target="ACCEPT"`,
		`uci -q delete firewall.guest_dhcp`,
		`uci set firewall.guest_dhcp="rule"`,
		`uci set firewall.guest_dhcp.name="Allow-DHCP-Guest"`,
		`uci set firewall.guest_dhcp.src="guest"`,
		`uci set firewall.guest_dhcp.dest_port="67"`,
		`uci set firewall.guest_dhcp.proto="udp"`,
		`uci set firewall.guest_dhcp.family="ipv4"`,
		`uci set firewall.guest_dhcp.target="ACCEPT"`,
		`uci commit firewall`,
		`service firewall restart`,
	})

	if got := PlanFirewallCommands(nil, errors.New("not found")); len(got) == 0 {
		t.Fatal("expected firewall commands when probe fails")
	}
}

func TestSuccessMarker(t *testing.T) {
	t.Parallel()

	if SuccessMarkerCommand() != "echo guest-network-ok" {
		t.Fatalf("SuccessMarkerCommand = %q", SuccessMarkerCommand())
	}
	if !HasSuccessMarker([]byte("guest-network-ok\n")) {
		t.Fatal("expected marker to be detected")
	}
	if HasSuccessMarker([]byte("unexpected")) {
		t.Fatal("did not expect marker to be detected")
	}
}

func assertCommands(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("commands = %#v, want %#v", got, want)
	}
}

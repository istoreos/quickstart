package guestnetwork

import "strings"

func NetworkProbeCommands() []string {
	return []string{"uci -q get network.guest_dev.name"}
}

func PlanNetworkCommands(output []byte, err error) []string {
	if err == nil && strings.HasPrefix(string(output), "br-guest") {
		return nil
	}
	return []string{
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
	}
}

func DHCPProbeCommands() []string {
	return []string{"uci -q dhcp.guest"}
}

func PlanDHCPCommands(output []byte, err error) []string {
	if err == nil && strings.HasPrefix(string(output), "dhcp") {
		return nil
	}
	return []string{
		`uci -q delete dhcp.guest`,
		`uci set dhcp.guest="dhcp"`,
		`uci set dhcp.guest.interface="guest"`,
		`uci set dhcp.guest.start="100"`,
		`uci set dhcp.guest.limit="150"`,
		`uci set dhcp.guest.leasetime="1h"`,
		`uci commit dhcp`,
		`service dnsmasq restart`,
	}
}

func FirewallProbeCommands() []string {
	return []string{"uci -q get firewall.guest"}
}

func PlanFirewallCommands(output []byte, err error) []string {
	if err == nil && strings.HasPrefix(string(output), "zone") {
		return nil
	}
	return []string{
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
	}
}

func SuccessMarkerCommand() string {
	return "echo guest-network-ok"
}

func HasSuccessMarker(output []byte) bool {
	return strings.Contains(string(output), "guest-network-ok")
}

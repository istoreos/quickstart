package networkbasics

import "fmt"

func BuildLanDHCPRangeCommands(start int, limit int) []string {
	return []string{
		fmt.Sprintf("uci set dhcp.lan.start=%v", start),
		fmt.Sprintf("uci set dhcp.lan.limit=%v", limit),
		"uci set dhcp.lan.leasetime=12h",
	}
}

func BuildWANModePendingConfigs(includeMasq bool, enableLanDhcp bool) []string {
	pending := []string{}
	if includeMasq {
		pending = append(pending, "firewall")
	}
	if enableLanDhcp {
		pending = append(pending, "dhcp")
	}
	return append(pending, "network")
}

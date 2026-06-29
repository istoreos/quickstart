package guestiface

import "fmt"

type Profile struct {
	IfaceName      string
	IfaceIndex     int
	WirelessIfName string
	SSID           string
	Encryption     string
	Key            string
	ApplyCommand   string
}

func BuildCommands(profile Profile) []string {
	key := profile.Key
	if key == "" {
		key = "goodlife"
	}
	applyCommand := profile.ApplyCommand
	if applyCommand == "" {
		applyCommand = "wifi"
	}
	return []string{
		fmt.Sprintf(`WIFI_DEV="$(uci -q get wireless.@wifi-iface[%d].device)"`, profile.IfaceIndex),
		fmt.Sprintf(`uci -q delete wireless.%s`, profile.IfaceName),
		fmt.Sprintf(`uci set wireless.%s="wifi-iface"`, profile.IfaceName),
		fmt.Sprintf(`uci set wireless.%s.device="${WIFI_DEV}"`, profile.IfaceName),
		fmt.Sprintf(`uci set wireless.%s.ifname="%s"`, profile.IfaceName, profile.WirelessIfName),
		fmt.Sprintf(`uci set wireless.%s.mode="ap"`, profile.IfaceName),
		fmt.Sprintf(`uci set wireless.%s.network="guest"`, profile.IfaceName),
		fmt.Sprintf(`uci set wireless.%s.guest="1"`, profile.IfaceName),
		fmt.Sprintf(`uci set wireless.%s.ssid="%s"`, profile.IfaceName, profile.SSID),
		fmt.Sprintf(`uci set wireless.%s.encryption="%s"`, profile.IfaceName, profile.Encryption),
		fmt.Sprintf(`uci set wireless.%s.key="%s"`, profile.IfaceName, key),
		fmt.Sprintf(`uci set wireless.%s.disabled=0`, profile.IfaceName),
		fmt.Sprintf(`uci set wireless.%s.isolate=1`, profile.IfaceName),
		fmt.Sprintf(`uci set wireless.%s.wds=1`, profile.IfaceName),
		`uci commit wireless`,
		applyCommand,
	}
}

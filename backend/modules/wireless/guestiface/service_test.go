package guestiface

import (
	"reflect"
	"testing"
)

func TestBuildCommandsCreatesGuestIfaceCommands(t *testing.T) {
	t.Parallel()

	got := BuildCommands(Profile{
		IfaceName:      "guest5g",
		IfaceIndex:     1,
		WirelessIfName: "wlan11",
		SSID:           "iStoreOS-5G-Guest",
		Encryption:     "psk2+ccmp",
		Key:            "goodlife",
		ApplyCommand:   "wifi",
	})

	want := []string{
		`WIFI_DEV="$(uci -q get wireless.@wifi-iface[1].device)"`,
		`uci -q delete wireless.guest5g`,
		`uci set wireless.guest5g="wifi-iface"`,
		`uci set wireless.guest5g.device="${WIFI_DEV}"`,
		`uci set wireless.guest5g.ifname="wlan11"`,
		`uci set wireless.guest5g.mode="ap"`,
		`uci set wireless.guest5g.network="guest"`,
		`uci set wireless.guest5g.guest="1"`,
		`uci set wireless.guest5g.ssid="iStoreOS-5G-Guest"`,
		`uci set wireless.guest5g.encryption="psk2+ccmp"`,
		`uci set wireless.guest5g.key="goodlife"`,
		`uci set wireless.guest5g.disabled=0`,
		`uci set wireless.guest5g.isolate=1`,
		`uci set wireless.guest5g.wds=1`,
		`uci commit wireless`,
		`wifi`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildCommands = %#v, want %#v", got, want)
	}
}

func TestBuildCommandsUsesDefaults(t *testing.T) {
	t.Parallel()

	got := BuildCommands(Profile{
		IfaceName:      "guest2g",
		IfaceIndex:     0,
		WirelessIfName: "ra1",
		SSID:           "iStoreOS-Guest",
		Encryption:     "mixed-psk",
	})

	if got[10] != `uci set wireless.guest2g.key="goodlife"` {
		t.Fatalf("default key command = %q", got[10])
	}
	if got[len(got)-1] != "wifi" {
		t.Fatalf("default apply command = %q, want wifi", got[len(got)-1])
	}
}

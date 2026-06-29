package networkbasics

import (
	"reflect"
	"testing"
)

func TestBuildPPPoECommands(t *testing.T) {
	got := BuildPPPoECommands("user", "pass")
	want := []string{
		"uci set network.wan.proto='pppoe'",
		"uci set network.wan.username='user'",
		"uci set network.wan.password='pass'",
		"uci del network.wan.dns",
		"uci del network.wan.peerdns",
		"uci del network.wan.defaultroute",
		"uci set network.lan.defaultroute=0",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBuildInterfaceCommandBatchesForDHCP(t *testing.T) {
	got, err := BuildInterfaceCommandBatches(InterfaceInput{
		InterfaceName: "wan",
		Proto:         "dhcp",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][]string{
		{
			"uci set network.wan.defaultroute=0",
			"uci set network.lan.defaultroute=0",
			"uci del network.wan.defaultroute",
			"uci set network.wan.proto=dhcp",
		},
		{
			"uci del network.wan.ipaddr",
			"uci del network.wan.netmask",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBuildInterfaceCommandBatchesForStatic(t *testing.T) {
	got, err := BuildInterfaceCommandBatches(InterfaceInput{
		InterfaceName: "wan",
		Proto:         "static",
		Netmask:       "255.255.255.0",
		IP:            "10.0.0.2",
		Gateway:       "10.0.0.1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := [][]string{
		{
			"uci set network.wan.defaultroute=0",
			"uci set network.lan.defaultroute=0",
			"uci del network.wan.defaultroute",
			"uci set network.wan.proto=static",
		},
		{
			"uci del network.wan.ipaddr",
			"uci del network.wan.netmask",
			"uci set network.wan.ipaddr='10.0.0.2'",
			"uci set network.wan.netmask='255.255.255.0'",
		},
		{
			"uci set network.wan.gateway=10.0.0.1",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBuildInterfaceCommandBatchesValidatesStaticInput(t *testing.T) {
	_, err := BuildInterfaceCommandBatches(InterfaceInput{
		InterfaceName: "wan",
		Proto:         "static",
		Netmask:       "255.255.255.0",
		IP:            "not-ip",
	})
	if err == nil || err.Error() != "not a valid IP" {
		t.Fatalf("expected invalid IP error, got %v", err)
	}

	_, err = BuildInterfaceCommandBatches(InterfaceInput{
		InterfaceName: "wan",
		Proto:         "static",
		Netmask:       "0.0.0.0",
		IP:            "10.0.0.2",
	})
	if err == nil || err.Error() != "not a valid NetMask" {
		t.Fatalf("expected invalid netmask error, got %v", err)
	}
}

func TestBuildDNSCommandBatchesForAuto(t *testing.T) {
	got := BuildDNSCommandBatches("wan", "auto", nil)
	want := [][]string{{
		"uci del network.wan.dns",
		"uci del network.wan.peerdns",
		"uci del network.lan.dns",
		"uci del network.lan.peerdns",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBuildDNSCommandBatchesForManualSkipsEmptyIPs(t *testing.T) {
	got := BuildDNSCommandBatches("wan", "manual", []string{"1.1.1.1", "", "8.8.8.8"})
	want := [][]string{
		{
			"uci del network.wan.dns",
			"uci del network.wan.peerdns",
			"uci del network.lan.dns",
			"uci del network.lan.peerdns",
		},
		{
			"uci del network.wan.dns",
			"uci set network.wan.peerdns=0",
			"uci add_list network.wan.dns='1.1.1.1'",
			"uci add_list network.wan.dns='8.8.8.8'",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBuildDNSCommandBatchesForManualWithoutIPsOnlyResetsDefaults(t *testing.T) {
	got := BuildDNSCommandBatches("wan", "manual", nil)
	want := [][]string{{
		"uci del network.wan.dns",
		"uci del network.wan.peerdns",
		"uci del network.lan.dns",
		"uci del network.lan.peerdns",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

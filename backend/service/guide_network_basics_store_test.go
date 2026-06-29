package service

import (
	"context"
	"errors"
	"sync"
	"testing"
)

var guideNetworkBasicsReaderTestMu sync.Mutex

func TestBuildGuideNetworkBasicsLANRange(t *testing.T) {
	start, end := buildGuideNetworkBasicsLANRange("192.168.100.1", "100", "150")
	if start != "192.168.100.100" || end != "192.168.100.249" {
		t.Fatalf("unexpected DHCP range: start=%q end=%q", start, end)
	}
}

func TestDefaultGuideNetworkBasicsReaderReadsDefaultOutboundAndDNSConfig(t *testing.T) {
	guideNetworkBasicsReaderTestMu.Lock()
	defer guideNetworkBasicsReaderTestMu.Unlock()

	originalOutbound := readGuideNetworkBasicsOutboundInterface
	originalDNS := readGuideNetworkBasicsNetDNSClient
	defer func() {
		readGuideNetworkBasicsOutboundInterface = originalOutbound
		readGuideNetworkBasicsNetDNSClient = originalDNS
	}()

	readGuideNetworkBasicsOutboundInterface = func() (*DefaultInterface, error) {
		return &DefaultInterface{interfaceName: "wan", deviceName: "eth0", proto: "dhcp"}, nil
	}
	readGuideNetworkBasicsNetDNSClient = func(net string) (string, []string) {
		if net != "wan" {
			t.Fatalf("unexpected network name: %s", net)
		}
		return "manual", []string{"1.1.1.1", "8.8.8.8"}
	}

	reader := newDefaultGuideNetworkBasicsReader()

	defaultIf, err := reader.ReadDefaultOutboundInterface(context.Background())
	if err != nil {
		t.Fatalf("unexpected default outbound error: %v", err)
	}
	if defaultIf.InterfaceName != "wan" || defaultIf.DeviceName != "eth0" || defaultIf.Proto != "dhcp" {
		t.Fatalf("unexpected default outbound snapshot: %#v", defaultIf)
	}

	dns, err := reader.ReadDNSConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected DNS snapshot error: %v", err)
	}
	if dns.InterfaceName != "wan" || dns.DNSProto != "manual" || len(dns.ManualDNSIP) != 2 || dns.ManualDNSIP[0] != "1.1.1.1" || dns.ManualDNSIP[1] != "8.8.8.8" {
		t.Fatalf("unexpected DNS snapshot: %#v", dns)
	}
}

func TestDefaultGuideNetworkBasicsReaderReadsWANRuntimeSnapshot(t *testing.T) {
	guideNetworkBasicsReaderTestMu.Lock()
	defer guideNetworkBasicsReaderTestMu.Unlock()

	originalStatus := readGuideNetworkBasicsWANStatus
	defer func() {
		readGuideNetworkBasicsWANStatus = originalStatus
	}()

	readGuideNetworkBasicsWANStatus = func(ctx context.Context, interfaceName string, blk *ubusNetworkInterfaceStatus) error {
		if interfaceName != "wan" {
			t.Fatalf("unexpected interface name: %s", interfaceName)
		}
		blk.Ipv4 = []*ubusNetworkInterfaceAddress{{Address: "10.0.0.2", Mask: 24}}
		blk.Route = []*ubusNetworkInterfaceRoute{{Target: "0.0.0.0", Mask: 0, Nexthop: "10.0.0.1"}}
		return nil
	}

	reader := newDefaultGuideNetworkBasicsReader()
	snapshot, err := reader.ReadWANRuntime(context.Background(), "wan")
	if err != nil {
		t.Fatalf("unexpected WAN runtime error: %v", err)
	}
	if snapshot.StaticIP != "10.0.0.2" || snapshot.SubnetMask != "255.255.255.0" || snapshot.Gateway != "10.0.0.1" {
		t.Fatalf("unexpected WAN runtime snapshot: %#v", snapshot)
	}
}

func TestDefaultGuideNetworkBasicsReaderReadsWANAndLANConfig(t *testing.T) {
	guideNetworkBasicsReaderTestMu.Lock()
	defer guideNetworkBasicsReaderTestMu.Unlock()

	originalLast := readGuideNetworkBasicsConfigLast
	originalIPAndMask := readGuideNetworkBasicsIPAndMask
	originalDNS := readGuideNetworkBasicsNetDNSClient
	originalDHCPEnabled := readGuideNetworkBasicsLanDHCPEnabled
	defer func() {
		readGuideNetworkBasicsConfigLast = originalLast
		readGuideNetworkBasicsIPAndMask = originalIPAndMask
		readGuideNetworkBasicsNetDNSClient = originalDNS
		readGuideNetworkBasicsLanDHCPEnabled = originalDHCPEnabled
	}()

	lastValues := map[string]string{
		"network.wan.proto":   "dhcp",
		"network.wan.gateway": "10.0.0.1",
		"network.wan.username": "pppoe-user",
		"network.wan.password": "pppoe-pass",
		"dhcp.lan.start":      "100",
		"dhcp.lan.limit":      "150",
	}
	readGuideNetworkBasicsConfigLast = func(config, section, option string) (string, bool) {
		value, ok := lastValues[config+"."+section+"."+option]
		return value, ok
	}
	readGuideNetworkBasicsIPAndMask = func(net string) (string, string) {
		switch net {
		case "wan":
			return "10.0.0.2", "255.255.255.0"
		case "lan":
			return "192.168.100.1", "255.255.255.0"
		default:
			return "", ""
		}
	}
	readGuideNetworkBasicsNetDNSClient = func(net string) (string, []string) {
		if net == "wan" {
			return "manual", []string{"1.1.1.1"}
		}
		return "auto", nil
	}
	readGuideNetworkBasicsLanDHCPEnabled = func() bool { return true }

	reader := newDefaultGuideNetworkBasicsReader()

	wan := reader.ReadWANConfig(context.Background())
	if !wan.Exists || wan.WanProto != "dhcp" || wan.StaticIP != "10.0.0.2" || wan.SubnetMask != "255.255.255.0" || wan.Gateway != "10.0.0.1" || wan.DNSProto != "manual" || len(wan.ManualDNSIP) != 1 || wan.ManualDNSIP[0] != "1.1.1.1" || wan.PPPoEAccount != "pppoe-user" || wan.PPPoEPassword != "pppoe-pass" {
		t.Fatalf("unexpected WAN config snapshot: %#v", wan)
	}

	lan := reader.ReadLANConfig(context.Background())
	if lan.LanIP != "192.168.100.1" || lan.NetMask != "255.255.255.0" || !lan.EnableDhcp || lan.DhcpStart != "192.168.100.100" || lan.DhcpEnd != "192.168.100.249" {
		t.Fatalf("unexpected LAN config snapshot: %#v", lan)
	}
}

func TestDefaultGuideNetworkBasicsReaderHandlesMissingWANAndWANRuntimeErrors(t *testing.T) {
	guideNetworkBasicsReaderTestMu.Lock()
	defer guideNetworkBasicsReaderTestMu.Unlock()

	originalLast := readGuideNetworkBasicsConfigLast
	originalStatus := readGuideNetworkBasicsWANStatus
	defer func() {
		readGuideNetworkBasicsConfigLast = originalLast
		readGuideNetworkBasicsWANStatus = originalStatus
	}()

	readGuideNetworkBasicsConfigLast = func(config, section, option string) (string, bool) {
		return "", false
	}
	reader := newDefaultGuideNetworkBasicsReader()
	wan := reader.ReadWANConfig(context.Background())
	if wan.Exists {
		t.Fatalf("expected missing WAN snapshot, got %#v", wan)
	}

	expectedErr := errors.New("wan runtime failed")
	readGuideNetworkBasicsWANStatus = func(ctx context.Context, interfaceName string, blk *ubusNetworkInterfaceStatus) error {
		return expectedErr
	}
	if _, err := reader.ReadWANRuntime(context.Background(), "wan"); !errors.Is(err, expectedErr) {
		t.Fatalf("expected WAN runtime error, got %v", err)
	}
}

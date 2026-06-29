package service

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	networkstatus "github.com/linkease/quick-start/istore-backend/modules/network/status"
)

var networkStatusStoreTestMu sync.Mutex

func TestFallbackNetworkStatusIfname(t *testing.T) {
	t.Parallel()

	got := fallbackNetworkStatusIPv4(&DefaultInterface{})
	if got.interfaceName != "wan" {
		t.Fatalf("expected fallback interface wan, got %q", got.interfaceName)
	}
	if got.deviceName != "eth0" {
		t.Fatalf("expected fallback device eth0, got %q", got.deviceName)
	}

	keep := fallbackNetworkStatusIPv4(&DefaultInterface{
		interfaceName: "wwan",
		deviceName:    "usb0",
	})
	if keep.interfaceName != "wwan" {
		t.Fatalf("expected existing interface preserved, got %q", keep.interfaceName)
	}
	if keep.deviceName != "usb0" {
		t.Fatalf("expected existing device preserved, got %q", keep.deviceName)
	}
}

func TestDefaultNetworkStatusReader(t *testing.T) {
	networkStatusStoreTestMu.Lock()
	defer networkStatusStoreTestMu.Unlock()

	oldOutbound := networkStatusOutboundInterfaces
	oldLoad := networkStatusLoadConfig
	oldGetLast := networkStatusGetLast
	oldGet := networkStatusGet
	defer func() {
		networkStatusOutboundInterfaces = oldOutbound
		networkStatusLoadConfig = oldLoad
		networkStatusGetLast = oldGetLast
		networkStatusGet = oldGet
	}()

	networkStatusOutboundInterfaces = func() (*DefaultInterfaces, error) {
		return &DefaultInterfaces{
			ipv4: &DefaultInterface{
				interfaceName: "",
				deviceName:    "",
				ip:            "10.0.0.2",
				mask:          24,
				proto:         "dhcp",
				dns:           []string{"1.1.1.1"},
				gateway:       "10.0.0.1",
				upTime:        90,
			},
			ipv6: &DefaultInterface{ip: "fe80::2"},
		}, nil
	}
	var loaded []string
	networkStatusLoadConfig = func(config string, reload bool) error {
		loaded = append(loaded, config)
		return nil
	}
	networkStatusGetLast = func(config string, section string, option string) (string, bool) {
		return "", false
	}
	networkStatusGet = func(config string, section string, option string) ([]string, bool) {
		return nil, false
	}

	snapshot, dnsConfig, err := newDefaultNetworkStatusReader().Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if !reflect.DeepEqual(loaded, []string{"network"}) {
		t.Fatalf("expected network config load, got %#v", loaded)
	}
	if snapshot.ResolvedIfName != "wan" {
		t.Fatalf("expected fallback interface wan, got %q", snapshot.ResolvedIfName)
	}
	if snapshot.IPv4 == nil || snapshot.IPv4.Address != "10.0.0.2" || snapshot.IPv4.Mask != 24 {
		t.Fatalf("expected ipv4 snapshot mapped, got %#v", snapshot.IPv4)
	}
	if snapshot.IPv6Addr != "fe80::2" {
		t.Fatalf("expected ipv6 mapped, got %q", snapshot.IPv6Addr)
	}
	if dnsConfig.Proto != "auto" || !reflect.DeepEqual(dnsConfig.DNSList, []string{"1.1.1.1"}) {
		t.Fatalf("unexpected dns config: %#v", dnsConfig)
	}
}

func TestDefaultNetworkStatusReaderLoadsManualDNS(t *testing.T) {
	networkStatusStoreTestMu.Lock()
	defer networkStatusStoreTestMu.Unlock()

	oldOutbound := networkStatusOutboundInterfaces
	oldLoad := networkStatusLoadConfig
	oldGetLast := networkStatusGetLast
	oldGet := networkStatusGet
	defer func() {
		networkStatusOutboundInterfaces = oldOutbound
		networkStatusLoadConfig = oldLoad
		networkStatusGetLast = oldGetLast
		networkStatusGet = oldGet
	}()

	networkStatusOutboundInterfaces = func() (*DefaultInterfaces, error) {
		return &DefaultInterfaces{
			ipv4: &DefaultInterface{
				interfaceName: "wan",
				deviceName:    "eth0",
				proto:         "dhcp",
				dns:           []string{"1.1.1.1"},
			},
		}, nil
	}
	networkStatusLoadConfig = func(config string, reload bool) error { return nil }
	networkStatusGetLast = func(config string, section string, option string) (string, bool) {
		if config == "network" && section == "wan" && option == "peerdns" {
			return "0", true
		}
		return "", false
	}
	networkStatusGet = func(config string, section string, option string) ([]string, bool) {
		if config == "network" && section == "wan" && option == "dns" {
			return []string{"9.9.9.9"}, true
		}
		return nil, false
	}

	_, dnsConfig, err := newDefaultNetworkStatusReader().Read(context.Background())
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}
	if dnsConfig.Proto != "manual" || !reflect.DeepEqual(dnsConfig.DNSList, []string{"9.9.9.9"}) {
		t.Fatalf("unexpected manual dns config: %#v", dnsConfig)
	}
}

func TestDefaultNetworkOnlineStatusChecker(t *testing.T) {
	t.Parallel()

	checker := &NetworkOnlineChecker{
		status:    NetworkOnlineFailedDns,
		cacheKey:  "10.0.0.2|10.0.0.1|1.1.1.1|",
		lastCheck: time.Now(),
	}
	got, err := newDefaultNetworkOnlineStatusChecker(checker).GetStatus("10.0.0.2", "10.0.0.1", []string{"1.1.1.1"})
	if err != nil {
		t.Fatalf("unexpected checker error: %v", err)
	}
	if got != networkstatus.OnlineFailedDNS {
		t.Fatalf("expected delegated checker status, got %v", got)
	}
}

func TestDefaultNetworkSetupMarker(t *testing.T) {
	networkStatusStoreTestMu.Lock()
	defer networkStatusStoreTestMu.Unlock()

	oldMark := networkStatusMarkSetupFinish
	defer func() {
		networkStatusMarkSetupFinish = oldMark
	}()

	called := false
	networkStatusMarkSetupFinish = func(ctx context.Context) {
		called = true
	}

	newDefaultNetworkSetupMarker().MarkSetupFinish(context.Background())
	if !called {
		t.Fatal("expected setup marker to delegate to markSetupFinish")
	}
}

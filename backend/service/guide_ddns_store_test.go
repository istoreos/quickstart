package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	simplejson "github.com/bitly/go-simplejson"
)

var guideDDNSReaderTestMu sync.Mutex

func TestDefaultGuideDDNSReaderReadsPendingChanges(t *testing.T) {
	guideDDNSReaderTestMu.Lock()
	defer guideDDNSReaderTestMu.Unlock()

	originalUbus := readGuideDDNSUbusCall
	defer func() { readGuideDDNSUbusCall = originalUbus }()

	readGuideDDNSUbusCall = func(ctx context.Context, arg string) (*simplejson.Json, error) {
		if arg != `uci changes {"config":"ddns","ubus_rpc_session":"sid"}` {
			t.Fatalf("unexpected ubus arg: %s", arg)
		}
		j := simplejson.New()
		j.Set("changes", []interface{}{"set ddns.foo=bar"})
		return j, nil
	}

	reader := newDefaultGuideDDNSReader()
	hasPending, err := reader.ReadDDNSPendingChanges(context.Background(), "sid")
	if err != nil {
		t.Fatalf("unexpected pending-change error: %v", err)
	}
	if !hasPending {
		t.Fatalf("expected pending changes to be true")
	}

	readGuideDDNSUbusCall = func(ctx context.Context, arg string) (*simplejson.Json, error) {
		j := simplejson.New()
		j.Set("changes", []interface{}{})
		return j, nil
	}
	hasPending, err = reader.ReadDDNSPendingChanges(context.Background(), "sid")
	if err != nil {
		t.Fatalf("unexpected empty pending-change error: %v", err)
	}
	if hasPending {
		t.Fatalf("expected pending changes to be false")
	}
}

func TestDefaultGuideDDNSReaderReadsOutboundInterfacesAndPublicIPv4(t *testing.T) {
	guideDDNSReaderTestMu.Lock()
	defer guideDDNSReaderTestMu.Unlock()

	originalOutbound := readGuideDDNSOutboundInterfaces
	originalPublicV4 := readGuideDDNSIsPublicIPv4
	defer func() {
		readGuideDDNSOutboundInterfaces = originalOutbound
		readGuideDDNSIsPublicIPv4 = originalPublicV4
	}()

	readGuideDDNSOutboundInterfaces = func() (*DefaultInterfaces, error) {
		return &DefaultInterfaces{
			ipv4: &DefaultInterface{
				interfaceName: "wan",
				deviceName:    "eth0",
				ip:            "1.2.3.4",
				gateway:       "1.2.3.1",
				proto:         "dhcp",
				dns:           []string{"1.1.1.1"},
			},
			ipv6: &DefaultInterface{
				interfaceName: "wan6",
				deviceName:    "eth1",
				ip:            "2408::1",
				gateway:       "2408::2",
				proto:         "dhcpv6",
				dns:           []string{"2408::53"},
			},
		}, nil
	}
	readGuideDDNSIsPublicIPv4 = func(ip string) bool {
		return ip == "1.2.3.4"
	}

	reader := newDefaultGuideDDNSReader()
	snapshot, err := reader.ReadOutboundInterfaces(context.Background())
	if err != nil {
		t.Fatalf("unexpected outbound-interface error: %v", err)
	}
	if snapshot.IPv4 == nil || snapshot.IPv4.InterfaceName != "wan" || snapshot.IPv4.IP != "1.2.3.4" || snapshot.IPv4.Gateway != "1.2.3.1" {
		t.Fatalf("unexpected ipv4 snapshot: %#v", snapshot.IPv4)
	}
	if snapshot.IPv6 == nil || snapshot.IPv6.InterfaceName != "wan6" || snapshot.IPv6.IP != "2408::1" || snapshot.IPv6.Gateway != "2408::2" {
		t.Fatalf("unexpected ipv6 snapshot: %#v", snapshot.IPv6)
	}
	if !reader.IsPublicIPv4(snapshot.IPv4.IP) {
		t.Fatalf("expected ipv4 to be considered public")
	}
}

func TestDefaultGuideDDNSReaderReadsDdnstoConfigAndPropagatesErrors(t *testing.T) {
	guideDDNSReaderTestMu.Lock()
	defer guideDDNSReaderTestMu.Unlock()

	originalUciGet := readGuideDDNSUciGet
	originalOutbound := readGuideDDNSOutboundInterfaces
	defer func() {
		readGuideDDNSUciGet = originalUciGet
		readGuideDDNSOutboundInterfaces = originalOutbound
	}()

	readGuideDDNSUciGet = func(ctx context.Context, location string) (string, error) {
		switch location {
		case "ddnsto.@ddnsto[0].enabled":
			return "1", nil
		case "ddnsto.@ddnsto[0].token":
			return "token-abc", nil
		case "ddnsto.@ddnsto[0].address":
			return "https://demo.example.com", nil
		default:
			t.Fatalf("unexpected uci get location: %s", location)
			return "", nil
		}
	}

	reader := newDefaultGuideDDNSReader()
	cfg, err := reader.ReadDdnstoConfig(context.Background())
	if err != nil {
		t.Fatalf("unexpected ddnsto config error: %v", err)
	}
	if !cfg.Enabled || cfg.Token != "token-abc" || cfg.Address != "https://demo.example.com" {
		t.Fatalf("unexpected ddnsto config: %#v", cfg)
	}

	uciErr := errors.New("uci get failed")
	readGuideDDNSUciGet = func(ctx context.Context, location string) (string, error) {
		return "", uciErr
	}
	if _, err := reader.ReadDdnstoConfig(context.Background()); !errors.Is(err, uciErr) {
		t.Fatalf("expected ddnsto config error, got %v", err)
	}

	outboundErr := errors.New("outbound failed")
	readGuideDDNSOutboundInterfaces = func() (*DefaultInterfaces, error) {
		return nil, outboundErr
	}
	if _, err := reader.ReadOutboundInterfaces(context.Background()); !errors.Is(err, outboundErr) {
		t.Fatalf("expected outbound-interface error, got %v", err)
	}
}

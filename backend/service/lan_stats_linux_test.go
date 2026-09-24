//go:build linux

package service

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func mustPrefix(t *testing.T, raw string) netip.Prefix {
	t.Helper()
	prefix, err := netip.ParsePrefix(raw)
	if err != nil {
		t.Fatalf("parse prefix %q: %v", raw, err)
	}
	return prefix
}

func TestReadProcConntrackFlowsParsesOriginalAndReplyCounters(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nf_conntrack")
	content := "ipv4 2 tcp 6 30 ESTABLISHED src=192.168.100.20 dst=1.1.1.1 sport=1 dport=443 packets=2 bytes=120 src=1.1.1.1 dst=192.168.9.215 sport=443 dport=1 packets=3 bytes=340 [ASSURED]\n" +
		"ipv6 2 udp 17 20 src=fd5f:e357:7969::20 dst=2606:4700:4700::1111 sport=2 dport=53 packets=1 bytes=80 src=2606:4700:4700::1111 dst=fd5f:e357:7969::20 sport=53 dport=2 packets=1 bytes=160\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	flows, err := readProcConntrackFlows(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(flows) != 2 || flows[0].source.String() != "192.168.100.20" || flows[0].upstream != 120 || flows[0].downstream != 340 {
		t.Fatalf("IPv4 flow = %#v", flows)
	}
	if flows[1].source.String() != "fd5f:e357:7969::20" || flows[1].upstream != 80 || flows[1].downstream != 160 {
		t.Fatalf("IPv6 flow = %#v", flows[1])
	}
}

func mustAddr(t *testing.T, raw string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		t.Fatalf("parse address %q: %v", raw, err)
	}
	return addr
}

func TestLanStatsRecordsOnlyLANEndpoints(t *testing.T) {
	stats := &LanStats{
		hosts:       make(map[string]*LanStatHost),
		lanPrefixes: []netip.Prefix{mustPrefix(t, "192.168.9.0/24"), mustPrefix(t, "fd12:3456::/64")},
	}

	stats.recordFlow(mustAddr(t, "192.168.9.20"), mustAddr(t, "1.1.1.1"), 100, 200)
	stats.recordFlow(mustAddr(t, "2606:4700:4700::1111"), mustAddr(t, "fd12:3456::20"), 300, 400)

	if len(stats.hosts) != 2 {
		t.Fatalf("expected only two LAN endpoints, got %#v", stats.hosts)
	}
	if _, ok := stats.hosts["1.1.1.1"]; ok {
		t.Fatal("public destination must not enter LAN traffic cache")
	}
	if _, ok := stats.hosts["2606:4700:4700::1111"]; ok {
		t.Fatal("public IPv6 source must not enter LAN traffic cache")
	}
	if got := stats.hosts["192.168.9.20"]; got == nil || got.txTotal != 100 || got.rxTotal != 200 {
		t.Fatalf("unexpected IPv4 LAN counters: %#v", got)
	}
	if got := stats.hosts["fd12:3456::20"]; got == nil || got.txTotal != 400 || got.rxTotal != 300 {
		t.Fatalf("unexpected IPv6 LAN counters: %#v", got)
	}
}

func TestLanStatsCacheIsBoundedAndExpiresIdleHosts(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	stats := &LanStats{
		hosts:       make(map[string]*LanStatHost),
		lanPrefixes: []netip.Prefix{mustPrefix(t, "192.168.0.0/16")},
		maxHosts:    2,
		hostIdleTTL: time.Minute,
		now:         func() time.Time { return now },
	}
	publicIP := mustAddr(t, "1.1.1.1")

	stats.recordFlow(mustAddr(t, "192.168.1.10"), publicIP, 1, 1)
	now = now.Add(time.Second)
	stats.recordFlow(mustAddr(t, "192.168.1.11"), publicIP, 1, 1)
	now = now.Add(time.Second)
	stats.recordFlow(mustAddr(t, "192.168.1.12"), publicIP, 1, 1)

	if len(stats.hosts) != 2 {
		t.Fatalf("expected bounded cache of two hosts, got %d", len(stats.hosts))
	}
	if _, ok := stats.hosts["192.168.1.10"]; ok {
		t.Fatal("expected least recently seen host to be evicted")
	}

	now = now.Add(2 * time.Minute)
	stats.pruneHosts(now)
	if len(stats.hosts) != 0 {
		t.Fatalf("expected idle hosts to expire, got %#v", stats.hosts)
	}
}

func TestLANPrefixesFromStatusIncludesIPv4AndIPv6Networks(t *testing.T) {
	status := &ubusLanStatus{network: ubusNetworkInterface{
		Ipv4: []*ubusNetworkInterfaceAddress{{Address: "192.168.9.1", Mask: 24}},
		Ipv6: []*ubusNetworkInterfaceAddress{{Address: "fd12:3456::1", Mask: 64}},
	}}

	prefixes := lanPrefixesFromStatus(status)
	stats := &LanStats{lanPrefixes: prefixes}
	for _, raw := range []string{"192.168.9.215", "fd12:3456::80"} {
		if !stats.isLanIP(mustAddr(t, raw)) {
			t.Errorf("expected %s to be classified as LAN from ubus status", raw)
		}
	}
	for _, raw := range []string{"192.168.10.1", "fd12:3457::1", "8.8.8.8"} {
		if stats.isLanIP(mustAddr(t, raw)) {
			t.Errorf("expected %s to be outside LAN prefixes", raw)
		}
	}
}

func TestRefreshLANPrefixesKeepsLastKnownPrefixesOnEmptyRead(t *testing.T) {
	knownPrefix := mustPrefix(t, "192.168.9.0/24")
	stats := &LanStats{
		lanPrefixes: []netip.Prefix{knownPrefix},
		lanPrefixReader: func() ([]netip.Prefix, error) {
			return nil, nil
		},
	}

	stats.refreshLANPrefixes()

	if len(stats.lanPrefixes) != 1 || stats.lanPrefixes[0] != knownPrefix {
		t.Fatalf("expected last known LAN prefix to survive an empty read, got %#v", stats.lanPrefixes)
	}
}

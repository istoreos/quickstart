package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

type fakeLANScopeReader struct {
	scopes []*models.LANScope
	err    error
}

func (reader fakeLANScopeReader) Read(context.Context) ([]*models.LANScope, error) {
	return reader.scopes, reader.err
}

func testLANScopeModule() *LANScopeModule {
	return &LANScopeModule{reader: fakeLANScopeReader{scopes: []*models.LANScope{
		{NetworkID: "lan", Device: "br-lan", State: "up", Version: "lan-version"},
		{NetworkID: "office", Device: "br-office", State: "up", Version: "office-version"},
	}}, now: time.Now}
}

func testLANConfig(t *testing.T, network, dhcp, firewall string) uci.Tree {
	t.Helper()
	dir := t.TempDir()
	for name, value := range map[string]string{"network": network, "dhcp": dhcp, "firewall": firewall} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	tree := uci.NewTree(dir)
	for _, config := range []string{"network", "dhcp", "firewall"} {
		if err := tree.LoadConfig(config, true); err != nil {
			t.Fatal(err)
		}
	}
	return tree
}

func testLANRuntime(name, device, address string) lanRuntimeInterface {
	return lanRuntimeInterface{Up: true, ubusNetworkInterface: ubusNetworkInterface{
		Name: name, L3Device: device, Proto: "static",
		Ipv4: []*ubusNetworkInterfaceAddress{{Address: address, Mask: 24}},
	}}
}

func TestLANScopeDiscoversLogicalNetworksAndDHCPBindings(t *testing.T) {
	tree := testLANConfig(t, `
config interface 'office'
 option proto 'static'
 option device 'br-office'
config interface 'guest'
 option proto 'static'
 option device 'br-guest'
config interface 'uplink'
 option proto 'static'
 option device 'eth0'
`, `
config dnsmasq 'server'
 option leasefile '/tmp/office.leases'
config dhcp 'office_pool'
 option interface 'office'
config dhcp 'guest_pool'
 option interface 'guest'
 option instance '0'
`, `
config zone 'home'
 option name 'trusted'
 list network 'office'
 list network 'guest'
config zone 'outside'
 option name 'wan'
 list network 'uplink'
`)
	scopes := buildLANScopes(tree, []lanRuntimeInterface{
		testLANRuntime("office", "br-office", "192.0.2.1"),
		testLANRuntime("guest", "br-guest", "198.51.100.1"),
		testLANRuntime("uplink", "eth0", "203.0.113.1"),
	})
	if len(scopes) != 2 || scopes[0].NetworkID != "guest" || scopes[1].NetworkID != "office" {
		t.Fatalf("unexpected LAN inventory: %#v", scopes)
	}
	office := scopes[1]
	if office.IPv4Prefixes[0] != "192.0.2.0/24" || office.FirewallZones[0] != "trusted" || office.ReadOnlyReason != "" {
		t.Fatalf("office mapping: %#v", office)
	}
	if office.DHCPBindings[0].Section != "office_pool" || office.DHCPBindings[0].Instance != "server" || office.DHCPBindings[0].LeaseFile != "/tmp/office.leases" {
		t.Fatalf("DHCP binding: %#v", office.DHCPBindings)
	}
	if scopes[0].DHCPBindings[0].Instance != "server" {
		t.Fatalf("numeric DHCP instance was not resolved: %#v", scopes[0])
	}
}

func TestLANScopeDoesNotTreatPortsAsNetworksOrDuplicateAliases(t *testing.T) {
	tree := testLANConfig(t, `
config device 'bridge'
 option name 'br-lan'
 option type 'bridge'
 list ports 'lan1'
 list ports 'lan2'
config interface 'lan'
 option proto 'static'
 option device 'br-lan'
config interface 'lan6'
 option proto 'dhcpv6'
 option device '@lan'
config interface 'guest'
 option proto 'static'
 option device 'br-lan.20'
config interface 'vpn'
 option proto 'wireguard'
`, "", "")
	alias := testLANRuntime("lan6", "br-lan", "")
	scopes := buildLANScopes(tree, []lanRuntimeInterface{
		testLANRuntime("lan", "br-lan", "192.0.2.1"), alias,
		testLANRuntime("guest", "br-lan.20", "198.51.100.1"),
	})
	if len(scopes) != 2 || scopes[0].NetworkID != "lan" || !lanContainsAlias(scopes[0].Aliases, "lan6") {
		t.Fatalf("port/alias inventory: %#v", scopes)
	}
	if scopes[0].ReadOnlyReason != "" {
		t.Fatal("IPv6 companion alias must not disable the ordinary LAN")
	}
}

func TestLANScopeRetainsAddresslessAndStoppedNetworks(t *testing.T) {
	tree := testLANConfig(t, "config interface 'office'\n option proto 'static'\n option device 'eth3'\n", "", "")
	runtime := testLANRuntime("office", "eth3", "")
	scopes := buildLANScopes(tree, []lanRuntimeInterface{runtime})
	if len(scopes) != 1 || scopes[0].ReadOnlyReason != "network_address_unavailable" {
		t.Fatalf("addressless configured network missing: %#v", scopes)
	}
	scopes = buildLANScopes(tree, nil)
	if len(scopes) != 1 || scopes[0].ReadOnlyReason != "network_down" {
		t.Fatalf("stopped configured network missing: %#v", scopes)
	}
}

func TestLANScopeCachesWithoutExposingMutableState(t *testing.T) {
	module := testLANScopeModule()
	first, err := module.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	first[0].NetworkID = "forged"
	second, err := module.Resolve(context.Background(), "lan")
	if err != nil || second.NetworkID != "lan" {
		t.Fatalf("caller mutated cached scope: %#v, %v", second, err)
	}
	module.reader = fakeLANScopeReader{err: errors.New("runtime unavailable")}
	if _, err := module.resolve(context.Background(), "lan", true); err == nil {
		t.Fatal("mutation accepted stale discovery after a reader failure")
	}
}

func TestLANRequestsPreserveLegacyAndRejectAmbiguousTargets(t *testing.T) {
	for _, item := range []struct{ query, body, want string }{
		{"", "{}", "lan"},
		{"?network=office", "{}", "office"},
		{"?lan=office", "{}", "office"},
		{"?network=office", `{"networkId":"office"}`, "office"},
		{"", `{"networkId":"office"}`, "office"},
		{"?network=office", `{"networkId":"lan"}`, ""},
		{"?network=office&lan=lan", "{}", ""},
		{"?network=office&network=lan", "{}", ""},
		{"?network=", "{}", ""},
	} {
		request := httptest.NewRequest(http.MethodPost, "/test"+item.query, strings.NewReader(item.body))
		got, err := requestedLAN(request)
		if (item.want == "") != (err != nil) || got != item.want {
			t.Errorf("query=%s body=%s: got=%q err=%v", item.query, item.body, got, err)
		}
		if item.want != "" {
			body, _ := io.ReadAll(request.Body)
			if string(body) != item.body {
				t.Fatal("scope parsing consumed the business request")
			}
		}
	}
}

func TestLANRequestBoundaryRejectsUnsafeOrUnavailableTargets(t *testing.T) {
	backend := &ServiceBackend{lanScopes: testLANScopeModule()}
	for _, query := range []string{"?network=missing", "?network=../lan", "?network=all"} {
		request := httptest.NewRequest(http.MethodPost, "/cgi-bin/luci/istore/lanctrl/v2/device-policy/"+query, strings.NewReader("{}"))
		if _, err := backend.PrepareLANRequest(context.Background(), request); err == nil {
			t.Errorf("unsafe/unadapted target entered legacy write: %s", query)
		}
	}
	backend.lanScopes.reader = fakeLANScopeReader{scopes: []*models.LANScope{{NetworkID: "office", Device: "br-office"}}}
	backend.lanScopes.cached = nil
	request := httptest.NewRequest(http.MethodPost, "/cgi-bin/luci/istore/lanctrl/v2/device-policy/", strings.NewReader("{}"))
	if _, err := backend.PrepareLANRequest(context.Background(), request); err == nil {
		t.Fatal("missing legacy LAN was silently replaced with the only other network")
	}
}

func TestLANConcurrentRequestsHaveIndependentContexts(t *testing.T) {
	backend := &ServiceBackend{lanScopes: testLANScopeModule()}
	var workers sync.WaitGroup
	for i := 0; i < 20; i++ {
		for _, network := range []string{"lan", "office"} {
			workers.Add(1)
			go func(network string) {
				defer workers.Done()
				request := httptest.NewRequest(http.MethodGet, "/cgi-bin/luci/istore/lanctrl/v2/router-context/?network="+network, nil)
				ctx, err := backend.PrepareLANRequest(context.Background(), request)
				if err != nil || selectedLAN(ctx) != network {
					t.Errorf("request scope %q: %v", network, err)
				}
			}(network)
		}
	}
	workers.Wait()
	if selectedLAN(context.Background()) != "lan" {
		t.Fatal("a request changed the legacy default")
	}
}

func TestLANScopeRenamedUplinkNeedsPositiveRoleEvidence(t *testing.T) {
	network := "config interface 'office'\n option proto 'static'\n option device 'br-office'\nconfig interface 'uplink'\n option proto 'dhcp'\n option device 'eth-uplink'\n"
	firewall := "config zone 'local'\n option name 'lan'\n list network 'office'\nconfig zone 'outside'\n option name 'internet'\n option input 'REJECT'\n option forward 'REJECT'\n list network 'uplink'\nconfig forwarding 'out'\n option src 'lan'\n option dest 'internet'\n"
	runtime := []lanRuntimeInterface{testLANRuntime("office", "br-office", "192.0.2.1"), testLANRuntime("uplink", "eth-uplink", "198.51.100.2")}
	runtime[0].Route = []*ubusNetworkInterfaceRoute{{Target: "0.0.0.0", Mask: 0, Nexthop: "192.0.2.254"}}
	runtime[1].Route = []*ubusNetworkInterfaceRoute{{Target: "0.0.0.0", Mask: 0, Nexthop: "198.51.100.1"}}
	for _, tc := range []struct {
		name, dhcp string
		count      int
	}{{"renamed_uplink", "", 1}, {"configured_local_dhcp", "config dhcp 'guest_server'\n option interface 'uplink'\n", 2}} {
		t.Run(tc.name, func(t *testing.T) {
			tree := testLANConfig(t, network, tc.dhcp, firewall)
			scopes := buildLANScopes(tree, runtime)
			if len(scopes) != tc.count || scopes[0].NetworkID != "office" {
				t.Fatalf("scopes=%+v", scopes)
			}
		})
	}
}

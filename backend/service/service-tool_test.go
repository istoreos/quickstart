package service

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func mockOutboundInterfaceDump(t *testing.T, dump string) {
	t.Helper()
	dir := t.TempDir()
	fixture := filepath.Join(dir, "dump.json")
	if err := os.WriteFile(fixture, []byte(dump), 0600); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
[ "$#" -eq 4 ] && [ "$1" = "-S" ] && [ "$2" = "call" ] && [ "$3" = "network.interface" ] && [ "$4" = "dump" ] || exit 2
exec /bin/cat "$QUICKSTART_TEST_UBUS_DUMP"
`
	if err := os.WriteFile(filepath.Join(dir, "ubus"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("QUICKSTART_TEST_UBUS_DUMP", fixture)
}

func TestOutboundInterfaceMetrics(t *testing.T) {
	tests := []struct {
		name    string
		dump    string
		ipv4    string
		ipv6    string
		gateway string
	}{
		{
			name: "cellular WAN after wired WAN",
			dump: `{"interface":[
{"interface":"wan","metric":100,"device":"eth0","ipv4-address":[{"address":"192.0.2.2","mask":24}],"route":[{"target":"0.0.0.0","mask":0,"nexthop":"192.0.2.1"}]},
{"interface":"wwan_5g_0","metric":10,"device":"usb0","l3_device":"usb0","proto":"dhcp","uptime":90,"dns-server":["198.51.100.1"],"ipv4-address":[{"address":"198.51.100.2","mask":24}],"route":[{"target":"0.0.0.0","mask":0,"nexthop":"198.51.100.1"}]}
]}`,
			ipv4: "wwan_5g_0",
		},
		{
			name: "wired WAN with lower metric wins",
			dump: `{"interface":[
{"interface":"cellular","metric":100,"ipv4-address":[{"address":"198.51.100.2"}],"route":[{"target":"0.0.0.0"}]},
{"interface":"wan","metric":10,"ipv4-address":[{"address":"192.0.2.2"}],"route":[{"target":"0.0.0.0"}]}
]}`,
			ipv4: "wan",
		},
		{
			name: "route metric overrides interface metric",
			dump: `{"interface":[
{"interface":"wan","metric":1,"ipv4-address":[{"address":"192.0.2.2"}],"route":[{"target":"0.0.0.0","metric":100}]},
{"interface":"cellular","metric":100,"ipv4-address":[{"address":"198.51.100.2"}],"route":[{"target":"0.0.0.0","metric":10}]}
]}`,
			ipv4: "cellular",
		},
		{
			name: "explicit zero route metric",
			dump: `{"interface":[
{"interface":"wan","metric":10,"ipv4-address":[{"address":"192.0.2.2"}],"route":[{"target":"0.0.0.0"}]},
{"interface":"cellular","metric":100,"ipv4-address":[{"address":"198.51.100.2"}],"route":[{"target":"0.0.0.0","metric":0}]}
]}`,
			ipv4: "cellular",
		},
		{
			name: "missing metrics default to zero",
			dump: `{"interface":[
{"interface":"wan","metric":10,"ipv4-address":[{"address":"192.0.2.2"}],"route":[{"target":"0.0.0.0"}]},
{"interface":"cellular","ipv4-address":[{"address":"198.51.100.2"}],"route":[{"target":"0.0.0.0"}]}
]}`,
			ipv4: "cellular",
		},
		{
			name: "equal metrics preserve dump order",
			dump: `{"interface":[
{"interface":"wan","metric":10,"ipv4-address":[{"address":"192.0.2.2"}],"route":[{"target":"0.0.0.0"}]},
{"interface":"cellular","metric":10,"ipv4-address":[{"address":"198.51.100.2"}],"route":[{"target":"0.0.0.0"}]}
]}`,
			ipv4: "wan",
		},
		{
			name: "lowest metric route on one interface",
			dump: `{"interface":[
{"interface":"wan","ipv4-address":[{"address":"192.0.2.2"}],"route":[{"target":"0.0.0.0","metric":100,"nexthop":"192.0.2.1"},{"target":"0.0.0.0","metric":10,"nexthop":"192.0.2.254"}]}
]}`,
			ipv4:    "wan",
			gateway: "192.0.2.254",
		},
		{
			name: "continue after finding both address families",
			dump: `{"interface":[
{"interface":"wan","metric":100,"ipv4-address":[{"address":"192.0.2.2"}],"ipv6-address":[{"address":"2001:db8:1::2"}],"route":[{"target":"0.0.0.0"},{"target":"::"}]},
{"interface":"cellular","metric":10,"ipv4-address":[{"address":"198.51.100.2"}],"ipv6-address":[{"address":"2001:db8:2::2"}],"route":[{"target":"0.0.0.0"},{"target":"::"}]}
]}`,
			ipv4: "cellular",
			ipv6: "cellular",
		},
		{
			name: "IPv4 and IPv6 metrics are independent",
			dump: `{"interface":[
{"interface":"wan","ipv4-address":[{"address":"192.0.2.2"}],"ipv6-address":[{"address":"2001:db8:1::2"}],"route":[{"target":"0.0.0.0","metric":100},{"target":"::","metric":10}]},
{"interface":"cellular","ipv4-address":[{"address":"198.51.100.2"}],"ipv6-address":[{"address":"2001:db8:2::2"}],"route":[{"target":"0.0.0.0","metric":10},{"target":"::","metric":100}]}
]}`,
			ipv4: "cellular",
			ipv6: "wan",
		},
		{
			name: "IPv6-only fallback chooses lower metric",
			dump: `{"interface":[
{"interface":"wan6","metric":100,"ipv6-address":[{"address":"2001:db8:1::2"}],"route":[{"target":"::"}]},
{"interface":"cellular6","metric":10,"ipv6-address":[{"address":"2001:db8:2::2"}],"route":[{"target":"::"}]}
]}`,
			ipv6: "cellular6",
		},
		{
			name: "IPv4 preferred over earlier IPv6 interface",
			dump: `{"interface":[
{"interface":"wan6","metric":1,"ipv6-address":[{"address":"2001:db8:1::2"}],"route":[{"target":"::"}]},
{"interface":"cellular","metric":10,"ipv4-address":[{"address":"198.51.100.2"}],"route":[{"target":"0.0.0.0"}]}
]}`,
			ipv4: "cellular",
			ipv6: "wan6",
		},
		{
			name: "IPv6 prefix local address remains supported",
			dump: `{"interface":[
{"interface":"wan6","ipv6-prefix-assignment":[{"local-address":{"address":"2001:db8:1::1","mask":64}}],"route":[{"target":"::"}]}
]}`,
			ipv6: "wan6",
		},
		{
			name: "exclude non-main interface tables",
			dump: `{"interface":[
{"interface":"policy","metric":1,"ip4table":100,"ip6table":100,"ipv4-address":[{"address":"192.0.2.2"}],"ipv6-address":[{"address":"2001:db8:1::2"}],"route":[{"target":"0.0.0.0"},{"target":"::"}]},
{"interface":"wan","metric":100,"ip4table":254,"ip6table":254,"ipv4-address":[{"address":"198.51.100.2"}],"ipv6-address":[{"address":"2001:db8:2::2"}],"route":[{"target":"0.0.0.0"},{"target":"::"}]}
]}`,
			ipv4: "wan",
			ipv6: "wan",
		},
		{
			name: "ignore non-default and addressless routes",
			dump: `{"interface":[
{"interface":"lan","ipv4-address":[{"address":"192.0.2.2"}],"route":[{"target":"0.0.0.0","mask":1}]},
{"interface":"addressless","route":[{"target":"0.0.0.0"},{"target":"::"}]},
{"interface":"cellular","metric":10,"ipv4-address":[{"address":"198.51.100.2"}],"route":[{"target":"0.0.0.0"}]}
]}`,
			ipv4: "cellular",
		},
		{
			name: "removed cellular route falls back to wired",
			dump: `{"interface":[
{"interface":"cellular","metric":10,"ipv4-address":[{"address":"198.51.100.2"}],"route":[],"inactive":{"route":[{"target":"0.0.0.0"}]}},
{"interface":"wan","metric":100,"ipv4-address":[{"address":"192.0.2.2"}],"route":[{"target":"0.0.0.0"}]}
]}`,
			ipv4: "wan",
		},
		{
			name: "no default route keeps legacy fallback",
			dump: `{"interface":[]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOutboundInterfaceDump(t, tt.dump)
			got, err := outboundInterfaces()
			if err != nil {
				t.Fatal(err)
			}
			if got.ipv4.interfaceName != tt.ipv4 || got.ipv6.interfaceName != tt.ipv6 {
				t.Fatalf("expected IPv4=%q IPv6=%q, got IPv4=%q IPv6=%q", tt.ipv4, tt.ipv6, got.ipv4.interfaceName, got.ipv6.interfaceName)
			}
			if tt.gateway != "" && got.ipv4.gateway != tt.gateway {
				t.Fatalf("expected gateway %q, got %q", tt.gateway, got.ipv4.gateway)
			}
			if tt.ipv4 == "wwan_5g_0" {
				want := &DefaultInterface{interfaceName: "wwan_5g_0", deviceName: "usb0", l3Device: "usb0", proto: "dhcp", upTime: 90, ip: "198.51.100.2", mask: 24, dns: []string{"198.51.100.1"}, gateway: "198.51.100.1"}
				if !reflect.DeepEqual(got.ipv4, want) {
					t.Fatalf("selected WAN metadata: got %#v, want %#v", got.ipv4, want)
				}
			}
			selected, err := outboundInterface()
			if err != nil {
				t.Fatal(err)
			}
			want := got.ipv4
			if tt.ipv4 == "" {
				want = got.ipv6
			}
			if tt.ipv4 == "" && tt.ipv6 == "" {
				want = &DefaultInterface{interfaceName: "wan", deviceName: "eth0"}
			}
			if !reflect.DeepEqual(selected, want) {
				t.Fatalf("traffic WAN %#v differs from status WAN %#v", selected, want)
			}
		})
	}
}

func TestOutboundInterfaceDumpErrors(t *testing.T) {
	for _, failure := range []string{"invalid JSON", "ubus failure"} {
		t.Run(failure, func(t *testing.T) {
			mockOutboundInterfaceDump(t, "invalid JSON")
			if failure == "ubus failure" {
				if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "ubus"), []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := outboundInterfaces(); err == nil {
				t.Fatal("expected status dump error")
			}
			if _, err := outboundInterface(); err == nil {
				t.Fatal("expected traffic dump error")
			}
		})
	}
}

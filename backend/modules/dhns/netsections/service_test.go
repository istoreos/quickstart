package netsections

import (
	"errors"
	"net"
	"reflect"
	"testing"
)

type fakeReader struct {
	sections []string
	values   map[string]map[string]string
}

func (reader fakeReader) Sections(config string, sectionType string) ([]string, bool) {
	if config != "network" || sectionType != "interface" {
		return nil, false
	}
	return reader.sections, true
}

func (reader fakeReader) Last(config string, section string, option string) (string, bool) {
	if config != "network" {
		return "", false
	}
	sectionValues, ok := reader.values[section]
	if !ok {
		return "", false
	}
	value, ok := sectionValues[option]
	return value, ok
}

func mustCIDR(t *testing.T, cidr string) net.IPNet {
	t.Helper()
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatal(err)
	}
	ipnet.IP = ip
	return *ipnet
}

func ipNetFromMask(ip string, mask string) net.IPNet {
	return net.IPNet{
		IP:   net.ParseIP(ip),
		Mask: net.IPMask(net.ParseIP(mask)),
	}
}

func TestCollectParsesStaticAndDHCPNetworkSections(t *testing.T) {
	reader := fakeReader{
		sections: []string{"loopback", "lan", "wan", "planb"},
		values: map[string]map[string]string{
			"loopback": {"device": "lo", "proto": "static", "ipaddr": "127.0.0.1", "netmask": "255.0.0.0"},
			"lan":      {"device": "br-lan", "proto": "static", "ipaddr": "192.168.8.1/24"},
			"wan":      {"device": "eth0", "proto": "dhcp", "gateway": "192.168.1.1"},
			"planb":    {"device": "br-lan", "proto": "static", "ipaddr": "192.168.9.2", "netmask": "255.255.255.0"},
		},
	}
	resolver := func(device string) (*net.IPNet, error) {
		if device != "eth0" {
			t.Fatalf("unexpected DHCP device lookup: %s", device)
		}
		ipnet := mustCIDR(t, "10.0.0.5/24")
		return &ipnet, nil
	}

	got := Collect(reader, resolver)

	want := map[string]*Section{
		"lan": {
			Name:   "lan",
			Device: "br-lan",
			Proto:  "static",
			IPNet:  mustCIDR(t, "192.168.8.1/24"),
		},
		"wan": {
			Name:    "wan",
			Device:  "eth0",
			Proto:   "dhcp",
			Gateway: "192.168.1.1",
			IPNet:   mustCIDR(t, "10.0.0.5/24"),
		},
		"planb": {
			Name:   "planb",
			Device: "br-lan",
			Proto:  "static",
			IPNet:  ipNetFromMask("192.168.9.2", "255.255.255.0"),
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Collect() = %#v, want %#v", got, want)
	}
}

func TestCollectSkipsIncompleteOrUnsupportedSections(t *testing.T) {
	reader := fakeReader{
		sections: []string{"no_device", "no_proto", "dhcp_no_ip", "static_no_ip", "static_no_mask", "pppoe"},
		values: map[string]map[string]string{
			"no_device":      {"proto": "static", "ipaddr": "192.168.1.1", "netmask": "255.255.255.0"},
			"no_proto":       {"device": "br-lan", "ipaddr": "192.168.1.1", "netmask": "255.255.255.0"},
			"dhcp_no_ip":     {"device": "eth0", "proto": "dhcp"},
			"static_no_ip":   {"device": "br-lan", "proto": "static", "netmask": "255.255.255.0"},
			"static_no_mask": {"device": "br-lan", "proto": "static", "ipaddr": "192.168.1.1"},
			"pppoe":          {"device": "eth1", "proto": "pppoe"},
		},
	}
	resolver := func(device string) (*net.IPNet, error) {
		return nil, errors.New("missing address")
	}

	got := Collect(reader, resolver)

	if len(got) != 0 {
		t.Fatalf("Collect() = %#v, want empty", got)
	}
}

func TestCollectReturnsEmptyWhenInterfacesAreMissing(t *testing.T) {
	reader := fakeReader{}

	got := Collect(reader, nil)

	if len(got) != 0 {
		t.Fatalf("Collect() = %#v, want empty", got)
	}
}

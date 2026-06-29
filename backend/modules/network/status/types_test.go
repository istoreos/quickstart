package status

import "testing"

func TestResolveDNSConfig(t *testing.T) {
	t.Parallel()

	staticCfg := ResolveDNSConfig("static", true, "1", []string{"1.1.1.1", "8.8.8.8"}, []string{"9.9.9.9"})
	if staticCfg.Proto != "manual" {
		t.Fatalf("expected static proto to force manual DNS, got %q", staticCfg.Proto)
	}
	if len(staticCfg.DNSList) != 1 || staticCfg.DNSList[0] != "9.9.9.9" {
		t.Fatalf("expected manual DNS list for static proto, got %#v", staticCfg.DNSList)
	}

	peerCfg := ResolveDNSConfig("dhcp", true, "0", []string{"1.1.1.1"}, []string{"4.4.4.4"})
	if peerCfg.Proto != "manual" {
		t.Fatalf("expected peerdns=0 to force manual DNS, got %q", peerCfg.Proto)
	}
	if len(peerCfg.DNSList) != 1 || peerCfg.DNSList[0] != "4.4.4.4" {
		t.Fatalf("expected manual DNS list for peerdns=0, got %#v", peerCfg.DNSList)
	}

	autoCfg := ResolveDNSConfig("dhcp", false, "", []string{"8.8.8.8", "1.1.1.1"}, []string{"4.4.4.4"})
	if autoCfg.Proto != "auto" {
		t.Fatalf("expected auto DNS, got %q", autoCfg.Proto)
	}
	if len(autoCfg.DNSList) != 2 || autoCfg.DNSList[0] != "8.8.8.8" || autoCfg.DNSList[1] != "1.1.1.1" {
		t.Fatalf("expected outbound DNS list preserved, got %#v", autoCfg.DNSList)
	}
}

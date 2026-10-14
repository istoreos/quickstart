package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRestrictionFixture(t *testing.T, dir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte("config rule 'unrelated'\n\toption name 'keep-me'\n\toption src 'lan'\n\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "eqos"), []byte("config eqos 'main'\n\toption enabled '1'\n\nconfig device 'existing'\n\toption ip '192.168.1.10'\n\toption upload '10'\n\toption download '50'\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestAccessPolicyDoesNotDependOnOrModifyEqos(t *testing.T) {
	dir := t.TempDir()
	writeRestrictionFixture(t, dir)
	eqosBefore, _ := os.ReadFile(filepath.Join(dir, "eqos"))
	mac := "AA:BB:CC:DD:EE:25"
	if err := writeDeviceAccessPolicyAt(dir, mac, false); err != nil {
		t.Fatal(err)
	}
	include, err := os.ReadFile(filepath.Join(dir, "quickstart-access.nft"))
	if err != nil || string(include) != "ether saddr aa:bb:cc:dd:ee:25 jump reject_to_wan comment \"QuickStart pause AA:BB:CC:DD:EE:25\"\n" {
		t.Fatalf("immediate access include = %q, %v", include, err)
	}
	firewall, err := os.ReadFile(filepath.Join(dir, "firewall"))
	if err != nil || !strings.Contains(string(firewall), "option position 'chain-prepend'") || !strings.Contains(string(firewall), "option chain 'forward'") {
		t.Fatalf("firewall include registration = %q, %v", firewall, err)
	}
	allowed, err := readDeviceAccessPolicyAt(dir, mac)
	if err != nil || allowed {
		t.Fatalf("blocked policy = %v, %v", allowed, err)
	}
	eqosAfter, _ := os.ReadFile(filepath.Join(dir, "eqos"))
	if string(eqosAfter) != string(eqosBefore) {
		t.Fatal("access policy modified eqos")
	}
	if err := writeDeviceAccessPolicyAt(dir, mac, true); err != nil {
		t.Fatal(err)
	}
	include, err = os.ReadFile(filepath.Join(dir, "quickstart-access.nft"))
	if err != nil || len(include) != 0 {
		t.Fatalf("cleared immediate access include = %q, %v", include, err)
	}
	allowed, err = readDeviceAccessPolicyAt(dir, mac)
	if err != nil || !allowed {
		t.Fatalf("restored policy = %v, %v", allowed, err)
	}
}

func TestSpeedPolicyDoesNotModifyFirewall(t *testing.T) {
	dir := t.TempDir()
	writeRestrictionFixture(t, dir)
	firewallBefore, _ := os.ReadFile(filepath.Join(dir, "firewall"))
	if err := writeDeviceSpeedPolicyAt(dir, "192.168.1.10", "AA:BB:CC:DD:EE:25", "电脑", true, 20, 100); err != nil {
		t.Fatal(err)
	}
	speed, err := readDeviceSpeedPolicyAt(dir, "192.168.1.10")
	if err != nil || speed.Section == "" || speed.Upload != 20 || speed.Download != 100 {
		t.Fatalf("speed = %#v, %v", speed, err)
	}
	firewallAfter, _ := os.ReadFile(filepath.Join(dir, "firewall"))
	if string(firewallAfter) != string(firewallBefore) {
		t.Fatal("speed policy modified firewall")
	}
}

func TestFirewallCapabilityIsIndependentFromEqosPresence(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte("config defaults 'main'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	capability := firewallCapabilityAt(dir)
	if capability.State != "available" {
		t.Fatalf("capability = %#v", capability)
	}
	if err := os.Remove(filepath.Join(dir, "firewall")); err != nil {
		t.Fatal(err)
	}
	capability = firewallCapabilityAt(dir)
	if capability.State != "unsupported" {
		t.Fatalf("missing capability = %#v", capability)
	}
}

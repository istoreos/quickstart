package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func TestMultiLANSharedZoneAccessPolicies(t *testing.T) {
	dir := t.TempDir()
	configuration := "config zone 'shared'\n option name 'shared'\n list network 'lan'\n list network 'office'\nconfig zone 'uplink'\n option name 'wan'\n list network 'wan'\n"
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte(configuration), 0600); err != nil {
		t.Fatal(err)
	}
	lan := &models.LANScope{NetworkID: "lan", Device: "br-lan", State: "up", FirewallZones: []string{"shared"}}
	office := &models.LANScope{NetworkID: "office", Device: "br-office", State: "up", FirewallZones: []string{"shared"}}
	mac := "02:00:00:00:00:55"
	for _, scope := range []*models.LANScope{lan, office} {
		if err := writeDeviceAccessPolicyAt(withLANScope(context.Background(), scope), dir, mac, false); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(immediateAccessIncludePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	for _, device := range []string{"br-lan", "br-office"} {
		if !strings.Contains(string(raw), "iifname \""+device+"\" ether saddr "+strings.ToLower(mac)) {
			t.Fatalf("missing ingress binding: %s", raw)
		}
	}
	if err := writeDeviceAccessPolicyAt(withLANScope(context.Background(), office), dir, mac, true); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		scope   *models.LANScope
		allowed bool
	}{{lan, false}, {office, true}} {
		allowed, err := readDeviceAccessPolicyAt(withLANScope(context.Background(), test.scope), dir, mac)
		if err != nil || allowed != test.allowed {
			t.Fatalf("%s allowed=%v err=%v", test.scope.NetworkID, allowed, err)
		}
	}
	raw, err = os.ReadFile(immediateAccessIncludePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "br-office") || !strings.Contains(string(raw), "br-lan") {
		t.Fatalf("cross-LAN access deletion: %s", raw)
	}
}

func TestMultiLANLegacyAccessNarrowedToOriginalNetwork(t *testing.T) {
	dir := t.TempDir()
	configuration := "config zone 'default'\n option name 'lan'\n list network 'lan'\n list network 'office'\nconfig zone 'external'\n option name 'wan'\n list network 'wan'\nconfig rule 'legacy'\n option name 'BL_old'\n option src 'lan'\n option dest 'wan'\n option src_mac '02:00:00:00:00:55'\n option target 'REJECT'\n"
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte(configuration), 0600); err != nil {
		t.Fatal(err)
	}
	lan := &models.LANScope{NetworkID: "lan", Device: "br-lan", FirewallZones: []string{"lan"}, State: "up"}
	office := &models.LANScope{NetworkID: "office", Device: "br-office", FirewallZones: []string{"lan"}, State: "up"}
	ctx := withLANScope(context.Background(), office)
	if err := writeDeviceAccessPolicyAt(ctx, dir, "02:00:00:00:00:55", false); err == nil {
		t.Fatal("unproven legacy network ownership accepted")
	}
	resolver := &LANScopeModule{reader: fakeLANScopeReader{scopes: []*models.LANScope{lan, office}}, now: time.Now}
	ctx = context.WithValue(ctx, lanScopeResolverKey{}, resolver)
	if err := writeDeviceAccessPolicyAt(ctx, dir, "02:00:00:00:00:55", false); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(immediateAccessIncludePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "iifname") != 2 {
		t.Fatalf("legacy rule remained global: %s", raw)
	}
}

func TestMultiLANUnscopedLegacyReadDoesNotConsumeAdditionalRules(t *testing.T) {
	dir := t.TempDir()
	raw := "config rule 'office'\n option name 'BL_office'\n option src_mac '02:00:00:00:00:55'\n option target 'REJECT'\n option quickstart_network 'office'\n option device 'br-office'\n"
	if err := os.WriteFile(filepath.Join(dir, "firewall"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	access, err := readDeviceAccessPolicyAt(context.Background(), dir, "02:00:00:00:00:55")
	if err != nil || !access {
		t.Fatalf("additional LAN rule leaked into old default call: access=%v err=%v", access, err)
	}
}

package service

import (
	"context"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

const multiLANDHCPFixture = `config dnsmasq 'main'
 option extraconftext '# administrator settings'
 option leasefile '/tmp/task-owned.leases'
config dhcp 'pool_a'
 option interface 'office'
 option instance 'main'
 option start '100'
 option limit '21'
 list dhcp_option '42,192.0.2.9'
config dhcp 'pool_b'
 option interface 'guest'
 option instance 'main'
 option start '100'
 option limit '21'
config host 'unrelated'
 option mac '02:00:00:00:00:66'
 option ip '203.0.113.66'
`

func multiLANDHCPScopes() []*models.LANScope {
	return []*models.LANScope{
		{NetworkID: "office", Device: "lana", State: "up", IPv4Prefixes: []string{"192.0.2.0/24"}, DHCPBindings: []models.LANDHCPBinding{{Section: "pool_a", Instance: "main"}}},
		{NetworkID: "guest", Device: "lanb", State: "up", IPv4Prefixes: []string{"198.51.100.0/24"}, DHCPBindings: []models.LANDHCPBinding{{Section: "pool_b", Instance: "main"}}},
	}
}

func TestMultiLANDHCPScopedMutations(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dhcp"), []byte(multiLANDHCPFixture), 0600); err != nil {
		t.Fatal(err)
	}
	scopes := multiLANDHCPScopes()
	for i, scope := range scopes {
		ip, gateway := "192.0.2.55", "192.0.2.2"
		if i == 1 {
			ip, gateway = "198.51.100.55", "198.51.100.2"
		}
		route := gatewayPolicyExecutionPlan{State: gatewayPolicySnapshot{Scope: scope}, Target: &gatewayTargetDescriptor{TagName: scopedDHCPSectionName(scope, "route"), TagTitle: "path", Options: gatewayAndDNSOptions(gateway), Materialize: true}}
		input := StaticAssignmentWriteInput{Scope: scope, Action: "add", AssignedMAC: "02:00:00:00:00:55", AssignedIP: ip, BindIP: true}
		if err := mutateDeviceNetworkPolicyConfigAt(dir, input, route); err != nil {
			t.Fatal(err)
		}
	}
	tree := uci.NewTree(dir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	for i, scope := range scopes {
		ctx := withLANScope(context.Background(), scope)
		binding, err := scopedDHCPBinding(ctx, tree)
		if err != nil || binding.Section != scope.DHCPBindings[0].Section {
			t.Fatalf("binding=%+v error=%v", binding, err)
		}
		count := 0
		hosts, _ := tree.GetSections("dhcp", "host")
		for _, host := range hosts {
			if dhcpHostInScope(tree, host, scope) {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("%s: %d host records", scope.NetworkID, count)
		}
		if i == 0 {
			plan := lanDHCPSettingsExecutionPlan{State: lanDHCPSettingsSnapshot{DHCPSection: binding.Section, Prefix: netip.MustParsePrefix("192.0.2.0/24"), RouterAddress: "192.0.2.1"}, Desired: &models.LanDHCPSettings{Enabled: true, PoolStart: "192.0.2.100", PoolEnd: "192.0.2.120", LeaseTime: "10m"}, DefaultGateway: "192.0.2.1"}
			if err := mutateLanDHCPSettingsAt(dir, plan); err != nil {
				t.Fatal(err)
			}
		}
	}
	exportMultiLANDHCPFixture(t, dir, "static")
	// Removing reservations must keep independent routes without inventing a
	// hostname or silently reserving each client's observed address.
	for _, scope := range scopes {
		route := gatewayPolicyExecutionPlan{State: gatewayPolicySnapshot{Scope: scope}, Target: &gatewayTargetDescriptor{TagName: scopedDHCPSectionName(scope, "route")}}
		if err := mutateDeviceNetworkPolicyConfigAt(dir, StaticAssignmentWriteInput{Scope: scope, Action: "add", AssignedMAC: "02:00:00:00:00:55"}, route); err != nil {
			t.Fatal(err)
		}
	}
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	text, _ := tree.GetLast("dhcp", "main", "extraconftext")
	if !strings.Contains(text, "# administrator settings") || strings.Count(text, "dhcp-host=tag:") != 2 {
		t.Fatalf("compiled hosts: %s", text)
	}
	options, _ := tree.Get("dhcp", "pool_a", "dhcp_option")
	if !strings.Contains(strings.Join(options, ";"), "42,192.0.2.9") {
		t.Fatalf("unrelated DHCP option lost: %v", options)
	}
	if value, _ := tree.GetLast("dhcp", "pool_b", "start"); value != "100" {
		t.Fatal("other pool changed")
	}
	if value, _ := tree.GetLast("dhcp", "unrelated", "ip"); value != "203.0.113.66" {
		t.Fatal("unrelated host changed")
	}
	exportMultiLANDHCPFixture(t, dir, "routes")
	disabled := lanDHCPSettingsExecutionPlan{State: lanDHCPSettingsSnapshot{DHCPSection: "pool_a", Prefix: netip.MustParsePrefix("192.0.2.0/24"), RouterAddress: "192.0.2.1"}, Desired: &models.LanDHCPSettings{Enabled: false, PoolStart: "192.0.2.100", PoolEnd: "192.0.2.120", LeaseTime: "10m"}, DefaultGateway: "192.0.2.1"}
	if err := mutateLanDHCPSettingsAt(dir, disabled); err != nil {
		t.Fatal(err)
	}
	exportMultiLANDHCPFixture(t, dir, "disabled")
	disabled.Desired.Enabled = true
	if err := mutateLanDHCPSettingsAt(dir, disabled); err != nil {
		t.Fatal(err)
	}

	for _, scope := range scopes {
		if err := mutateDeviceNetworkPolicyConfigAt(dir, StaticAssignmentWriteInput{Scope: scope, AssignedMAC: "02:00:00:00:00:55"}, gatewayPolicyExecutionPlan{Target: &gatewayTargetDescriptor{}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	text, _ = tree.GetLast("dhcp", "main", "extraconftext")
	if strings.Contains(text, "dhcp-host=") || !strings.Contains(text, "# administrator settings") {
		t.Fatalf("stale route or lost administrator settings: %s", text)
	}
	exportMultiLANDHCPFixture(t, dir, "deleted")
}

func exportMultiLANDHCPFixture(t *testing.T, dir, phase string) {
	t.Helper()
	if target := os.Getenv("QUICKSTART_MULTI_LAN_DHCP_FIXTURE_DIR"); target != "" {
		data, err := os.ReadFile(filepath.Join(dir, "dhcp"))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(target, phase)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(path, "dhcp"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMultiLANDHCPAmbiguousBindingAndReservation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dhcp"), []byte(multiLANDHCPFixture+"\nconfig dhcp 'duplicate'\n option interface 'office'\n option instance 'main'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(dir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	if _, err := scopedDHCPBinding(withLANScope(context.Background(), multiLANDHCPScopes()[0]), tree); err == nil {
		t.Fatal("ambiguous binding accepted")
	}
	state := gatewayPolicySnapshot{Scope: multiLANDHCPScopes()[0], Prefixes: []netip.Prefix{netip.MustParsePrefix("192.0.2.0/24")}, LAN: LanStatusSnapshot{LanAddr: "192.0.2.1"}}
	for _, address := range []string{"192.0.2.0", "192.0.2.1", "192.0.2.255", "198.51.100.55", "invalid", "::1"} {
		if err := validateScopedReservation(StaticAssignmentWriteInput{AssignedIP: address, BindIP: true}, state); err == nil {
			t.Fatalf("accepted %s", address)
		}
	}
	if err := validateScopedReservation(StaticAssignmentWriteInput{AssignedIP: "192.0.2.55", BindIP: true}, state); err != nil {
		t.Fatal(err)
	}
}

func TestMultiLANGatewayReferencesStayInNetwork(t *testing.T) {
	store := newJSONDeviceGroupStore(filepath.Join(t.TempDir(), "groups.json"))
	for _, network := range []string{"lan", "office"} {
		ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: network})
		_, err := store.Mutate(ctx, &models.DeviceGroupMutationRequest{Action: "set_global_policy", GlobalPolicy: &models.GroupPolicy{TargetID: "route"}}, "")
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx := withLANScope(context.Background(), &models.LANScope{NetworkID: "office"})
	if err := store.forContext(ctx).ReplaceGatewayTarget("route", "default", ""); err != nil {
		t.Fatal(err)
	}
	main, err := store.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	office, err := store.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if main.GlobalPolicy.TargetID != "route" || office.GlobalPolicy.TargetID != "default" {
		t.Fatal("gateway reference crossed networks")
	}
}

func TestMultiLANDHCPAnonymousInstanceIdentity(t *testing.T) {
	dir := t.TempDir()
	raw := "config dnsmasq\n option leasefile '/tmp/first.leases'\nconfig dnsmasq 'named'\nconfig dhcp 'custom'\n option interface 'office'\n option instance 'cfg01411c'\n"
	if err := os.WriteFile(filepath.Join(dir, "dhcp"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(dir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	aliases := dnsmasqInstanceAliases("dhcp.cfg01411c=dnsmasq\ndhcp.cfg01411c.leasefile='/tmp/first.leases'\ndhcp.named=dnsmasq\n")
	bindings := lanDHCPBindings(tree, "office", aliases)
	if len(bindings) != 1 || bindings[0].Instance != "@dnsmasq[0]" || bindings[0].RuntimeInstance != "cfg01411c" || bindings[0].LeaseFile != "/tmp/first.leases" {
		t.Fatalf("bindings=%+v", bindings)
	}
	scope := &models.LANScope{NetworkID: "office", Device: "lana", DHCPBindings: bindings}
	input := StaticAssignmentWriteInput{Scope: scope, AssignedMAC: "02:00:00:00:00:55"}
	if err := mutateDeviceNetworkPolicyConfigAt(dir, input, gatewayPolicyExecutionPlan{Target: &gatewayTargetDescriptor{TagName: "path"}}); err != nil {
		t.Fatal(err)
	}
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	section := scopedDHCPSectionName(scope, "quickstart_020000000055")
	if value, _ := tree.GetLast("dhcp", section, "instance"); value != "cfg01411c" {
		t.Fatalf("dnsmasq instance filter=%s", value)
	}
	if text, _ := tree.GetLast("dhcp", "@dnsmasq[0]", "extraconftext"); !strings.Contains(text, "dhcp-host=tag:lana") {
		t.Fatal("host compiled into wrong anonymous instance")
	}
	if text, _ := tree.GetLast("dhcp", "named", "extraconftext"); text != "" {
		t.Fatal("unrelated instance changed")
	}
}

func TestMultiLANDHCPReadOnlyBindings(t *testing.T) {
	if os.Getenv("QUICKSTART_MULTI_LAN_TEST_DEVICE") != "1" {
		t.Skip("explicit test-router environment required")
	}
	networks, err := NewLANScopeModule().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, scope := range networks {
		if len(scope.DHCPBindings) != 1 || len(scope.IPv4Prefixes) == 0 {
			continue
		}
		ctx := withLANScope(context.Background(), scope)
		state, err := (&defaultLanDHCPSettingsStore{routerContext: NewDefaultRouterContextModule()}).Read(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if state.DHCPSection != scope.DHCPBindings[0].Section || state.Authority != "local" || !state.Settings.Enabled {
			t.Fatalf("%s binding/authority mismatch: %+v", scope.NetworkID, state)
		}
		t.Logf("network=%s section=%s selector=%s runtime=%s authority=%s", scope.NetworkID, state.DHCPSection, scope.DHCPBindings[0].Instance, scope.DHCPBindings[0].RuntimeInstance, state.Authority)
		checked++
	}
	if checked == 0 {
		t.Fatal("no configured DHCP service checked")
	}
}

func TestMultiLANDHCPLeaseEvidenceAndEffectsAreScoped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "leases")
	if err := os.WriteFile(path, []byte("0 02:00:00:00:00:55 198.51.100.55 guest *\n"), 0600); err != nil {
		t.Fatal(err)
	}
	store := newJSONPolicyEffectStore(filepath.Join(dir, "effects.json"))
	observer := &dnsmasqLeaseObserver{path: path}
	for i, scope := range multiLANDHCPScopes() {
		scope.DHCPBindings[0].LeaseFile = path
		ctx := withLANScope(context.Background(), scope)
		effect := &models.PolicyEffect{Desired: &models.DesiredNetworkPolicy{TargetID: scope.NetworkID}}
		if err := store.Write(ctx, "same-device", effect); err != nil {
			t.Fatal(err)
		}
		observation, err := observer.Observe(ctx, "02:00:00:00:00:55", time.Time{})
		if err != nil || observation.Observed != (i == 1) {
			t.Fatalf("%s lease observation=%+v err=%v", scope.NetworkID, observation, err)
		}
	}
	for _, scope := range multiLANDHCPScopes() {
		ctx := withLANScope(context.Background(), scope)
		effect, ok, err := newJSONPolicyEffectStore(store.path).Read(ctx, "same-device")
		if err != nil || !ok || effect.Desired.TargetID != scope.NetworkID {
			t.Fatalf("%s persisted effect=%+v err=%v", scope.NetworkID, effect, err)
		}
	}
	if _, ok, err := store.Read(context.Background(), "same-device"); err != nil || ok {
		t.Fatal("additional LAN effects leaked into legacy storage")
	}
	audit := NewNetworkAuditModule(filepath.Join(dir, "audit.json"), nil)
	for _, scope := range multiLANDHCPScopes() {
		ctx := withLANScope(context.Background(), scope)
		audit.forContext(ctx).Record("same-device", "network", "policy_changed", "success", "")
	}
	for _, scope := range multiLANDHCPScopes() {
		child := audit.forContext(withLANScope(context.Background(), scope))
		if len(child.document.Events) != 1 || child.document.Events[0].NetworkID != scope.NetworkID {
			t.Fatal("audit network identity lost")
		}
	}
	if _, err := os.Stat(audit.path); !os.IsNotExist(err) {
		t.Fatal("additional LAN event wrote the default audit file")
	}
}

func TestMultiLANDHCPRollbackRestoresSharedConfiguration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dhcp")
	original := []byte(multiLANDHCPFixture)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	beforeSnapshot, beforeMutate, beforeReload, beforeRestore := lanDHCPSettingsTakeSnapshot, lanDHCPSettingsMutate, lanDHCPSettingsReload, lanDHCPSettingsRestore
	t.Cleanup(func() {
		lanDHCPSettingsTakeSnapshot, lanDHCPSettingsMutate, lanDHCPSettingsReload, lanDHCPSettingsRestore = beforeSnapshot, beforeMutate, beforeReload, beforeRestore
	})
	lanDHCPSettingsTakeSnapshot = func() (dhcpConfigSnapshot, error) {
		raw, err := os.ReadFile(path)
		return dhcpConfigSnapshot{Data: raw, Mode: 0600}, err
	}
	lanDHCPSettingsMutate = func(plan lanDHCPSettingsExecutionPlan) error { return mutateLanDHCPSettingsAt(dir, plan) }
	lanDHCPSettingsReload = func(context.Context) error { return fmt.Errorf("injected selected-instance verification failure") }
	lanDHCPSettingsRestore = func(_ context.Context, snapshot dhcpConfigSnapshot) error {
		return os.WriteFile(path, snapshot.Data, snapshot.Mode)
	}
	plan := lanDHCPSettingsExecutionPlan{State: lanDHCPSettingsSnapshot{DHCPSection: "pool_a", Prefix: netip.MustParsePrefix("192.0.2.0/24"), RouterAddress: "192.0.2.1"}, Desired: &models.LanDHCPSettings{Enabled: true, PoolStart: "192.0.2.100", PoolEnd: "192.0.2.120", LeaseTime: "10m"}, DefaultGateway: "192.0.2.2"}
	if err := (&defaultLanDHCPSettingsStore{}).Apply(withLANScope(context.Background(), multiLANDHCPScopes()[0]), plan); err == nil {
		t.Fatal("verification failure was ignored")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(original) {
		t.Fatal("shared configuration was not restored")
	}
}

func TestMultiLANFloatingGatewayStaysOnDefaultLAN(t *testing.T) {
	ctx := withLANScope(context.Background(), multiLANDHCPScopes()[0])
	if _, err := NewFloatingGatewayModule(&fakeFloatingGatewayStore{}, nil).Get(ctx); err == nil {
		t.Fatal("additional LAN read default floating gateway")
	}
	if err := (&systemFloatingGatewayStore{}).Apply(ctx, &models.FloatingGatewayConfig{}); err == nil {
		t.Fatal("additional LAN could write default floating gateway")
	}
}

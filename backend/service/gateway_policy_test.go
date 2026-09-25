package service

import (
	"context"
	"encoding/json"
	"errors"
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

type fakeGatewayPolicyStore struct {
	state   gatewayPolicySnapshot
	applies int
	err     error
}

func (store *fakeGatewayPolicyStore) ReadState(context.Context) (gatewayPolicySnapshot, error) {
	return store.state, store.err
}

func (store *fakeGatewayPolicyStore) Apply(_ context.Context, plan gatewayPolicyExecutionPlan) error {
	if store.err != nil {
		return store.err
	}
	store.applies++
	if plan.Target.Materialize {
		store.state.DHCP.Tags = append(store.state.DHCP.Tags, DhcpTagRecord{
			TagName: plan.Target.TagName, TagTitle: plan.Target.TagTitle, Gateway: plan.Target.Public.Gateway,
			DhcpOption: append([]string(nil), plan.Target.Options...), AutoCreated: true,
		})
	}
	for index := range store.state.Hosts {
		if normalizeInventoryMAC(store.state.Hosts[index].MAC) == normalizeInventoryMAC(plan.MAC) {
			store.state.Hosts[index].TagName = plan.Target.TagName
			return nil
		}
	}
	store.state.Hosts = append(store.state.Hosts, gatewayPolicyHost{MAC: plan.MAC, TagName: plan.Target.TagName})
	return nil
}

func gatewayPolicyTestState() gatewayPolicySnapshot {
	prefix := netip.MustParsePrefix("192.168.100.0/24")
	return gatewayPolicySnapshot{
		LAN: LanStatusSnapshot{LanAddr: "192.168.100.1", Nexthop: "192.168.100.254"},
		DHCP: &LanDhcpState{Tags: []DhcpTagRecord{
			{TagName: "legacy_route", TagTitle: "office", Gateway: "192.168.100.2", DhcpOption: []string{"3,192.168.100.2", "6,192.168.100.2"}},
			{TagName: "legacy_unknown", TagTitle: "legacy", Gateway: "192.168.100.3", DhcpOption: []string{"3,192.168.100.3", "42,192.168.100.3"}},
		}},
		Hosts:    []gatewayPolicyHost{{SectionName: "host1", MAC: "AA:BB:CC:DD:EE:01", TagName: "legacy_route"}},
		Prefixes: []netip.Prefix{prefix}, Version: "version-one",
	}
}

func gatewayPolicyTestModule(t *testing.T, store *fakeGatewayPolicyStore) *GatewayPolicyModule {
	t.Helper()
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	inventory := newTestDeviceInventory(t, &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{
		ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.100.20", Online: true}},
	}}}, &now, 2048)
	return NewGatewayPolicyModule(inventory, store)
}

func TestGatewayPolicyListsOpaqueTargetsAndPreservesUnsupportedOptions(t *testing.T) {
	store := &fakeGatewayPolicyStore{state: gatewayPolicyTestState()}
	response, err := gatewayPolicyTestModule(t, store).ListTargets(context.Background())
	if err != nil || response.Result == nil || len(response.Result.Targets) != 5 {
		t.Fatalf("target list = %#v, %v", response, err)
	}
	encoded, _ := json.Marshal(response)
	if strings.Contains(string(encoded), "legacy_route") || strings.Contains(string(encoded), "legacy_unknown") {
		t.Fatalf("internal DHCP tag leaked in API: %s", encoded)
	}
	var unsupported *models.GatewayTarget
	for _, target := range response.Result.Targets {
		if target.Gateway == "192.168.100.3" {
			unsupported = target
		}
	}
	if unsupported == nil || unsupported.Supported || unsupported.Reasons[0] != "unsupported_dhcp_option" {
		t.Fatalf("unsupported target = %#v", unsupported)
	}
}

func TestGatewayPolicyPlanIsReadOnlyAndApplyIsIdempotent(t *testing.T) {
	store := &fakeGatewayPolicyStore{state: gatewayPolicyTestState()}
	module := gatewayPolicyTestModule(t, store)
	request := &models.GatewayAssignmentRequest{Action: "assign", DeviceID: "mac:aa:bb:cc:dd:ee:01", TargetID: "upstream"}
	planned, err := module.PlanAssignment(context.Background(), request)
	if err != nil || !planned.Result.CanApply || len(planned.Result.Changes) != 1 || store.applies != 0 {
		t.Fatalf("plan = %#v, applies=%d, err=%v", planned, store.applies, err)
	}
	request.ExpectedVersion = planned.Result.Version
	applied, err := module.ApplyAssignment(context.Background(), request)
	if err != nil || !applied.Result.Changed || applied.Result.EffectState != "pending_renewal" || store.applies != 1 {
		t.Fatalf("apply = %#v, applies=%d, err=%v", applied, store.applies, err)
	}
	repeated, err := module.ApplyAssignment(context.Background(), request)
	if err != nil || repeated.Result.Changed || len(repeated.Result.Plan.Changes) != 0 || store.applies != 1 {
		t.Fatalf("repeat = %#v, applies=%d, err=%v", repeated, store.applies, err)
	}
}

func TestGatewayPolicyRejectsReferencedDeleteAndStaleVersion(t *testing.T) {
	store := &fakeGatewayPolicyStore{state: gatewayPolicyTestState()}
	module := gatewayPolicyTestModule(t, store)
	list, _ := module.ListTargets(context.Background())
	var referenced string
	for _, target := range list.Result.Targets {
		if target.ReferenceCount > 0 {
			referenced = target.ID
		}
	}
	deleted, err := module.PlanAssignment(context.Background(), &models.GatewayAssignmentRequest{Action: "delete_target", TargetID: referenced})
	if err != nil || deleted.Result.Error == nil || deleted.Result.Error.Code != "conflict" || store.applies != 0 {
		t.Fatalf("delete plan = %#v, %v", deleted, err)
	}
	stale, err := module.ApplyAssignment(context.Background(), &models.GatewayAssignmentRequest{
		Action: "assign", DeviceID: "mac:aa:bb:cc:dd:ee:01", TargetID: "self", ExpectedVersion: "stale",
	})
	if err != nil || stale.Result.Error == nil || stale.Result.Error.Code != "conflict" || store.applies != 0 {
		t.Fatalf("stale apply = %#v, %v", stale, err)
	}
}

func TestGatewayPolicyRejectsOutsideLANAndScans2048References(t *testing.T) {
	state := gatewayPolicyTestState()
	state.DHCP.Tags = append(state.DHCP.Tags, DhcpTagRecord{
		TagName: "outside", TagTitle: "outside", Gateway: "203.0.113.1", DhcpOption: []string{"3,203.0.113.1"},
	})
	for index := 0; index < 2048; index++ {
		state.Hosts = append(state.Hosts, gatewayPolicyHost{MAC: fmt.Sprintf("02:00:00:%02x:%02x:%02x", index>>16, index>>8, index), TagName: "legacy_route"})
	}
	store := &fakeGatewayPolicyStore{state: state}
	module := gatewayPolicyTestModule(t, store)
	started := time.Now()
	list, err := module.ListTargets(context.Background())
	if err != nil || time.Since(started) > 500*time.Millisecond {
		t.Fatalf("2048 reference scan took %s: %v", time.Since(started), err)
	}
	var outside string
	for _, target := range list.Result.Targets {
		if target.Gateway == "203.0.113.1" {
			outside = target.ID
		}
	}
	plan, err := module.PlanAssignment(context.Background(), &models.GatewayAssignmentRequest{
		Action: "assign", DeviceID: "mac:aa:bb:cc:dd:ee:01", TargetID: outside,
	})
	if err != nil || plan.Result.Error == nil || !strings.Contains(plan.Result.Error.Message, "outside") {
		t.Fatalf("outside plan = %#v, %v", plan, err)
	}
}

func TestMutateGatewayPolicyPreservesHostAndMaterializesTarget(t *testing.T) {
	directory := t.TempDir()
	config := "config dnsmasq\n\toption domainneeded '1'\n\nconfig host 'legacy'\n\toption mac 'AA:BB:CC:DD:EE:01'\n\toption ip '192.168.100.20'\n\toption name 'printer'\n"
	if err := os.WriteFile(filepath.Join(directory, "dhcp"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	plan := gatewayPolicyExecutionPlan{
		Public: &models.GatewayAssignmentPlan{Action: "assign", Changes: []*models.GatewayPlanChange{{Kind: "assignment"}}},
		MAC:    "AA:BB:CC:DD:EE:01",
		Target: &gatewayTargetDescriptor{TagName: "t_auto_c0a86402", TagTitle: "bypass", Options: []string{"3,192.168.100.2", "6,192.168.100.2"}, Materialize: true},
	}
	if err := mutateGatewayPolicyConfigAt(directory, plan); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(directory)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	if ip, _ := tree.GetLast("dhcp", "legacy", "ip"); ip != "192.168.100.20" {
		t.Fatalf("static IP lost: %q", ip)
	}
	if name, _ := tree.GetLast("dhcp", "legacy", "name"); name != "printer" {
		t.Fatalf("hostname lost: %q", name)
	}
	if tag, _ := tree.GetLast("dhcp", "legacy", "tag"); tag != "t_auto_c0a86402" {
		t.Fatalf("assignment tag = %q", tag)
	}
	if options, _ := tree.Get("dhcp", "t_auto_c0a86402", "dhcp_option"); len(options) != 2 {
		t.Fatalf("target options = %#v", options)
	}
}

func TestGatewayPolicyApplyRollsBackMutationFailure(t *testing.T) {
	originalSnapshot, originalMutate, originalReload, originalRestore := gatewayPolicyTakeSnapshot, gatewayPolicyMutate, gatewayPolicyReload, gatewayPolicyRestore
	t.Cleanup(func() {
		gatewayPolicyTakeSnapshot, gatewayPolicyMutate, gatewayPolicyReload, gatewayPolicyRestore = originalSnapshot, originalMutate, originalReload, originalRestore
	})
	gatewayPolicyTakeSnapshot = func() (dhcpConfigSnapshot, error) { return dhcpConfigSnapshot{Data: []byte("before"), Mode: 0600}, nil }
	gatewayPolicyMutate = func(gatewayPolicyExecutionPlan) error { return errors.New("write failed") }
	restored := false
	gatewayPolicyRestore = func(context.Context, dhcpConfigSnapshot) error { restored = true; return nil }
	store := &defaultGatewayPolicyStore{}
	err := store.Apply(context.Background(), gatewayPolicyExecutionPlan{
		Public: &models.GatewayAssignmentPlan{Action: "delete_target", Changes: []*models.GatewayPlanChange{{Kind: "delete_target"}}},
		Target: &gatewayTargetDescriptor{TagName: "custom"},
	})
	if err == nil || !restored {
		t.Fatalf("rollback = %v, %v", restored, err)
	}
}

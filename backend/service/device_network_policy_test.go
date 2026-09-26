package service

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

type fakeDeviceNetworkPolicyWriter struct {
	policyStore  *fakeDevicePolicyStore
	gatewayStore *fakeGatewayPolicyStore
	calls        int
	err          error
}

type fakePolicyLeaseObserver struct {
	observation leaseObservation
	err         error
}

func (observer *fakePolicyLeaseObserver) Observe(context.Context, string, time.Time) (leaseObservation, error) {
	return observer.observation, observer.err
}

func (writer *fakeDeviceNetworkPolicyWriter) Apply(_ context.Context, input StaticAssignmentWriteInput, route gatewayPolicyExecutionPlan) error {
	writer.calls++
	if writer.err != nil {
		return writer.err
	}
	writer.policyStore.policy.Static = &models.DeviceStaticPolicy{
		Enabled: input.BindIP || input.Hostname != "", AssignedIP: input.AssignedIP,
		BindIP: input.BindIP, Hostname: input.Hostname,
	}
	if route.Target.Materialize && route.Target.TagName != "" {
		writer.gatewayStore.state.DHCP.Tags = append(writer.gatewayStore.state.DHCP.Tags, DhcpTagRecord{
			TagName: route.Target.TagName, TagTitle: route.Target.TagTitle, Gateway: route.Target.Public.Gateway,
			DhcpOption: append([]string(nil), route.Target.Options...), AutoCreated: true,
		})
	}
	for index := range writer.gatewayStore.state.Hosts {
		if normalizeInventoryMAC(writer.gatewayStore.state.Hosts[index].MAC) == normalizeInventoryMAC(input.AssignedMAC) {
			writer.gatewayStore.state.Hosts[index].TagName = route.Target.TagName
			return nil
		}
	}
	writer.gatewayStore.state.Hosts = append(writer.gatewayStore.state.Hosts, gatewayPolicyHost{MAC: input.AssignedMAC, TagName: route.Target.TagName})
	return nil
}

func deviceNetworkPolicyTestModule(t *testing.T) (*DeviceNetworkPolicyModule, *fakeDeviceNetworkPolicyWriter, *fakeDevicePolicyStore, *fakeGatewayPolicyStore) {
	t.Helper()
	policyStore := availablePolicyStore()
	policyStore.policy.DeviceID = "mac:aa:bb:cc:dd:ee:01"
	policyStore.policy.MAC = "AA:BB:CC:DD:EE:01"
	gatewayStore := &fakeGatewayPolicyStore{state: gatewayPolicyTestState()}
	gateway := gatewayPolicyTestModule(t, gatewayStore)
	writer := &fakeDeviceNetworkPolicyWriter{policyStore: policyStore, gatewayStore: gatewayStore}
	effects := NewPolicyEffectModule(newMemoryPolicyEffectStore(), &fakePolicyLeaseObserver{observation: leaseObservation{SourceAvailable: true}}, func() time.Time {
		return time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	})
	return NewDeviceNetworkPolicyModule(gateway.inventory, newDevicePolicyModuleForTest(policyStore), gateway, writer, effects), writer, policyStore, gatewayStore
}

func TestDeviceNetworkPolicyReturnsOpaqueInternetPaths(t *testing.T) {
	module, _, _, _ := deviceNetworkPolicyTestModule(t)
	response, err := module.Get(context.Background(), "mac:aa:bb:cc:dd:ee:01")
	if err != nil || response.Result.Error != nil || response.Result.Policy.Path.TargetID == "" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	raw, _ := json.Marshal(response)
	encoded := string(raw)
	if strings.Contains(encoded, "legacy_route") || strings.Contains(encoded, "tagName") || strings.Contains(encoded, "dhcpOption") {
		t.Fatalf("internal DHCP details leaked: %s", encoded)
	}
	if response.Result.Policy.Path.Effect.Observed.State != "unverifiable" {
		t.Fatalf("effect = %#v", response.Result.Policy.Path.Effect)
	}
}

func TestDeviceNetworkPolicyRejectsApplyWhenDHCPIsDisabled(t *testing.T) {
	module, writer, _, gatewayStore := deviceNetworkPolicyTestModule(t)
	gatewayStore.state.DHCP.DhcpIgnore = true
	response, err := module.Apply(context.Background(), &models.DeviceNetworkPolicyApplyRequest{
		DeviceID: "mac:aa:bb:cc:dd:ee:01", TargetID: "default", Static: &models.DeviceAddressPolicy{},
	})
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "dhcp_authority_unavailable" || writer.calls != 0 {
		t.Fatalf("response=%#v calls=%d err=%v", response, writer.calls, err)
	}
}

func TestDeviceNetworkPolicyAtomicallyAppliesStaticAndPath(t *testing.T) {
	module, writer, policyStore, gatewayStore := deviceNetworkPolicyTestModule(t)
	listed, _ := module.Get(context.Background(), "mac:aa:bb:cc:dd:ee:01")
	request := &models.DeviceNetworkPolicyApplyRequest{
		DeviceID: "mac:aa:bb:cc:dd:ee:01", TargetID: "upstream", ExpectedVersion: listed.Result.Policy.Version,
		Static: &models.DeviceAddressPolicy{Enabled: true, AssignedIP: "192.168.100.50", BindIP: true, Hostname: "living-room"},
	}
	response, err := module.Apply(context.Background(), request)
	if err != nil || response.Result.Error != nil || !response.Result.Changed || writer.calls != 1 {
		t.Fatalf("response=%#v calls=%d err=%v", response, writer.calls, err)
	}
	if response.Result.Policy.Path.TargetID != "upstream" || response.Result.Policy.Path.Effect.Observed.State != "pending_renewal" || response.Result.Policy.Path.Effect.Applied.State != "server_applied" {
		t.Fatalf("path=%#v", response.Result.Policy.Path)
	}
	if policyStore.policy.Static.AssignedIP != "192.168.100.50" || gatewayStore.state.Hosts[0].TagName == "legacy_route" {
		t.Fatalf("static=%#v hosts=%#v", policyStore.policy.Static, gatewayStore.state.Hosts)
	}
	repeated, _ := module.Apply(context.Background(), &models.DeviceNetworkPolicyApplyRequest{
		DeviceID: request.DeviceID, TargetID: "upstream", Static: request.Static,
	})
	if repeated.Result.Changed || writer.calls != 1 || repeated.Result.Policy.Path.Effect.Observed.State != "pending_renewal" {
		t.Fatalf("repeat=%#v calls=%d", repeated.Result, writer.calls)
	}
}

func TestDeviceNetworkPolicyRejectsInvalidAndConflictingInputBeforeWrite(t *testing.T) {
	module, writer, policyStore, _ := deviceNetworkPolicyTestModule(t)
	invalid, _ := module.Apply(context.Background(), &models.DeviceNetworkPolicyApplyRequest{
		DeviceID: policyStore.policy.DeviceID, TargetID: "default",
		Static: &models.DeviceAddressPolicy{Enabled: true, AssignedIP: "192.168.100.50", BindIP: true, Hostname: "中文"},
	})
	if invalid.Result.Error == nil || invalid.Result.Error.Code != "validation_failed" || writer.calls != 0 {
		t.Fatalf("invalid=%#v calls=%d", invalid.Result, writer.calls)
	}
	policyStore.rules = &models.DevicePolicyRulesResult{Static: []*models.LANStaticAssigned{{AssignedIP: "192.168.100.50", AssignedMac: "AA:BB:CC:DD:EE:99", BindIP: true}}}
	conflict, _ := module.Apply(context.Background(), &models.DeviceNetworkPolicyApplyRequest{
		DeviceID: policyStore.policy.DeviceID, TargetID: "default",
		Static: &models.DeviceAddressPolicy{Enabled: true, AssignedIP: "192.168.100.50", BindIP: true},
	})
	if conflict.Result.Error == nil || conflict.Result.Error.Code != "conflict" || writer.calls != 0 {
		t.Fatalf("conflict=%#v calls=%d", conflict.Result, writer.calls)
	}
}

func TestDeviceNetworkPolicyRejectsStaleVersionAndReportsWriterRollback(t *testing.T) {
	module, writer, _, _ := deviceNetworkPolicyTestModule(t)
	request := &models.DeviceNetworkPolicyApplyRequest{DeviceID: "mac:aa:bb:cc:dd:ee:01", TargetID: "self", ExpectedVersion: "stale", Static: &models.DeviceAddressPolicy{}}
	stale, _ := module.Apply(context.Background(), request)
	if stale.Result.Error == nil || stale.Result.Error.Code != "conflict" || writer.calls != 0 {
		t.Fatalf("stale=%#v calls=%d", stale.Result, writer.calls)
	}
	request.ExpectedVersion = ""
	writer.err = errors.New("reload failed; original configuration restored")
	failed, _ := module.Apply(context.Background(), request)
	if failed.Result.Error == nil || failed.Result.Error.Code != "rolled_back" || failed.Result.Transaction.Status != "rolled_back" || writer.calls != 1 {
		t.Fatalf("failed=%#v calls=%d", failed.Result, writer.calls)
	}
}

func TestDeviceNetworkPolicyIdempotencyDoesNotRepeatWrite(t *testing.T) {
	module, writer, _, _ := deviceNetworkPolicyTestModule(t)
	request := &models.DeviceNetworkPolicyApplyRequest{
		DeviceID: "mac:aa:bb:cc:dd:ee:01", IdempotencyKey: "network-request-1", TargetID: "upstream",
		Static: &models.DeviceAddressPolicy{Enabled: true, AssignedIP: "192.168.100.60", BindIP: true, Hostname: "office-pc"},
	}
	first, _ := module.Apply(context.Background(), request)
	second, _ := module.Apply(context.Background(), request)
	if first.Result.Transaction.Status != "committed" || !second.Result.Transaction.Replayed || second.Result.Changed || writer.calls != 1 {
		t.Fatalf("first=%#v second=%#v calls=%d", first.Result, second.Result, writer.calls)
	}
	conflicting := *request
	conflicting.TargetID = "self"
	third, _ := module.Apply(context.Background(), &conflicting)
	if third.Result.Error == nil || third.Result.Error.Code != "conflict" || writer.calls != 1 {
		t.Fatalf("conflict=%#v calls=%d", third.Result, writer.calls)
	}
}

func TestMutateDeviceNetworkPolicyUsesOneHostAndPreservesUnrelatedOptions(t *testing.T) {
	directory := t.TempDir()
	config := "config dnsmasq\n\toption domainneeded '1'\n\nconfig host 'legacy'\n\toption mac 'AA:BB:CC:DD:EE:01'\n\toption leasetime '2h'\n\nconfig host 'duplicate'\n\toption mac 'AA:BB:CC:DD:EE:01'\n"
	if err := os.WriteFile(filepath.Join(directory, "dhcp"), []byte(config), 0600); err != nil {
		t.Fatal(err)
	}
	plan := gatewayPolicyExecutionPlan{Target: &gatewayTargetDescriptor{
		TagName: "t_auto_c0a86401", TagTitle: "main", Options: []string{"3,192.168.100.1"}, Materialize: true,
	}}
	input := StaticAssignmentWriteInput{AssignedMAC: "AA:BB:CC:DD:EE:01", AssignedIP: "192.168.100.50", BindIP: true, Hostname: "living-room"}
	if err := mutateDeviceNetworkPolicyConfigAt(directory, input, plan); err != nil {
		t.Fatal(err)
	}
	tree := uci.NewTree(directory)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		t.Fatal(err)
	}
	sections, _ := tree.GetSections("dhcp", "host")
	if len(sections) != 1 {
		t.Fatalf("host sections=%#v", sections)
	}
	if value, _ := tree.GetLast("dhcp", sections[0], "leasetime"); value != "2h" {
		t.Fatalf("unrelated option lost: %q", value)
	}
	for option, want := range map[string]string{"ip": "192.168.100.50", "name": "living-room", "tag": "t_auto_c0a86401"} {
		if value, _ := tree.GetLast("dhcp", sections[0], option); value != want {
			t.Fatalf("%s=%q want %q", option, value, want)
		}
	}
}

func TestDeviceNetworkPolicyWriterRestoresSnapshotAfterReloadFailure(t *testing.T) {
	originalPreflight, originalSnapshot, originalMutate, originalReload, originalRestore := deviceNetworkPolicyPreflight, deviceNetworkPolicySnapshot, deviceNetworkPolicyMutate, deviceNetworkPolicyReload, deviceNetworkPolicyRestore
	t.Cleanup(func() {
		deviceNetworkPolicyPreflight = originalPreflight
		deviceNetworkPolicySnapshot, deviceNetworkPolicyMutate, deviceNetworkPolicyReload, deviceNetworkPolicyRestore = originalSnapshot, originalMutate, originalReload, originalRestore
	})
	deviceNetworkPolicyPreflight = func(context.Context, StaticAssignmentWriteInput) error { return nil }
	deviceNetworkPolicySnapshot = func() (dhcpConfigSnapshot, error) { return dhcpConfigSnapshot{Data: []byte("before"), Mode: 0600}, nil }
	deviceNetworkPolicyMutate = func(StaticAssignmentWriteInput, gatewayPolicyExecutionPlan) error { return nil }
	deviceNetworkPolicyReload = func(context.Context) error { return errors.New("reload failed") }
	restored := false
	deviceNetworkPolicyRestore = func(context.Context, dhcpConfigSnapshot) error { restored = true; return nil }
	err := (&defaultDeviceNetworkPolicyWriter{}).Apply(context.Background(), StaticAssignmentWriteInput{AssignedMAC: "AA:BB:CC:DD:EE:01"}, gatewayPolicyExecutionPlan{Target: &gatewayTargetDescriptor{}})
	if err == nil || !restored {
		t.Fatalf("err=%v restored=%v", err, restored)
	}
}

func TestDeviceNetworkPolicyWriterReportsEveryFailureStage(t *testing.T) {
	originalPreflight, originalSnapshot, originalMutate, originalReload, originalRestore := deviceNetworkPolicyPreflight, deviceNetworkPolicySnapshot, deviceNetworkPolicyMutate, deviceNetworkPolicyReload, deviceNetworkPolicyRestore
	t.Cleanup(func() {
		deviceNetworkPolicyPreflight = originalPreflight
		deviceNetworkPolicySnapshot, deviceNetworkPolicyMutate, deviceNetworkPolicyReload, deviceNetworkPolicyRestore = originalSnapshot, originalMutate, originalReload, originalRestore
	})
	tests := []struct {
		name, wantStage, wantStatus                  string
		preflight, snapshot, mutate, reload, restore bool
	}{
		{name: "preflight", preflight: true, wantStage: "validate", wantStatus: "failed"},
		{name: "snapshot", snapshot: true, wantStage: "snapshot", wantStatus: "failed"},
		{name: "write rollback", mutate: true, wantStage: "apply", wantStatus: "rolled_back"},
		{name: "reload rollback", reload: true, wantStage: "verify", wantStatus: "rolled_back"},
		{name: "rollback failure", mutate: true, restore: true, wantStage: "recover", wantStatus: "recovery_required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deviceNetworkPolicyPreflight = func(context.Context, StaticAssignmentWriteInput) error {
				if test.preflight {
					return errors.New("preflight")
				}
				return nil
			}
			deviceNetworkPolicySnapshot = func() (dhcpConfigSnapshot, error) {
				if test.snapshot {
					return dhcpConfigSnapshot{}, errors.New("snapshot")
				}
				return dhcpConfigSnapshot{Data: []byte("before"), Mode: 0600}, nil
			}
			deviceNetworkPolicyMutate = func(StaticAssignmentWriteInput, gatewayPolicyExecutionPlan) error {
				if test.mutate {
					return errors.New("write")
				}
				return nil
			}
			deviceNetworkPolicyReload = func(context.Context) error {
				if test.reload {
					return errors.New("reload")
				}
				return nil
			}
			deviceNetworkPolicyRestore = func(context.Context, dhcpConfigSnapshot) error {
				if test.restore {
					return errors.New("restore")
				}
				return nil
			}
			err := (&defaultDeviceNetworkPolicyWriter{}).Apply(context.Background(), StaticAssignmentWriteInput{AssignedMAC: "AA:BB:CC:DD:EE:01"}, gatewayPolicyExecutionPlan{Target: &gatewayTargetDescriptor{}})
			var failure *networkPolicyApplyError
			if !errors.As(err, &failure) || failure.Stage != test.wantStage || failure.Status != test.wantStatus {
				t.Fatalf("failure = %#v (%v)", failure, err)
			}
		})
	}
}

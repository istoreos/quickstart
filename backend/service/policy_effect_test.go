package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type mutableLeaseObserver struct {
	observation leaseObservation
	err         error
}

func (observer *mutableLeaseObserver) Observe(context.Context, string, time.Time) (leaseObservation, error) {
	return observer.observation, observer.err
}

func TestPolicyEffectNeverClaimsTerminalActivation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 25, 9, 0, 0, 0, time.UTC)
	store := newJSONPolicyEffectStore(filepath.Join(t.TempDir(), "effects.json"))
	observer := &mutableLeaseObserver{observation: leaseObservation{SourceAvailable: true}}
	module := NewPolicyEffectModule(store, observer, func() time.Time { return now })
	static := &models.DeviceAddressPolicy{Enabled: true, AssignedIP: "192.168.1.20", BindIP: true}
	if err := module.RecordApplied(ctx, "mac:aa", "upstream", "v2", static); err != nil {
		t.Fatal(err)
	}
	pending := module.Resolve(ctx, "mac:aa", "AA:BB:CC:DD:EE:01", "upstream", "v2", static)
	if pending.Applied.State != "server_applied" || pending.Observed.State != "pending_renewal" || !pending.NeedsAttention {
		t.Fatalf("pending effect = %#v", pending)
	}
	observer.observation = leaseObservation{SourceAvailable: true, Observed: true, ObservedAt: now.Add(time.Minute)}
	observed := module.Resolve(ctx, "mac:aa", "AA:BB:CC:DD:EE:01", "upstream", "v2", static)
	if observed.Observed.State != "lease_observed_unverifiable" || observed.Observed.Reason != "terminal_gateway_and_dns_unverifiable" || observed.NeedsAttention {
		t.Fatalf("observed effect = %#v", observed)
	}
	// A new module simulates a service restart and must retain the bounded terminal observation.
	restarted := NewPolicyEffectModule(newJSONPolicyEffectStore(store.path), &mutableLeaseObserver{err: errors.New("should not be called")}, time.Now)
	restored := restarted.Resolve(ctx, "mac:aa", "AA:BB:CC:DD:EE:01", "upstream", "v2", static)
	if restored.Observed.State != "lease_observed_unverifiable" {
		t.Fatalf("restored effect = %#v", restored)
	}
}

func TestPolicyEffectHandlesMissingAndFailedObservationThenRecovers(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	observer := &mutableLeaseObserver{}
	module := NewPolicyEffectModule(newMemoryPolicyEffectStore(), observer, func() time.Time { return now })
	if err := module.RecordApplied(ctx, "mac:bb", "self", "v1", &models.DeviceAddressPolicy{}); err != nil {
		t.Fatal(err)
	}
	missing := module.Resolve(ctx, "mac:bb", "AA:BB:CC:DD:EE:02", "self", "v1", nil)
	if missing.Observed.State != "unverifiable" || missing.Observed.Reason != "lease_source_unavailable" {
		t.Fatalf("missing = %#v", missing)
	}
	observer.err = errors.New("read failed")
	failed := module.Resolve(ctx, "mac:bb", "AA:BB:CC:DD:EE:02", "self", "v1", nil)
	if failed.Observed.State != "observation_error" || !failed.NeedsAttention {
		t.Fatalf("failed = %#v", failed)
	}
	observer.err = nil
	observer.observation = leaseObservation{SourceAvailable: true}
	recovered := module.Resolve(ctx, "mac:bb", "AA:BB:CC:DD:EE:02", "self", "v1", nil)
	if recovered.Observed.State != "pending_renewal" || !recovered.NeedsAttention {
		t.Fatalf("recovered = %#v", recovered)
	}
}

func TestPolicyEffectStoreIsBoundedAndFailureIsExplicit(t *testing.T) {
	ctx := context.Background()
	store := newJSONPolicyEffectStore(filepath.Join(t.TempDir(), "effects.json"))
	store.limit = 1
	module := NewPolicyEffectModule(store, &mutableLeaseObserver{}, time.Now)
	if err := module.RecordFailure(ctx, "mac:one", "upstream", "default", "v1", nil, "server_apply_failed"); err != nil {
		t.Fatal(err)
	}
	effect := module.Resolve(ctx, "mac:one", "AA:BB:CC:DD:EE:01", "default", "v1", nil)
	if effect.Observed.State != "failed" || effect.Applied.State != "apply_failed" || !effect.NeedsAttention {
		t.Fatalf("failure = %#v", effect)
	}
	if err := module.RecordApplied(ctx, "mac:two", "self", "v2", nil); err == nil {
		t.Fatal("expected bounded store error")
	}
}

func TestGatewayAndDNSOptionsAreCompiledTogether(t *testing.T) {
	options := gatewayAndDNSOptions("192.168.1.9")
	if !sameStringSet(options, []string{"3,192.168.1.9", "6,192.168.1.9"}) {
		t.Fatalf("options = %#v", options)
	}
	state := gatewayPolicyTestState()
	state.DHCP.Tags = append(state.DHCP.Tags, DhcpTagRecord{TagName: "legacy", TagTitle: "bypass", Gateway: "192.168.100.9", DhcpOption: []string{"3,192.168.100.9"}})
	index := buildGatewayTargetIndex(state)
	target := index.targets[gatewayOpaqueID("bypass", "192.168.100.9")]
	if target == nil || !target.Materialize || !sameStringSet(target.Options, []string{"3,192.168.100.9", "6,192.168.100.9"}) {
		t.Fatalf("normalized target = %#v", target)
	}
}

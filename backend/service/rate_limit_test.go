package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type fakeRateLimitProvider struct {
	observation rateLimitProviderObservation
	applyCalls  int
	lastTarget  rateLimitTarget
}

func (provider *fakeRateLimitProvider) Name() string         { return "test" }
func (provider *fakeRateLimitProvider) IdentityKind() string { return "ipv4" }
func (provider *fakeRateLimitProvider) SupportsIPv6() bool   { return false }
func (provider *fakeRateLimitProvider) Inspect(context.Context, rateLimitTarget) (rateLimitProviderObservation, error) {
	return provider.observation, nil
}
func (provider *fakeRateLimitProvider) Apply(_ context.Context, target rateLimitTarget, _ models.DeviceSpeedPolicy) error {
	provider.applyCalls++
	provider.lastTarget = target
	return nil
}

type fakeRateLimitRouteResolver struct {
	route rateLimitRoute
	err   error
}

func (resolver *fakeRateLimitRouteResolver) Resolve(context.Context, string) (rateLimitRoute, error) {
	return resolver.route, resolver.err
}

func TestRateLimitModuleReportsVerifiedLocalPolicy(t *testing.T) {
	provider := &fakeRateLimitProvider{observation: rateLimitProviderObservation{
		Policy:     models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 5, DownloadSpeed: 30},
		Configured: true, Loaded: true, Verified: true, Offload: "risk",
	}}
	module := NewRateLimitModule(provider, &fakeRateLimitRouteResolver{route: rateLimitRoute{TargetID: "self", Gateway: "192.168.1.1", Local: true}})
	module.now = fixedRateLimitTime
	policy, enforcement, err := module.Inspect(context.Background(), rateLimitTarget{
		CurrentIPv4: "192.168.1.20", ReservedIP: "192.168.1.20", HasIPv6: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Enabled || enforcement.State != "verified" || !enforcement.CanApply || enforcement.ExecutionNode != "local" {
		t.Fatalf("unexpected policy/enforcement: %#v %#v", policy, enforcement)
	}
	if len(enforcement.Warnings) != 2 || enforcement.OffloadState != "risk" || enforcement.IPv6State != "unsupported" {
		t.Fatalf("expected IPv6 and offload warnings: %#v", enforcement)
	}
}

func TestRateLimitModuleReportsNativeMACPolicyCoversIPv6(t *testing.T) {
	provider := NewNativeRateLimitProvider("http://127.0.0.1:8765", nil)
	module := NewRateLimitModule(provider, nil)
	module.now = fixedRateLimitTime
	enforcement := module.enforcement(provider, rateLimitTarget{
		MAC:         "46:F1:F3:B9:9F:B9",
		CurrentIPv4: "192.168.1.20",
		HasIPv6:     true,
		Route:       rateLimitRoute{TargetID: "self", Gateway: "192.168.1.1", Local: true},
	}, rateLimitProviderObservation{Configured: true, Loaded: true, Verified: true, Offload: "compatible"})
	if enforcement.AddressState != "stable_identity" || enforcement.IPv6State != "covered" || len(enforcement.Warnings) != 0 {
		t.Fatalf("expected native MAC identity to cover IPv4 and IPv6: %#v", enforcement)
	}
}

func TestRateLimitModuleRejectsRemoteExecutionNode(t *testing.T) {
	provider := &fakeRateLimitProvider{}
	module := NewRateLimitModule(provider, &fakeRateLimitRouteResolver{route: rateLimitRoute{TargetID: "bypass-1", Gateway: "192.168.1.2", Local: false}})
	err := module.Apply(context.Background(), rateLimitTarget{CurrentIPv4: "192.168.1.20", ReservedIP: "192.168.1.20"}, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 5, DownloadSpeed: 30})
	var policyErr *PolicyError
	if !errors.As(err, &policyErr) || policyErr.Code != "execution_node_unavailable" {
		t.Fatalf("expected execution-node error, got %v", err)
	}
	if provider.applyCalls != 0 {
		t.Fatal("provider must not write on the wrong execution node")
	}
}

func TestRateLimitModuleRequiresStableIPv4ForIPProvider(t *testing.T) {
	provider := &fakeRateLimitProvider{}
	module := NewRateLimitModule(provider, &fakeRateLimitRouteResolver{route: rateLimitRoute{TargetID: "self", Local: true}})
	err := module.Apply(context.Background(), rateLimitTarget{CurrentIPv4: "192.168.1.20", ReservedIP: "192.168.1.21"}, models.DeviceSpeedPolicy{Enabled: true})
	var policyErr *PolicyError
	if !errors.As(err, &policyErr) || policyErr.Code != "address_reservation_required" {
		t.Fatalf("expected stable-address error, got %v", err)
	}
	if provider.applyCalls != 0 {
		t.Fatal("provider must not write an unstable IPv4 rule")
	}
}

func TestRateLimitModuleAllowsDisableWithoutLocalRouteOrReservation(t *testing.T) {
	provider := &fakeRateLimitProvider{}
	module := NewRateLimitModule(provider, &fakeRateLimitRouteResolver{route: rateLimitRoute{TargetID: "bypass-1", Local: false}})
	if err := module.Apply(context.Background(), rateLimitTarget{CurrentIPv4: "192.168.1.20"}, models.DeviceSpeedPolicy{}); err != nil {
		t.Fatal(err)
	}
	if provider.applyCalls != 1 {
		t.Fatalf("expected one cleanup write, got %d", provider.applyCalls)
	}
}

func TestRateLimitModuleFailsClosedWhenRouteCannotBeRead(t *testing.T) {
	module := NewRateLimitModule(&fakeRateLimitProvider{}, &fakeRateLimitRouteResolver{err: errors.New("boom")})
	_, enforcement, err := module.Inspect(context.Background(), rateLimitTarget{})
	if err != nil || enforcement.State != "unavailable" || enforcement.Reason != "route_unavailable" {
		t.Fatalf("unexpected fail-closed state: %#v, %v", enforcement, err)
	}
}

func fixedRateLimitTime() time.Time {
	return time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
}

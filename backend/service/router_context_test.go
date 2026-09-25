package service

import (
	"context"
	"errors"
	"testing"
)

type fakeRouterContextProbe struct {
	facts *RouterContextFacts
	err   error
}

func (probe *fakeRouterContextProbe) Probe(_ context.Context, lan string) (*RouterContextFacts, error) {
	if probe.err != nil {
		return nil, probe.err
	}
	copy := *probe.facts
	copy.LAN = lan
	return &copy, nil
}

func TestRouterContextAuthorityFixtures(t *testing.T) {
	tests := []struct {
		name      string
		facts     RouterContextFacts
		authority string
		position  string
		editable  bool
	}{
		{name: "ordinary primary router", facts: RouterContextFacts{LocalAddress: "192.168.1.1", LocalDHCPConfigured: true, LocalDHCPHealthy: true}, authority: "local", position: "lan_gateway_candidate", editable: true},
		{name: "downstream router external DHCP", facts: RouterContextFacts{LocalAddress: "192.168.1.2", UpstreamGateways: []string{"192.168.1.1"}, ObservedExternalServers: []string{"192.168.1.1"}}, authority: "external_observed", position: "downstream_router"},
		{name: "no DHCP evidence", facts: RouterContextFacts{LocalAddress: "192.168.1.2"}, authority: "none_detected", position: "lan_gateway_candidate"},
		{name: "local and external evidence", facts: RouterContextFacts{LocalDHCPConfigured: true, LocalDHCPHealthy: true, ObservedExternalServers: []string{"192.168.1.1"}}, authority: "ambiguous", position: "lan_gateway_candidate"},
		{name: "two external servers", facts: RouterContextFacts{ObservedExternalServers: []string{"192.168.1.9", "192.168.1.1"}}, authority: "ambiguous", position: "lan_gateway_candidate"},
		{name: "multiple default routes", facts: RouterContextFacts{LocalDHCPConfigured: true, LocalDHCPHealthy: true, UpstreamGateways: []string{"192.168.2.1", "192.168.1.1"}}, authority: "local", position: "ambiguous", editable: true},
		{name: "local service unhealthy", facts: RouterContextFacts{LocalDHCPConfigured: true}, authority: "error", position: "lan_gateway_candidate"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := BuildRouterContext(test.facts)
			if got.DHCPAuthority != test.authority || got.TopologyPosition != test.position || got.RouteEditability.Editable != test.editable {
				t.Fatalf("context = %#v", got)
			}
		})
	}
}

func TestRouterContextIsScopedBySelectedLANAndRecoversOnRetry(t *testing.T) {
	probe := &fakeRouterContextProbe{facts: &RouterContextFacts{LocalDHCPConfigured: true, LocalDHCPHealthy: true}}
	module := NewRouterContextModule(probe)
	response, err := module.Get(context.Background(), "guest_20")
	if err != nil || response.Result.LAN != "guest_20" || !response.Result.RouteEditability.Editable {
		t.Fatalf("selected LAN = %#v, %v", response, err)
	}
	probe.err = errors.New("ubus unavailable")
	response, err = module.Get(context.Background(), "guest_20")
	if err != nil || response.Result.DHCPAuthority != "error" || response.Result.RouteEditability.Editable {
		t.Fatalf("error context = %#v, %v", response, err)
	}
	probe.err = nil
	response, err = module.Get(context.Background(), "guest_20")
	if err != nil || response.Result.DHCPAuthority != "local" {
		t.Fatalf("recovered context = %#v, %v", response, err)
	}
}

func TestRouterContextRejectsUnsafeLANIdentifier(t *testing.T) {
	module := NewRouterContextModule(&fakeRouterContextProbe{facts: &RouterContextFacts{}})
	if _, err := module.Get(context.Background(), "lan;reboot"); err == nil {
		t.Fatal("expected invalid LAN identifier error")
	}
}

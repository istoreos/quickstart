package conflict

import (
	"net"
	"reflect"
	"testing"
)

func mustCIDR(t *testing.T, cidr string) net.IPNet {
	t.Helper()
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		t.Fatalf("ParseCIDR(%q): %v", cidr, err)
	}
	ipnet.IP = ip
	return *ipnet
}

func TestNoConflictIPChoosesAddressOutsideWANAndLAN(t *testing.T) {
	t.Parallel()

	wan := mustCIDR(t, "192.168.1.2/24")
	lan := mustCIDR(t, "192.168.100.1/24")

	if got := NoConflictIP(wan, lan); got != "192.168.101.1" {
		t.Fatalf("NoConflictIP = %q, want 192.168.101.1", got)
	}

	wan = mustCIDR(t, "192.168.100.2/24")
	lan = mustCIDR(t, "192.168.101.1/24")
	if got := NoConflictIP(wan, lan); got != "" {
		t.Fatalf("NoConflictIP = %q, want empty when both candidates conflict", got)
	}
}

func TestEvaluateWANLANConflictEnablesDHNSForDHCPOrGatewayMode(t *testing.T) {
	t.Parallel()

	wan := NetworkSection{IPNet: mustCIDR(t, "10.0.0.2/24")}

	for _, lan := range []NetworkSection{
		{Proto: "dhcp", IPNet: mustCIDR(t, "192.168.1.1/24")},
		{Proto: "static", Gateway: "192.168.1.254", IPNet: mustCIDR(t, "192.168.1.1/24")},
	} {
		decision := EvaluateWANLANConflict(wan, lan, false)
		if !decision.NeedDHNS || decision.NewLANIP != "" {
			t.Fatalf("unexpected decision for lan=%#v: %#v", lan, decision)
		}
	}
}

func TestEvaluateWANLANConflictSuggestsNewLANIPOnOverlap(t *testing.T) {
	t.Parallel()

	decision := EvaluateWANLANConflict(
		NetworkSection{IPNet: mustCIDR(t, "192.168.1.2/24")},
		NetworkSection{Section: "lan", Proto: "static", IPNet: mustCIDR(t, "192.168.1.1/24")},
		true,
	)

	if decision.NeedDHNS {
		t.Fatalf("expected conflict with replacement IP not to need DHNS: %#v", decision)
	}
	if !decision.Overlaps {
		t.Fatalf("expected overlap marker: %#v", decision)
	}
	if decision.NewLANIP != "192.168.100.1" {
		t.Fatalf("NewLANIP = %q, want 192.168.100.1", decision.NewLANIP)
	}
}

func TestEvaluateWANLANConflictUsesDHCPDisabledWhenNoOverlap(t *testing.T) {
	t.Parallel()

	wan := NetworkSection{IPNet: mustCIDR(t, "10.0.0.2/24")}
	lan := NetworkSection{Proto: "static", IPNet: mustCIDR(t, "192.168.1.1/24")}

	if decision := EvaluateWANLANConflict(wan, lan, true); !decision.NeedDHNS {
		t.Fatalf("expected disabled LAN DHCP to need DHNS: %#v", decision)
	}
	if decision := EvaluateWANLANConflict(wan, lan, false); decision.NeedDHNS {
		t.Fatalf("expected enabled LAN DHCP not to need DHNS: %#v", decision)
	}
}

func TestPlanWANLANConflictBuildsLANReplacementCommands(t *testing.T) {
	t.Parallel()

	plan := PlanWANLANConflict(
		NetworkSection{IPNet: mustCIDR(t, "192.168.1.2/24")},
		NetworkSection{Section: "lan", Proto: "static", IPNet: mustCIDR(t, "192.168.1.1/24")},
		true,
	)

	if plan.NeedDHNS {
		t.Fatalf("expected overlap replacement not to need DHNS: %#v", plan)
	}
	wantCommands := []string{
		"set network.lan.ipaddr=192.168.100.1",
		"commit network",
	}
	if !reflect.DeepEqual(plan.NetworkUCICommands, wantCommands) {
		t.Fatalf("NetworkUCICommands = %#v, want %#v", plan.NetworkUCICommands, wantCommands)
	}
	if !plan.ReloadNetwork {
		t.Fatalf("expected network reload marker: %#v", plan)
	}
}

func TestPlanWANLANConflictNeedsDHNSForGatewayOrDisabledLANDHCP(t *testing.T) {
	t.Parallel()

	wan := NetworkSection{IPNet: mustCIDR(t, "10.0.0.2/24")}
	for _, tc := range []struct {
		name            string
		lan             NetworkSection
		lanDHCPDisabled bool
	}{
		{
			name: "gateway static lan",
			lan:  NetworkSection{Proto: "static", Gateway: "10.0.0.1", IPNet: mustCIDR(t, "192.168.1.1/24")},
		},
		{
			name:            "dhcp disabled",
			lan:             NetworkSection{Proto: "static", IPNet: mustCIDR(t, "192.168.1.1/24")},
			lanDHCPDisabled: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := PlanWANLANConflict(wan, tc.lan, tc.lanDHCPDisabled)
			if !plan.NeedDHNS {
				t.Fatalf("expected DHNS plan: %#v", plan)
			}
			if len(plan.NetworkUCICommands) != 0 || plan.ReloadNetwork {
				t.Fatalf("expected no UCI command side effects: %#v", plan)
			}
		})
	}
}

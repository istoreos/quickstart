package recovery

import (
	"net"
	"testing"
)

func mustCIDR(cidr string) net.IPNet {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(err)
	}
	ipnet.IP = ip
	return *ipnet
}

func TestPlanDHCPChangeRejectsInvalidLease(t *testing.T) {
	t.Parallel()

	decision := PlanDHCPChange(DHCPChangeInput{
		LAN:  Section{IPNet: mustCIDR("192.168.1.1/24")},
		DHCP: DHCPLease{IP: "bad", Subnet: "255.255.255.0"},
	})

	if decision.Valid {
		t.Fatalf("expected invalid decision, got %#v", decision)
	}
}

func TestPlanDHCPChangeDetectsConflictAndRestartsLANWhenPlanBIsUp(t *testing.T) {
	t.Parallel()

	decision := PlanDHCPChange(DHCPChangeInput{
		LAN:   Section{IPNet: mustCIDR("192.168.1.1/24")},
		PlanB: &Section{Up: true},
		DHCP:  DHCPLease{IP: "192.168.1.50", Subnet: "255.255.255.0"},
	})

	if !decision.Valid || !decision.Conflict || !decision.RestartLAN {
		t.Fatalf("unexpected conflict decision: %#v", decision)
	}
	if decision.CreatePlanB || decision.IfupPlanB {
		t.Fatalf("did not expect planb setup on conflict: %#v", decision)
	}
}

func TestPlanDHCPChangePlansPlanBCreationWhenMissing(t *testing.T) {
	t.Parallel()

	decision := PlanDHCPChange(DHCPChangeInput{
		LAN:  Section{Device: "br-lan", IPNet: mustCIDR("192.168.1.1/24")},
		DHCP: DHCPLease{IP: "10.0.0.10", Subnet: "255.255.255.0"},
	})

	if !decision.Valid || decision.Conflict {
		t.Fatalf("unexpected decision: %#v", decision)
	}
	if !decision.CreatePlanB || !decision.IfupPlanB {
		t.Fatalf("expected missing planb to be created and brought up: %#v", decision)
	}
	want := []string{
		"set network.planb=interface",
		"set network.planb.proto=dhcp",
		"set network.planb.device=br-lan",
		"set network.planb.auto=0",
	}
	if len(decision.PlanBUCICommands) != len(want) {
		t.Fatalf("unexpected planb commands: %#v", decision.PlanBUCICommands)
	}
	for i := range want {
		if decision.PlanBUCICommands[i] != want[i] {
			t.Fatalf("command[%d]=%q, want %q", i, decision.PlanBUCICommands[i], want[i])
		}
	}
}

func TestPlanDHCPChangeDoesNotIfupAlreadyUpPlanB(t *testing.T) {
	t.Parallel()

	decision := PlanDHCPChange(DHCPChangeInput{
		LAN:   Section{Device: "br-lan", IPNet: mustCIDR("192.168.1.1/24")},
		PlanB: &Section{Up: true},
		DHCP:  DHCPLease{IP: "10.0.0.10", Subnet: "255.255.255.0"},
	})

	if !decision.Valid || decision.Conflict || decision.CreatePlanB || decision.IfupPlanB {
		t.Fatalf("unexpected planb-up decision: %#v", decision)
	}
}

func TestPlanIfaceChangeWhenWanIsUpAndLanNeedsDHNS(t *testing.T) {
	t.Parallel()

	decision := PlanIfaceChange(IfaceChangeInput{
		LAN:   Section{Proto: "dhcp", Up: false},
		WAN:   &Section{Up: true},
		PlanB: &Section{Up: true},
	})

	if !decision.StopUdhcpc || !decision.IfdownPlanB || !decision.SetupStaticIPInOtherNS || !decision.IfupLAN {
		t.Fatalf("unexpected wan-up decision: %#v", decision)
	}
	if decision.StopDhns || decision.StartUdhcpc {
		t.Fatalf("unexpected extra actions: %#v", decision)
	}
}

func TestPlanIfaceChangeWhenWanIsUpAndStaticLanNeedsConflictCheck(t *testing.T) {
	t.Parallel()

	decision := PlanIfaceChange(IfaceChangeInput{
		LAN: Section{Proto: "static", Up: true},
		WAN: &Section{Up: true},
	})

	if !decision.StopUdhcpc || !decision.CheckWANLANConflict {
		t.Fatalf("expected udhcpc stop and conflict check: %#v", decision)
	}
	if decision.StopDhns || decision.SetupStaticIPInOtherNS || decision.IfupLAN {
		t.Fatalf("unexpected extra actions: %#v", decision)
	}
}

func TestPlanIfaceChangeWhenWanDownAndLanNeedsDHNS(t *testing.T) {
	t.Parallel()

	decision := PlanIfaceChange(IfaceChangeInput{
		LAN:   Section{Proto: "static", Gateway: "192.168.1.1", Up: false},
		PlanB: &Section{Up: true},
	})

	if !decision.StopUdhcpc || !decision.IfdownPlanBAndIfupLAN || !decision.CheckLANHasIP || !decision.SetupStaticIPInOtherNS {
		t.Fatalf("unexpected wan-down lan-dhns decision: %#v", decision)
	}
	if decision.StartUdhcpc {
		t.Fatalf("did not expect udhcpc start: %#v", decision)
	}
}

func TestPlanIfaceChangeWhenNetworkAlreadyOK(t *testing.T) {
	t.Parallel()

	decision := PlanIfaceChange(IfaceChangeInput{
		LAN:       Section{Proto: "static", Up: true},
		NetworkOK: true,
	})

	if !decision.StopUdhcpc {
		t.Fatalf("expected stop udhcpc: %#v", decision)
	}
	if decision.StartUdhcpc || decision.IfupLAN || decision.SetupStaticIPInOtherNS {
		t.Fatalf("unexpected recovery actions: %#v", decision)
	}
}

func TestPlanIfaceChangeStartsUdhcpcWhenLanIsOnlyRecoveryPath(t *testing.T) {
	t.Parallel()

	decision := PlanIfaceChange(IfaceChangeInput{
		LAN: Section{Proto: "static", Up: false},
	})

	if !decision.IfupLAN || !decision.StartUdhcpc {
		t.Fatalf("expected lan ifup and udhcpc start: %#v", decision)
	}
	if decision.StopUdhcpc || decision.SetupStaticIPInOtherNS {
		t.Fatalf("unexpected actions: %#v", decision)
	}
}

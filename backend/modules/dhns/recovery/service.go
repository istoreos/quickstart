package recovery

import (
	"fmt"
	"net"
)

type Section struct {
	Device  string
	Proto   string
	Gateway string
	IPNet   net.IPNet
	Up      bool
}

type DHCPLease struct {
	IP      string
	Subnet  string
	Gateway string
}

type DHCPChangeInput struct {
	LAN   Section
	PlanB *Section
	DHCP  DHCPLease
}

type DHCPDecision struct {
	Valid            bool
	Conflict         bool
	RestartLAN       bool
	CreatePlanB      bool
	IfupPlanB        bool
	PlanBUCICommands []string
}

type IfaceChangeInput struct {
	LAN       Section
	WAN       *Section
	PlanB     *Section
	NetworkOK bool
}

type IfaceDecision struct {
	StopUdhcpc             bool
	StopDhns               bool
	SetupStaticIPInOtherNS bool
	CheckWANLANConflict    bool
	IfdownPlanB            bool
	IfdownPlanBAndIfupLAN  bool
	IfupLAN                bool
	CheckLANHasIP          bool
	StartUdhcpc            bool
}

func PlanDHCPChange(input DHCPChangeInput) DHCPDecision {
	var newIPNet net.IPNet
	newIPNet.IP = net.ParseIP(input.DHCP.IP)
	newIPNet.Mask = net.IPMask(net.ParseIP(input.DHCP.Subnet))
	if newIPNet.IP == nil || newIPNet.Mask == nil {
		return DHCPDecision{}
	}

	if newIPNet.Contains(input.LAN.IPNet.IP) || input.LAN.IPNet.Contains(newIPNet.IP) {
		return DHCPDecision{
			Valid:      true,
			Conflict:   true,
			RestartLAN: input.PlanB != nil && input.PlanB.Up,
		}
	}

	decision := DHCPDecision{Valid: true}
	if input.PlanB == nil {
		decision.CreatePlanB = true
		decision.PlanBUCICommands = []string{
			"set network.planb=interface",
			"set network.planb.proto=dhcp",
			fmt.Sprintf("set network.planb.device=%s", input.LAN.Device),
			"set network.planb.auto=0",
		}
	}
	decision.IfupPlanB = input.PlanB == nil || !input.PlanB.Up
	return decision
}

func PlanIfaceChange(input IfaceChangeInput) IfaceDecision {
	if input.WAN != nil && input.WAN.Up {
		decision := IfaceDecision{StopUdhcpc: true}
		if input.PlanB != nil {
			decision.IfdownPlanB = true
		}
		if lanNeedsDHNS(input.LAN) {
			decision.SetupStaticIPInOtherNS = true
		} else {
			decision.CheckWANLANConflict = true
		}
		if !input.LAN.Up || (input.PlanB != nil && input.PlanB.Up) {
			decision.IfupLAN = true
		}
		return decision
	}

	if lanNeedsDHNS(input.LAN) {
		decision := IfaceDecision{
			StopUdhcpc:             true,
			CheckLANHasIP:          true,
			SetupStaticIPInOtherNS: true,
		}
		if input.PlanB != nil && input.PlanB.Up {
			decision.IfdownPlanBAndIfupLAN = true
		} else if !input.LAN.Up {
			decision.IfupLAN = true
		}
		return decision
	}

	if input.NetworkOK {
		return IfaceDecision{StopUdhcpc: true}
	}
	if input.PlanB != nil && input.PlanB.Up && input.PlanB.IPNet.IP != nil {
		return IfaceDecision{StopUdhcpc: true}
	}

	return IfaceDecision{
		IfupLAN:     !input.LAN.Up,
		StartUdhcpc: true,
	}
}

func lanNeedsDHNS(lan Section) bool {
	return lan.Proto == "dhcp" || (lan.Proto == "static" && lan.Gateway != "")
}

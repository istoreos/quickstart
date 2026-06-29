package conflict

import (
	"fmt"
	"net"
)

type NetworkSection struct {
	Section string
	Proto   string
	Gateway string
	IPNet   net.IPNet
}

type Decision struct {
	NeedDHNS bool
	NewLANIP string
	Overlaps bool
}

type WANLANPlan struct {
	NeedDHNS           bool
	NetworkUCICommands []string
	ReloadNetwork      bool
}

func NoConflictIP(netIP1, netIP2 net.IPNet) string {
	ip1 := net.ParseIP("192.168.100.1")
	ip2 := net.ParseIP("192.168.101.1")
	if !netIP2.Contains(ip1) && !netIP1.Contains(ip1) {
		return ip1.String()
	}
	if !netIP2.Contains(ip2) && !netIP1.Contains(ip2) {
		return ip2.String()
	}
	return ""
}

func EvaluateWANLANConflict(wanSec NetworkSection, lanSec NetworkSection, lanDHCPDisabled bool) Decision {
	if lanSec.Proto == "dhcp" || (lanSec.Proto == "static" && lanSec.Gateway != "") {
		return Decision{NeedDHNS: true}
	}
	if wanSec.IPNet.Contains(lanSec.IPNet.IP) || lanSec.IPNet.Contains(wanSec.IPNet.IP) {
		return Decision{NewLANIP: NoConflictIP(wanSec.IPNet, lanSec.IPNet), Overlaps: true}
	}
	if lanDHCPDisabled {
		return Decision{NeedDHNS: true}
	}
	return Decision{}
}

func PlanWANLANConflict(wanSec NetworkSection, lanSec NetworkSection, lanDHCPDisabled bool) WANLANPlan {
	decision := EvaluateWANLANConflict(wanSec, lanSec, lanDHCPDisabled)
	if decision.Overlaps {
		if decision.NewLANIP == "" {
			return WANLANPlan{}
		}
		return WANLANPlan{
			NetworkUCICommands: []string{
				fmt.Sprintf("set network.%s.ipaddr=%s", lanSec.Section, decision.NewLANIP),
				"commit network",
			},
			ReloadNetwork: true,
		}
	}
	if decision.NeedDHNS {
		return WANLANPlan{NeedDHNS: true}
	}
	return WANLANPlan{}
}

package interfacewrite

import (
	"fmt"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Input struct {
	Configs []*models.NetworkInterfaceConfig
}

type Snapshot struct {
	Interfaces     []*models.NetworkInterfaceInfo
	DeviceSections []DeviceSnapshot
	FirewallZones  []FirewallZoneSnapshot
}

type DeviceSnapshot struct {
	SectionName string
	Name        string
}

type FirewallZoneSnapshot struct {
	SectionName string
	Name        string
	Networks    []string
}

type WritePlan struct {
	DeleteInterfaces []string
}

type CommandPlan struct {
	DeleteCommands    []string
	BridgeCommands    [][]string
	InterfaceCommands [][]string
	FirewallCommands  [][]string
}

func BuildDeleteCommands(deleteInterfaces []string) []string {
	commands := make([]string, 0, len(deleteInterfaces))
	for _, name := range deleteInterfaces {
		if name == "" {
			continue
		}
		commands = append(commands, fmt.Sprintf("uci del network.%v", name))
	}
	return commands
}

func BuildDeletePlan(existing []*models.NetworkInterfaceInfo, configs []*models.NetworkInterfaceConfig) []string {
	next := make(map[string]struct{}, len(configs))
	for _, cfg := range configs {
		if cfg != nil {
			next[cfg.Name] = struct{}{}
		}
	}

	deleteCons := make([]string, 0)
	for _, con := range existing {
		if con == nil {
			continue
		}
		if _, ok := next[con.Name]; !ok {
			deleteCons = append(deleteCons, con.Name)
		}
	}
	return deleteCons
}

func NormalizeDevices(name string, devices []string, hasPort func(string) bool) []string {
	out := append([]string(nil), devices...)
	if name != "lan" {
		return out
	}
	for _, device := range []string{"dsm-ext", "pve-ext", "vm-ext"} {
		if hasPort != nil && hasPort(device) {
			out = append(out, device)
		}
	}
	return out
}

func BuildBridgePlan(cfg *models.NetworkInterfaceConfig, devices []string, sections []DeviceSnapshot) []string {
	deviceName := "br-" + cfg.Name
	brSection := ""
	for _, sec := range sections {
		if sec.Name == deviceName {
			brSection = sec.SectionName
			break
		}
	}
	if brSection == "" {
		cmds := []string{
			"uci add network device",
			"uci set network.@device[-1].type='bridge'",
			fmt.Sprintf("uci set network.@device[-1].name=%v", deviceName),
		}
		for _, device := range devices {
			cmds = append(cmds, fmt.Sprintf("uci add_list network.@device[-1].ports=%v", device))
		}
		return cmds
	}

	cmds := []string{
		fmt.Sprintf("uci del network.%v.ports", brSection),
		fmt.Sprintf("uci del network.%v.ports", brSection),
	}
	for _, device := range devices {
		cmds = append(cmds, fmt.Sprintf("uci add_list network.%v.ports=%v", brSection, device))
	}
	return cmds
}

func BuildInterfacePlan(cfg *models.NetworkInterfaceConfig, deviceName string, bridged bool) []string {
	cmds := []string{
		fmt.Sprintf("uci set network.%v=interface", cfg.Name),
		fmt.Sprintf("uci set network.%v.proto=%v", cfg.Name, cfg.Proto),
	}
	if bridged {
		cmds = append(cmds, fmt.Sprintf("uci set network.%v.device=%v", cfg.Name, deviceName))
		return cmds
	}
	cmds = append(cmds, fmt.Sprintf("uci del network.%v.device", cfg.Name))
	if deviceName != "" {
		cmds = append(cmds, fmt.Sprintf("uci set network.%v.device=%v", cfg.Name, deviceName))
	}
	return cmds
}

func BuildFirewallBindingPlan(cfg *models.NetworkInterfaceConfig, zones []FirewallZoneSnapshot) []string {
	isBound := false
	firewallSec := ""
	for _, zone := range zones {
		for _, net := range zone.Networks {
			if net == cfg.Name {
				isBound = true
			}
		}
		if zone.Name == cfg.FirewallType {
			firewallSec = zone.SectionName
		}
	}
	if isBound || firewallSec == "" {
		return nil
	}
	return []string{fmt.Sprintf("uci add_list firewall.%v.network=%v", firewallSec, cfg.Name)}
}

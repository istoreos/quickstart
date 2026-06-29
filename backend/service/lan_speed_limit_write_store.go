package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type LanSpeedLimitWriteStore interface {
	ReadRuleMatches(ctx context.Context) ([]SpeedLimitRuleMatch, []SpeedLimitRuleMatch, error)
	ApplyPlan(ctx context.Context, plan SpeedLimitWritePlan) error
	ApplyModuleConfig(ctx context.Context, input SpeedLimitModuleInput) error
}

type defaultLanSpeedLimitWriteStore struct{}

var lanSpeedLimitWriteExec = func(ctx context.Context, commands []string) error {
	return utils.BatchRun(ctx, commands, 0)
}

var lanSpeedLimitWriteLoadConfig = uci.LoadConfig

func NewDefaultLanSpeedLimitWriteStore() LanSpeedLimitWriteStore {
	return &defaultLanSpeedLimitWriteStore{}
}

func findSpeedLimitRuleMatchByIP(matches []SpeedLimitRuleMatch, ip string) (SpeedLimitRuleMatch, bool) {
	for _, match := range matches {
		if match.MatchIP == ip {
			return match, true
		}
	}
	return SpeedLimitRuleMatch{}, false
}

func findSpeedLimitRuleMatchByMAC(matches []SpeedLimitRuleMatch, mac string) (SpeedLimitRuleMatch, bool) {
	normalized := normalizeLanSpeedLimitedDeviceMAC(mac)
	for _, match := range matches {
		if normalizeLanSpeedLimitedDeviceMAC(match.MatchMAC) == normalized {
			return match, true
		}
	}
	return SpeedLimitRuleMatch{}, false
}

func buildBlockedRuleName(mac string) string {
	return "BL_" + strings.ReplaceAll(normalizeLanSpeedLimitedDeviceMAC(mac), ":", "")
}

func BuildSpeedLimitWritePlan(input SpeedLimitWriteInput, eqosMatches, firewallMatches []SpeedLimitRuleMatch) SpeedLimitWritePlan {
	plan := SpeedLimitWritePlan{
		Input:          input,
		DeleteSections: make([]SpeedLimitRuleMatch, 0, 2),
	}

	if match, ok := findSpeedLimitRuleMatchByIP(eqosMatches, input.IP); ok {
		plan.DeleteSections = append(plan.DeleteSections, match)
	}
	if match, ok := findSpeedLimitRuleMatchByMAC(firewallMatches, input.MAC); ok {
		plan.DeleteSections = append(plan.DeleteSections, match)
	}

	if input.Action == "add" || input.Action == "modify" {
		if input.NetworkAccess {
			plan.AddSpeedLimit = true
		} else {
			plan.AddBlockRule = true
		}
	}

	return plan
}

func buildSpeedLimitWriteCommands(input SpeedLimitWriteInput, plan SpeedLimitWritePlan) []string {
	commands := make([]string, 0, len(plan.DeleteSections)*2+7)
	for _, match := range plan.DeleteSections {
		commands = append(commands, fmt.Sprintf("uci del %s.%s", match.Config, match.SectionName))
		commands = append(commands, fmt.Sprintf("uci commit %s", match.Config))
	}

	if plan.AddSpeedLimit {
		commands = append(commands,
			"uci add eqos device",
			fmt.Sprintf("uci set eqos.@device[-1].ip='%s'", input.IP),
			fmt.Sprintf("uci set eqos.@device[-1].download='%d'", input.DownloadSpeed),
			fmt.Sprintf("uci set eqos.@device[-1].upload='%d'", input.UploadSpeed),
			fmt.Sprintf("uci set eqos.@device[-1].comment='%s'", input.Comment),
		)
	}

	if plan.AddBlockRule {
		commands = append(commands,
			"uci add firewall rule",
			fmt.Sprintf("uci set firewall.@rule[-1].name='%s'", buildBlockedRuleName(input.MAC)),
			"uci set firewall.@rule[-1].src='lan'",
			"uci set firewall.@rule[-1].dest='wan'",
			"uci set firewall.@rule[-1].target='REJECT'",
			"uci set firewall.@rule[-1].proto='all'",
			fmt.Sprintf("uci set firewall.@rule[-1].src_mac='%s'", normalizeLanSpeedLimitedDeviceMAC(input.MAC)),
		)
	}

	return commands
}

func (store *defaultLanSpeedLimitWriteStore) ApplyPlan(ctx context.Context, plan SpeedLimitWritePlan) error {
	_ = store
	return lanSpeedLimitWriteExec(ctx, buildSpeedLimitWriteCommands(plan.Input, plan))
}

func (store *defaultLanSpeedLimitWriteStore) ReadRuleMatches(ctx context.Context) ([]SpeedLimitRuleMatch, []SpeedLimitRuleMatch, error) {
	_ = ctx
	_ = lanSpeedLimitWriteLoadConfig("eqos", true)
	_ = lanSpeedLimitWriteLoadConfig("firewall", true)

	eqosSecs, ok := uci.GetSections("eqos", "device")
	if !ok {
		return nil, nil, errors.New("fetch eqos device failed")
	}
	firewallSecs, ok := uci.GetSections("firewall", "rule")
	if !ok {
		return nil, nil, errors.New("fetch firewall rule failed")
	}

	eqosMatches := make([]SpeedLimitRuleMatch, 0, len(eqosSecs))
	for _, sectionName := range eqosSecs {
		ip, _ := uci.GetLast("eqos", sectionName, "ip")
		eqosMatches = append(eqosMatches, SpeedLimitRuleMatch{
			Config:      "eqos",
			SectionName: sectionName,
			MatchIP:     ip,
		})
	}

	firewallMatches := make([]SpeedLimitRuleMatch, 0, len(firewallSecs))
	for _, sectionName := range firewallSecs {
		name, _ := uci.GetLast("firewall", sectionName, "name")
		mac, _ := uci.GetLast("firewall", sectionName, "src_mac")
		target, _ := uci.GetLast("firewall", sectionName, "target")
		if _, ok := buildBlockedDeviceRule(mac, name, target); !ok {
			continue
		}
		firewallMatches = append(firewallMatches, SpeedLimitRuleMatch{
			Config:      "firewall",
			SectionName: sectionName,
			MatchMAC:    normalizeLanSpeedLimitedDeviceMAC(mac),
		})
	}

	return eqosMatches, firewallMatches, nil
}

func (store *defaultLanSpeedLimitWriteStore) ApplyModuleConfig(ctx context.Context, input SpeedLimitModuleInput) error {
	_ = store
	enabled := 0
	if input.Enabled {
		enabled = 1
	}
	return lanSpeedLimitWriteExec(ctx, []string{
		fmt.Sprintf("uci set eqos.@eqos[0].enabled='%d'", enabled),
		fmt.Sprintf("uci set eqos.@eqos[0].upload='%d'", input.UploadSpeed),
		fmt.Sprintf("uci set eqos.@eqos[0].download='%d'", input.DownloadSpeed),
		"uci commit eqos",
	})
}

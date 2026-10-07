package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/utils"
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

var lanSpeedLimitWriteApplyPlan = func(_ context.Context, plan SpeedLimitWritePlan) error {
	return applySpeedLimitPlanAt(filepath.Dir(dhcpConfigPath), plan)
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

func (store *defaultLanSpeedLimitWriteStore) ApplyPlan(ctx context.Context, plan SpeedLimitWritePlan) error {
	_ = store
	return lanSpeedLimitWriteApplyPlan(ctx, plan)
}

func applySpeedLimitPlanAt(configDir string, plan SpeedLimitWritePlan) error {
	tree := uci.NewTree(configDir)
	needEqos, needFirewall := plan.AddSpeedLimit, plan.AddBlockRule
	for _, match := range plan.DeleteSections {
		needEqos = needEqos || match.Config == "eqos"
		needFirewall = needFirewall || match.Config == "firewall"
	}
	if needEqos {
		if err := tree.LoadConfig("eqos", true); err != nil {
			return err
		}
	}
	if needFirewall {
		if err := tree.LoadConfig("firewall", true); err != nil {
			return err
		}
	}
	if err := applySpeedLimitPlanToTree(tree, plan); err != nil {
		return err
	}
	return tree.Commit()
}

func applySpeedLimitPlanToTree(tree uci.Tree, plan SpeedLimitWritePlan) error {
	for _, match := range plan.DeleteSections {
		if match.Config != "eqos" && match.Config != "firewall" {
			return errors.New("unsupported speed policy config")
		}
		tree.DelSection(match.Config, match.SectionName)
	}
	sectionID := func(prefix, identity string) string {
		sum := sha256.Sum256([]byte(identity))
		return prefix + hex.EncodeToString(sum[:6])
	}
	if plan.AddSpeedLimit {
		section := sectionID("quickstart_speed_", plan.Input.IP)
		tree.DelSection("eqos", section)
		if err := tree.AddSection("eqos", section, "device"); err != nil {
			return err
		}
		if !tree.Set("eqos", section, "ip", plan.Input.IP) ||
			!tree.Set("eqos", section, "download", strconv.FormatInt(plan.Input.DownloadSpeed, 10)) ||
			!tree.Set("eqos", section, "upload", strconv.FormatInt(plan.Input.UploadSpeed, 10)) ||
			!tree.Set("eqos", section, "comment", safeSpeedPolicyComment(plan.Input.Comment)) {
			return errors.New("set speed policy")
		}
	}
	if plan.AddBlockRule {
		section := sectionID("quickstart_block_", normalizeLanSpeedLimitedDeviceMAC(plan.Input.MAC))
		tree.DelSection("firewall", section)
		if err := tree.AddSection("firewall", section, "rule"); err != nil {
			return err
		}
		if !tree.Set("firewall", section, "name", buildBlockedRuleName(plan.Input.MAC)) ||
			!tree.Set("firewall", section, "src", "lan") || !tree.Set("firewall", section, "dest", "wan") ||
			!tree.Set("firewall", section, "target", "REJECT") || !tree.Set("firewall", section, "proto", "all") ||
			!tree.Set("firewall", section, "src_mac", normalizeLanSpeedLimitedDeviceMAC(plan.Input.MAC)) {
			return errors.New("set access policy")
		}
	}
	return nil
}

func safeSpeedPolicyComment(value string) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > 64 {
		runes = runes[:64]
	}
	for index, character := range runes {
		if character == '\'' {
			runes[index] = '’'
		} else if unicode.IsControl(character) {
			runes[index] = ' '
		}
	}
	return strings.TrimSpace(string(runes))
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

package service

import (
	"context"
	"errors"
	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

func scopedSpeedRuleTree(ctx context.Context) (uci.Tree, error) {
	if selectedLAN(ctx) != defaultLAN {
		return nil, errors.New("legacy speed provider supports only the default LAN")
	}
	tree := uci.NewTree(deviceRestrictionConfigDir())
	for _, config := range []string{"eqos", "firewall"} {
		if err := tree.LoadConfig(config, true); err != nil {
			return nil, err
		}
	}
	return tree, nil
}

func readScopedSpeedRuleMatches(ctx context.Context) ([]SpeedLimitRuleMatch, []SpeedLimitRuleMatch, error) {
	tree, err := scopedSpeedRuleTree(ctx)
	if err != nil {
		return nil, nil, err
	}
	var speed, blocks []SpeedLimitRuleMatch
	sections, _ := tree.GetSections("eqos", "device")
	for _, section := range sections {
		ip, _ := tree.GetLast("eqos", section, "ip")
		speed = append(speed, SpeedLimitRuleMatch{Config: "eqos", SectionName: section, MatchIP: ip})
	}
	sections, _ = tree.GetSections("firewall", "rule")
	for _, section := range sections {
		owner, _ := tree.GetLast("firewall", section, "quickstart_owner")
		if owner != "" || !accessRuleInScope(tree, section, selectedLANScope(ctx)) {
			continue
		}
		mac, _ := tree.GetLast("firewall", section, "src_mac")
		name, _ := tree.GetLast("firewall", section, "name")
		target, _ := tree.GetLast("firewall", section, "target")
		if _, ok := buildBlockedDeviceRule(mac, name, target); ok {
			blocks = append(blocks, SpeedLimitRuleMatch{Config: "firewall", SectionName: section, MatchMAC: normalizeLanSpeedLimitedDeviceMAC(mac)})
		}
	}
	return speed, blocks, nil
}

func readScopedSpeedRuleLists(ctx context.Context) ([]*models.LANCtrlSpeedLimitItem, []*models.LANCtrlSpeedLimitItem, error) {
	tree, err := scopedSpeedRuleTree(ctx)
	if err != nil {
		return nil, nil, err
	}
	var speed, blocks []*models.LANCtrlSpeedLimitItem
	sections, _ := tree.GetSections("eqos", "device")
	for _, section := range sections {
		ip, _ := tree.GetLast("eqos", section, "ip")
		up, _ := tree.GetLast("eqos", section, "upload")
		down, _ := tree.GetLast("eqos", section, "download")
		comment, _ := tree.GetLast("eqos", section, "comment")
		speed = append(speed, buildSpeedLimitRule(ip, up, down, comment))
	}
	sections, _ = tree.GetSections("firewall", "rule")
	for _, section := range sections {
		if !accessRuleInScope(tree, section, selectedLANScope(ctx)) {
			continue
		}
		mac, _ := tree.GetLast("firewall", section, "src_mac")
		name, _ := tree.GetLast("firewall", section, "name")
		target, _ := tree.GetLast("firewall", section, "target")
		if item, ok := buildBlockedDeviceRule(mac, name, target); ok {
			blocks = append(blocks, item)
		}
	}
	return blocks, speed, nil
}

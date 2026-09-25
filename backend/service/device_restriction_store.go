package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
)

var (
	deviceRestrictionConfigDir = func() string { return filepath.Dir(dhcpConfigPath) }
	deviceRestrictionApply     = func(ctx context.Context, configs []string) error {
		return NewDefaultLanSpeedLimitApply().Apply(ctx, configs)
	}
)

func readDeviceAccessPolicyAt(configDir, mac string) (bool, error) {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("firewall", true); err != nil {
		return true, err
	}
	sections, ok := tree.GetSections("firewall", "rule")
	if !ok {
		return true, errors.New("fetch firewall rules failed")
	}
	for _, section := range sections {
		name, _ := tree.GetLast("firewall", section, "name")
		sourceMAC, _ := tree.GetLast("firewall", section, "src_mac")
		target, _ := tree.GetLast("firewall", section, "target")
		if normalizeLanSpeedLimitedDeviceMAC(sourceMAC) == normalizeLanSpeedLimitedDeviceMAC(mac) {
			if _, recognized := buildBlockedDeviceRule(sourceMAC, name, target); recognized {
				return false, nil
			}
		}
	}
	return true, nil
}

func writeDeviceAccessPolicyAt(configDir, mac string, networkAccess bool) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("firewall", true); err != nil {
		return err
	}
	if err := mutateDeviceAccessPolicyTree(tree, mac, networkAccess); err != nil {
		return err
	}
	return tree.Commit()
}

func writeDeviceAccessPolicyBatchAt(configDir string, changes []groupBatchRestrictionChange) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("firewall", true); err != nil {
		return err
	}
	for _, change := range changes {
		if change.request == nil || change.request.Access == nil || change.current == nil {
			return errors.New("invalid access batch change")
		}
		if err := mutateDeviceAccessPolicyTree(tree, change.current.MAC, change.request.Access.NetworkAccess); err != nil {
			return err
		}
	}
	return tree.Commit()
}

func mutateDeviceAccessPolicyTree(tree uci.Tree, mac string, networkAccess bool) error {
	sections, _ := tree.GetSections("firewall", "rule")
	for _, section := range sections {
		name, _ := tree.GetLast("firewall", section, "name")
		sourceMAC, _ := tree.GetLast("firewall", section, "src_mac")
		target, _ := tree.GetLast("firewall", section, "target")
		if normalizeLanSpeedLimitedDeviceMAC(sourceMAC) == normalizeLanSpeedLimitedDeviceMAC(mac) {
			if _, recognized := buildBlockedDeviceRule(sourceMAC, name, target); recognized {
				tree.DelSection("firewall", section)
			}
		}
	}
	if !networkAccess {
		section := "quickstart_block_" + strings.ToLower(strings.ReplaceAll(normalizeLanSpeedLimitedDeviceMAC(mac), ":", ""))
		tree.DelSection("firewall", section)
		if err := tree.AddSection("firewall", section, "rule"); err != nil {
			return err
		}
		if !tree.Set("firewall", section, "name", buildBlockedRuleName(mac)) || !tree.Set("firewall", section, "src", "lan") ||
			!tree.Set("firewall", section, "dest", "wan") || !tree.Set("firewall", section, "target", "REJECT") ||
			!tree.Set("firewall", section, "proto", "all") || !tree.Set("firewall", section, "src_mac", normalizeLanSpeedLimitedDeviceMAC(mac)) {
			return errors.New("set access policy")
		}
	}
	return nil
}

func readDeviceSpeedPolicyAt(configDir, ip string) (*speedPolicySnapshot, error) {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("eqos", true); err != nil {
		return nil, err
	}
	sections, ok := tree.GetSections("eqos", "device")
	if !ok {
		return nil, errors.New("fetch eqos rules failed")
	}
	result := &speedPolicySnapshot{}
	for _, section := range sections {
		matchIP, _ := tree.GetLast("eqos", section, "ip")
		if matchIP != ip {
			continue
		}
		result.Section = section
		result.Upload, _ = parseSpeedValue(tree, section, "upload")
		result.Download, _ = parseSpeedValue(tree, section, "download")
		break
	}
	return result, nil
}

type speedPolicySnapshot struct {
	Section          string
	Upload, Download int64
}

func parseSpeedValue(tree uci.Tree, section, option string) (int64, error) {
	value, _ := tree.GetLast("eqos", section, option)
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func writeDeviceSpeedPolicyAt(configDir, ip, mac, comment string, enabled bool, upload, download int64) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("eqos", true); err != nil {
		return err
	}
	sections, _ := tree.GetSections("eqos", "device")
	matches := make([]SpeedLimitRuleMatch, 0, 1)
	for _, section := range sections {
		matchIP, _ := tree.GetLast("eqos", section, "ip")
		if matchIP == ip {
			matches = append(matches, SpeedLimitRuleMatch{Config: "eqos", SectionName: section, MatchIP: matchIP})
		}
	}
	action := "delete"
	if enabled {
		action = "modify"
	}
	plan := BuildSpeedLimitWritePlan(SpeedLimitWriteInput{Action: action, IP: ip, MAC: mac, NetworkAccess: true, UploadSpeed: upload, DownloadSpeed: download, Comment: comment}, matches, nil)
	return applySpeedLimitPlanAt(configDir, plan)
}

func writeDeviceSpeedPolicyBatchAt(configDir string, changes []groupBatchRestrictionChange) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("eqos", true); err != nil {
		return err
	}
	for _, change := range changes {
		if change.request == nil || change.request.Speed == nil || change.current == nil {
			return errors.New("invalid speed batch change")
		}
		sections, _ := tree.GetSections("eqos", "device")
		matches := make([]SpeedLimitRuleMatch, 0, 1)
		for _, section := range sections {
			matchIP, _ := tree.GetLast("eqos", section, "ip")
			if matchIP == change.current.CurrentIPv4 {
				matches = append(matches, SpeedLimitRuleMatch{Config: "eqos", SectionName: section, MatchIP: matchIP})
			}
		}
		action := "delete"
		if change.request.Speed.Enabled {
			action = "modify"
		}
		plan := BuildSpeedLimitWritePlan(SpeedLimitWriteInput{
			Action: action, IP: change.current.CurrentIPv4, MAC: change.current.MAC, NetworkAccess: true,
			UploadSpeed: change.request.Speed.UploadSpeed, DownloadSpeed: change.request.Speed.DownloadSpeed, Comment: change.current.DisplayName,
		}, matches, nil)
		if err := applySpeedLimitPlanToTree(tree, plan); err != nil {
			return err
		}
	}
	return tree.Commit()
}

func firewallCapabilityAt(configDir string) *modelsDeviceCapability {
	if _, err := os.Stat(filepath.Join(configDir, "firewall")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &modelsDeviceCapability{State: "unsupported", Reason: "firewall_unavailable"}
		}
		return &modelsDeviceCapability{State: "error", Reason: "status_unavailable"}
	}
	return &modelsDeviceCapability{State: "available"}
}

// Local alias keeps the storage helper independent from transport concerns.
type modelsDeviceCapability struct{ State, Reason string }

package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/utils"
)

var (
	deviceRestrictionConfigDir = func() string { return filepath.Dir(dhcpConfigPath) }
	deviceRestrictionApply     = func(ctx context.Context, configs []string) error {
		return NewDefaultLanSpeedLimitApply().Apply(ctx, configs)
	}
	deviceAccessRestrictionApply = func(ctx context.Context) error {
		return utils.BatchRun(ctx, []string{"uci commit firewall", "/etc/init.d/firewall restart"}, 0)
	}
)

func readDeviceAccessPolicyAt(ctx context.Context, configDir, mac string) (bool, error) {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("firewall", true); err != nil {
		return true, err
	}
	sections, ok := tree.GetSections("firewall", "rule")
	if !ok {
		return true, errors.New("fetch firewall rules failed")
	}
	for _, section := range sections {
		if owner := accessOwner(ctx); owner != "" {
			actual, _ := tree.GetLast("firewall", section, "quickstart_owner")
			if actual != owner {
				continue
			}
		}
		name, _ := tree.GetLast("firewall", section, "name")
		sourceMAC, _ := tree.GetLast("firewall", section, "src_mac")
		target, _ := tree.GetLast("firewall", section, "target")
		if _, recognized := buildBlockedDeviceRule(sourceMAC, name, target); !recognized || normalizeLanSpeedLimitedDeviceMAC(sourceMAC) != normalizeLanSpeedLimitedDeviceMAC(mac) {
			continue
		}
		scope := selectedLANScope(ctx)
		if scope != nil && scope.Device != "" {
			if accessRuleInScope(tree, section, scope) && !accessBindingCurrent(tree, section, scope) {
				return false, &PolicyError{Code: "access_binding_pending", Message: "正在恢复原上网限制，请稍后重试"}
			}
			device, _ := tree.GetLast("firewall", section, "device")
			source, _ := tree.GetLast("firewall", section, "src")
			if device == "" && !accessRuleInScope(tree, section, scope) && (source == "" || lanContainsAlias(scope.FirewallZones, source)) {
				return true, &PolicyError{Code: "access_binding_pending", Message: "正在恢复原上网限制，请稍后重试"}
			}
		}
		if accessRuleInScope(tree, section, selectedLANScope(ctx)) && normalizeLanSpeedLimitedDeviceMAC(sourceMAC) == normalizeLanSpeedLimitedDeviceMAC(mac) {
			if _, recognized := buildBlockedDeviceRule(sourceMAC, name, target); recognized {
				return false, nil
			}
		}
	}
	return true, nil
}

func writeDeviceAccessPolicyAt(ctx context.Context, configDir, mac string, networkAccess bool) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("firewall", true); err != nil {
		return err
	}
	if _, err := bindDeviceAccessRules(ctx, tree); err != nil {
		return err
	}
	if err := mutateDeviceAccessPolicyTree(ctx, tree, mac, networkAccess); err != nil {
		return err
	}
	if err := syncImmediateAccessRules(tree, immediateAccessIncludePath(configDir)); err != nil {
		return err
	}
	return tree.Commit()
}

func writeDeviceAccessPolicyBatchAt(ctx context.Context, configDir string, changes []groupBatchRestrictionChange) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("firewall", true); err != nil {
		return err
	}
	if _, err := bindDeviceAccessRules(ctx, tree); err != nil {
		return err
	}
	for _, change := range changes {
		if change.request == nil || change.request.Access == nil || change.current == nil {
			return errors.New("invalid access batch change")
		}
		if err := mutateDeviceAccessPolicyTree(context.WithValue(ctx, accessOwnerKey{}, change.accessOwner), tree, change.current.MAC, change.request.Access.NetworkAccess); err != nil {
			return err
		}
	}
	if err := syncImmediateAccessRules(tree, immediateAccessIncludePath(configDir)); err != nil {
		return err
	}
	return tree.Commit()
}

func immediateAccessIncludePath(configDir string) string {
	return filepath.Join(configDir, "quickstart-access.nft")
}

func syncImmediateAccessRules(tree uci.Tree, path string) error {
	const section = "quickstart_access_prepend"
	tree.DelSection("firewall", section)
	if err := tree.AddSection("firewall", section, "include"); err != nil {
		return err
	}
	if !tree.Set("firewall", section, "type", "nftables") || !tree.Set("firewall", section, "path", path) ||
		!tree.Set("firewall", section, "position", "chain-prepend") || !tree.Set("firewall", section, "chain", "forward") {
		return errors.New("register immediate access policy")
	}

	sections, _ := tree.GetSections("firewall", "rule")
	rules := make([]string, 0)
	seen := map[string]bool{}
	for _, ruleSection := range sections {
		name, _ := tree.GetLast("firewall", ruleSection, "name")
		sourceMAC, _ := tree.GetLast("firewall", ruleSection, "src_mac")
		target, _ := tree.GetLast("firewall", ruleSection, "target")
		if _, recognized := buildBlockedDeviceRule(sourceMAC, name, target); !recognized {
			continue
		}
		hardware, err := net.ParseMAC(normalizeLanSpeedLimitedDeviceMAC(sourceMAC))
		if err != nil || len(hardware) != 6 {
			continue
		}
		normalized := strings.ToLower(hardware.String())
		device, _ := tree.GetLast("firewall", ruleSection, "device")
		zone, _ := tree.GetLast("firewall", ruleSection, "dest")
		if zone == "" {
			zone = "wan"
		}
		selector := ""
		if device != "" {
			selector = fmt.Sprintf("iifname %q ", device)
		}
		chain := "reject_to_" + zone
		if !dhcpTagNamePattern.MatchString(chain) {
			return errors.New("invalid access destination zone")
		}
		rule := fmt.Sprintf("%sether saddr %s jump %s comment \"QuickStart pause %s\"\n", selector, normalized, chain, strings.ToUpper(normalized))
		if !seen[rule] {
			seen[rule] = true
			rules = append(rules, rule)
		}
	}
	sort.Strings(rules)
	content := strings.Join(rules, "")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temporary := path + ".tmp"
	if err := os.WriteFile(temporary, []byte(content), 0600); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

func mutateDeviceAccessPolicyTree(ctx context.Context, tree uci.Tree, mac string, networkAccess bool) error {
	sections, _ := tree.GetSections("firewall", "rule")
	for _, section := range sections {
		name, _ := tree.GetLast("firewall", section, "name")
		sourceMAC, _ := tree.GetLast("firewall", section, "src_mac")
		target, _ := tree.GetLast("firewall", section, "target")
		owner, _ := tree.GetLast("firewall", section, "quickstart_owner")
		if owner == accessOwner(ctx) && accessRuleInScope(tree, section, selectedLANScope(ctx)) && normalizeLanSpeedLimitedDeviceMAC(sourceMAC) == normalizeLanSpeedLimitedDeviceMAC(mac) {
			if _, recognized := buildBlockedDeviceRule(sourceMAC, name, target); recognized {
				tree.DelSection("firewall", section)
			}
		}
	}
	if !networkAccess {
		section := scopedDHCPSectionName(selectedLANScope(ctx), "quickstart_block_"+accessOwner(ctx)+strings.ToLower(strings.ReplaceAll(normalizeLanSpeedLimitedDeviceMAC(mac), ":", "")))
		tree.DelSection("firewall", section)
		if err := tree.AddSection("firewall", section, "rule"); err != nil {
			return err
		}
		if !tree.Set("firewall", section, "name", buildBlockedRuleName(mac)) || !tree.Set("firewall", section, "src", "lan") ||
			!tree.Set("firewall", section, "dest", "wan") || !tree.Set("firewall", section, "target", "REJECT") ||
			!tree.Set("firewall", section, "proto", "all") || !tree.Set("firewall", section, "src_mac", normalizeLanSpeedLimitedDeviceMAC(mac)) {
			return errors.New("set access policy")
		}
		if !tree.Set("firewall", section, "quickstart_access_version", "1") {
			return errors.New("set access rule version")
		}
		if err := bindAccessRule(tree, section, selectedLANScope(ctx)); err != nil {
			return err
		}
		if accessOwner(ctx) != "" && !tree.Set("firewall", section, "quickstart_owner", accessOwner(ctx)) {
			return errors.New("set access owner")
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

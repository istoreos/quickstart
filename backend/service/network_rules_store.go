package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

type systemNetworkRulesStore struct {
	inventory *DeviceInventoryModule
	policy    *DevicePolicyModule
	gateway   *GatewayPolicyModule
}
type networkRulesFileSnapshot struct {
	Path   string
	Data   []byte
	Mode   os.FileMode
	Exists bool
}

var networkRulesReloadAfterRestore = func(ctx context.Context) error {
	for _, item := range [][]string{{"/etc/init.d/dnsmasq", "reload"}, {"/etc/init.d/firewall", "reload"}, {"/etc/init.d/eqos", "restart"}} {
		if _, err := os.Stat(item[0]); errors.Is(err, os.ErrNotExist) {
			continue
		}
		if output, err := exec.CommandContext(ctx, item[0], item[1]).CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %s: %w", item[0], strings.TrimSpace(string(output)), err)
		}
	}
	return nil
}

var (
	networkRulesDeleteStatic = func(ctx context.Context, rule *models.NetworkRule) error {
		if err := clearNetworkRuleStaticAt(ctx, filepath.Dir(dhcpConfigPath), rule.MAC); err != nil {
			return err
		}
		if groupBatchDefersReload(ctx) {
			return nil
		}
		return reloadAndVerifyDnsmasq(ctx)
	}
	networkRulesDeleteSpeed = func(ctx context.Context, rule *models.NetworkRule) error {
		if groupBatchDefersReload(ctx) {
			return writeDeviceSpeedPolicyAt(deviceRestrictionConfigDir(), rule.IP, rule.MAC, "", false, 0, 0)
		}
		return NewDefaultLanSpeedLimitWriteService().UpsertSpeedLimitRule(ctx, SpeedLimitWriteInput{Action: "delete", IP: rule.IP, MAC: rule.MAC})
	}
	networkRulesClearRoute = func(ctx context.Context, rule *models.NetworkRule) error {
		if err := clearNetworkRuleRouteAt(ctx, filepath.Dir(dhcpConfigPath), rule.MAC); err != nil {
			return err
		}
		if groupBatchDefersReload(ctx) {
			return nil
		}
		return reloadAndVerifyDnsmasq(ctx)
	}
	networkRulesTakeSnapshots    = snapshotNetworkRuleFiles
	networkRulesRestoreSnapshots = restoreNetworkRuleFiles
)

func (store *systemNetworkRulesStore) Read(ctx context.Context) (networkRulesSnapshot, error) {
	deviceByMAC, deviceByIP := map[string]*models.DeviceInventoryItem{}, map[string]*models.DeviceInventoryItem{}
	if store.inventory != nil {
		if inventory, err := store.inventory.Snapshot(ctx); err == nil && inventory.Result != nil {
			for _, device := range inventory.Result.Devices {
				if device == nil {
					continue
				}
				if device.Mac != "" {
					deviceByMAC[normalizeInventoryMAC(device.Mac)] = device
				}
				for _, address := range device.Addresses.Current {
					if address != nil {
						deviceByIP[address.Address] = device
					}
				}
			}
		}
	}
	rules := make([]*models.NetworkRule, 0)
	policyRules, err := store.policy.ListRules(ctx)
	if err != nil {
		return networkRulesSnapshot{}, err
	}
	if policyRules.Result != nil {
		for _, item := range policyRules.Result.Static {
			if item == nil || (item.AssignedIP == "" && item.Hostname == "") {
				continue
			}
			device := deviceByMAC[normalizeInventoryMAC(item.AssignedMac)]
			rule := &models.NetworkRule{ID: stableNetworkRuleID("static", normalizeInventoryMAC(item.AssignedMac), item.AssignedIP, item.Hostname), Kind: "static", MAC: item.AssignedMac, IP: item.AssignedIP, Status: "active", Source: "device", Summary: staticRuleSummary(item)}
			attachNetworkRuleDevice(rule, device)
			rules = append(rules, rule)
		}
		for _, item := range policyRules.Result.Speed {
			if item == nil {
				continue
			}
			device := deviceByMAC[normalizeInventoryMAC(item.Mac)]
			if device == nil {
				device = deviceByIP[item.IP]
			}
			if !item.NetworkAccess {
				if _, system := store.policy.store.(*systemDevicePolicyStore); system {
					continue
				}
			}
			kind, summary := "speed", fmt.Sprintf("上传 %d · 下载 %d Mbit/s", item.UploadSpeed, item.DownloadSpeed)
			if !item.NetworkAccess {
				kind, summary = "access", "禁止联网"
			}
			rule := &models.NetworkRule{ID: stableNetworkRuleID(kind, normalizeInventoryMAC(item.Mac), item.IP), Kind: kind, MAC: item.Mac, IP: item.IP, Status: "active", Source: "device", Summary: summary}
			attachNetworkRuleDevice(rule, device)
			rules = append(rules, rule)
		}
	}
	if store.gateway != nil {
		state, stateErr := store.gateway.store.ReadState(ctx)
		if stateErr == nil {
			index := buildGatewayTargetIndex(state)
			for _, host := range state.Hosts {
				targetID := index.targetIDForTag(host.TagName)
				if targetID == "default" {
					continue
				}
				target := index.targets[targetID]
				summary := "未知路线"
				status := "unsupported"
				issueCode := "gateway_target_missing"
				nextAction := "restore_default_or_choose"
				if target != nil {
					summary = target.Public.Name
					if target.Public.Gateway != "" {
						summary += " · " + target.Public.Gateway
					}
					if state.UnreachableGateways[target.Public.Gateway] {
						status = "unreachable"
						issueCode = "gateway_unreachable"
						nextAction = "check_gateway_or_restore_default"
					} else if target.Public.Supported {
						status = "active"
						issueCode, nextAction = "", ""
					} else {
						issueCode = "gateway_target_unsupported"
						nextAction = "restore_default_or_edit"
					}
				}
				rule := &models.NetworkRule{ID: stableNetworkRuleID("route", normalizeInventoryMAC(host.MAC), targetID), Kind: "route", MAC: host.MAC, Status: status, Source: "device", Summary: summary, TargetID: targetID, IssueCode: issueCode, NextAction: nextAction}
				attachNetworkRuleDevice(rule, deviceByMAC[normalizeInventoryMAC(host.MAC)])
				rules = append(rules, rule)
			}
		}
	}
	health := &models.DeviceInventoryHealth{State: "ready"}
	extra, extraErr := store.scopedRestrictionRules(ctx, deviceByMAC)
	rules = append(rules, extra...)
	if extraErr != nil {
		health.State = "partial"
		health.Reasons = []string{"restriction_rules_unavailable"}
	}
	for _, rule := range rules {
		rule.NetworkID = selectedLAN(ctx)
	}
	sort.Slice(rules, func(i, j int) bool {
		if rules[i].Orphaned != rules[j].Orphaned {
			return !rules[i].Orphaned
		}
		if rules[i].DisplayName != rules[j].DisplayName {
			return rules[i].DisplayName < rules[j].DisplayName
		}
		if rules[i].Kind != rules[j].Kind {
			return rules[i].Kind < rules[j].Kind
		}
		return rules[i].ID < rules[j].ID
	})
	return networkRulesSnapshot{Rules: rules, Version: versionNetworkRules(rules), Health: health}, nil
}

func staticRuleSummary(item *models.LANStaticAssigned) string {
	parts := []string{}
	if item.AssignedIP != "" {
		parts = append(parts, item.AssignedIP)
	}
	if item.Hostname != "" {
		parts = append(parts, item.Hostname)
	}
	return strings.Join(parts, " · ")
}
func attachNetworkRuleDevice(rule *models.NetworkRule, device *models.DeviceInventoryItem) {
	if device == nil {
		rule.Orphaned = true
		rule.Status = "orphaned"
		rule.DisplayName = "离线或未知设备"
		return
	}
	rule.DeviceID = device.DeviceID
	rule.DisplayName = device.DisplayName
	if rule.MAC == "" {
		rule.MAC = device.Mac
	}
}

func (store *systemNetworkRulesStore) Apply(ctx context.Context, rules []*models.NetworkRule) error {
	ctx = context.WithValue(ctx, groupBatchContextKey{}, &groupBatchAccumulator{})
	paths := []string{filepath.Join(deviceRestrictionConfigDir(), "dhcp"), filepath.Join(deviceRestrictionConfigDir(), "eqos"), filepath.Join(deviceRestrictionConfigDir(), "firewall"), immediateAccessIncludePath(deviceRestrictionConfigDir())}
	snapshots, err := networkRulesTakeSnapshots(paths)
	if err != nil {
		return err
	}
	var nativeRestore func(context.Context) error
	rollback := func(cause error) error {
		if nativeRestore != nil {
			if err := nativeRestore(ctx); err != nil {
				return &networkRecoveryError{fmt.Errorf("native rollback failed: %w", err)}
			}
		}
		if err := networkRulesRestoreSnapshots(ctx, snapshots); err != nil {
			return &networkRecoveryError{fmt.Errorf("%v; rollback failed: %w", cause, err)}
		}
		return cause
	}
	nativeChanges := []groupBatchRestrictionChange{}
	for _, rule := range rules {
		if rule.Provider == nativePolicyProviderName {
			nativeChanges = append(nativeChanges, nativeRuleRemoval(rule.MAC))
		}
	}
	if len(nativeChanges) > 0 {
		provider := store.nativeProvider()
		if provider == nil {
			return errors.New("native provider unavailable")
		}
		nativeRestore, err = applyNativeSpeedBatch(ctx, nil, provider, nativeChanges)
		if err != nil {
			return err
		}
	}
	need := map[string]bool{}
	for _, rule := range rules {
		if rule.Provider == nativePolicyProviderName {
			continue
		}
		switch rule.Kind {
		case "static":
			err = networkRulesDeleteStatic(ctx, rule)
			need["dhcp"] = true
		case "route":
			err = networkRulesClearRoute(ctx, rule)
			need["dhcp"] = true
		case "speed":
			err = networkRulesDeleteSpeed(ctx, rule)
			need["eqos"] = true
		case "access":
			err = writeDeviceAccessPolicyAt(ctx, deviceRestrictionConfigDir(), rule.MAC, true)
			need["firewall"] = true
		default:
			err = errors.New("unsupported network rule kind")
		}
		if err != nil {
			return rollback(err)
		}
	}
	if need["dhcp"] {
		if err := reloadAndVerifyDnsmasq(ctx); err != nil {
			return rollback(err)
		}
	}
	if need["eqos"] {
		if err := deviceRestrictionApply(ctx, []string{"eqos"}); err != nil {
			return rollback(err)
		}
	}
	if need["firewall"] {
		if err := deviceAccessRestrictionApply(ctx); err != nil {
			return rollback(err)
		}
	}
	return nil
}

func clearNetworkRuleRouteAt(ctx context.Context, configDir, mac string) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return err
	}
	found := false
	sections, _ := tree.GetSections("dhcp", "host")
	for _, section := range sections {
		value, _ := tree.GetLast("dhcp", section, "mac")
		if normalizeInventoryMAC(value) != normalizeInventoryMAC(mac) || !scopedRuleDHCPHost(ctx, tree, section) {
			continue
		}
		tree.Del("dhcp", section, "tag")
		tree.Del("dhcp", section, "tag_title")
		found = true
	}
	if !found {
		return errors.New("gateway route rule no longer exists")
	}
	if selectedLANScope(ctx) != nil {
		if err := compileScopedDHCPHosts(tree); err != nil {
			return err
		}
	}
	return tree.Commit()
}

func clearNetworkRuleStaticAt(ctx context.Context, configDir, mac string) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return err
	}
	found := false
	sections, _ := tree.GetSections("dhcp", "host")
	for _, section := range sections {
		value, _ := tree.GetLast("dhcp", section, "mac")
		if normalizeInventoryMAC(value) != normalizeInventoryMAC(mac) || !scopedRuleDHCPHost(ctx, tree, section) {
			continue
		}
		tree.Del("dhcp", section, "ip")
		tree.Del("dhcp", section, "name")
		tree.Del("dhcp", section, "enabled")
		found = true
	}
	if !found {
		return errors.New("address reservation rule no longer exists")
	}
	if selectedLANScope(ctx) != nil {
		if err := compileScopedDHCPHosts(tree); err != nil {
			return err
		}
	}
	return tree.Commit()
}

func snapshotNetworkRuleFiles(paths []string) ([]networkRulesFileSnapshot, error) {
	result := make([]networkRulesFileSnapshot, 0, len(paths))
	for _, path := range paths {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			result = append(result, networkRulesFileSnapshot{Path: path})
			continue
		}
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		result = append(result, networkRulesFileSnapshot{Path: path, Data: data, Mode: info.Mode().Perm(), Exists: true})
	}
	return result, nil
}
func restoreNetworkRuleFiles(ctx context.Context, snapshots []networkRulesFileSnapshot) error {
	for _, snapshot := range snapshots {
		if snapshot.Exists {
			if err := os.WriteFile(snapshot.Path, snapshot.Data, snapshot.Mode); err != nil {
				return err
			}
		} else if err := os.Remove(snapshot.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return networkRulesReloadAfterRestore(ctx)
}

package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

func accessRuleInScope(tree uci.Tree, section string, scope *models.LANScope) bool {
	if scope == nil {
		scope = &models.LANScope{NetworkID: defaultLAN}
	}
	if network, _ := tree.GetLast("firewall", section, "quickstart_network"); network != "" {
		return network == scope.NetworkID
	}
	if device, _ := tree.GetLast("firewall", section, "device"); device != "" {
		return device == scope.Device
	}
	if scope.NetworkID != defaultLAN {
		return false
	}
	source, _ := tree.GetLast("firewall", section, "src")
	return len(scope.FirewallZones) == 0 || source == "" || lanContainsAlias(scope.FirewallZones, source)
}

func accessWANZone(tree uci.Tree) (string, error) {
	sections, _ := tree.GetSections("firewall", "zone")
	for _, section := range sections {
		name, _ := tree.GetLast("firewall", section, "name")
		networks, _ := tree.Get("firewall", section, "network")
		if name == "wan" || lanContainsAlias(networks, "wan") {
			return name, nil
		}
	}
	// A renamed uplink is resolved from the actual default route, not from the
	// selected LAN's identifier or its zone.
	uplink, err := outboundInterface()
	if err == nil && uplink != nil {
		for _, section := range sections {
			networks, _ := tree.Get("firewall", section, "network")
			if lanContainsAlias(networks, uplink.interfaceName) {
				name, _ := tree.GetLast("firewall", section, "name")
				return name, nil
			}
		}
	}
	return "", &PolicyError{Code: "access_scope_unavailable", Message: "无法确认当前的外网区域，暂不能修改上网限制"}
}

func bindAccessRule(tree uci.Tree, section string, scope *models.LANScope) error {
	if scope == nil {
		return nil
	}
	if !dnsmasqScopeTag.MatchString(scope.Device) || len(scope.FirewallZones) != 1 {
		return &PolicyError{Code: "access_scope_unavailable", Message: "此网络没有明确的防火墙接入区域"}
	}
	wan, err := accessWANZone(tree)
	if err != nil {
		return err
	}
	if !tree.Set("firewall", section, "quickstart_network", scope.NetworkID) ||
		!tree.Set("firewall", section, "src", scope.FirewallZones[0]) || !tree.Set("firewall", section, "dest", wan) ||
		!tree.Set("firewall", section, "device", scope.Device) || !tree.Set("firewall", section, "direction", "in") {
		return errors.New("set access network binding")
	}
	return nil
}

// Binding is part of a retained rule, independent of whether its access value
// changes. Resolve each logical network once and preserve owners/generations.
func bindDeviceAccessRules(ctx context.Context, tree uci.Tree) (bool, error) {
	selected := selectedLANScope(ctx)
	resolver, _ := ctx.Value(lanScopeResolverKey{}).(*LANScopeModule)
	if selected == nil && resolver == nil {
		return false, nil
	}
	scopes := map[string]*models.LANScope{}
	sections, _ := tree.GetSections("firewall", "rule")
	changed := false
	for _, section := range sections {
		name, _ := tree.GetLast("firewall", section, "name")
		mac, _ := tree.GetLast("firewall", section, "src_mac")
		target, _ := tree.GetLast("firewall", section, "target")
		if _, ok := buildBlockedDeviceRule(mac, name, target); !ok {
			continue
		}
		network, _ := tree.GetLast("firewall", section, "quickstart_network")
		device, _ := tree.GetLast("firewall", section, "device")
		legacy := network == "" && device == ""
		if network == "" {
			if device != "" {
				continue
			} // An old interface-only rule has no proven logical owner.
			network = defaultLAN
		}
		scope := scopes[network]
		if scope == nil {
			var err error
			if resolver != nil {
				scope, err = resolver.resolve(ctx, network, true)
			} else if selected != nil && selected.NetworkID == network {
				scope = selected
			}
			if err != nil || scope == nil {
				if !legacy && device != "" {
					continue
				} // Keep an isolated orphan; never reassign it.
				return false, &PolicyError{Code: "legacy_access_scope_unavailable", Message: "需先确认原局域网上网规则的归属"}
			}
			if selected != nil && selected.NetworkID == scope.NetworkID && selected.Version != scope.Version {
				return false, &PolicyError{Code: "network_changed", Message: "网络配置已改变，请重新预览"}
			}
			scopes[network] = scope
		}
		if scope.State == "down" || scope.ReadOnlyReason != "" {
			if !legacy {
				continue
			}
			return false, &PolicyError{Code: "access_scope_unavailable", Message: "原局域网暂不可管理，暂不能恢复原上网限制"}
		}
		source, _ := tree.GetLast("firewall", section, "src")
		if legacy && (len(scope.FirewallZones) != 1 || source != "" && !strings.EqualFold(source, scope.FirewallZones[0])) {
			return false, &PolicyError{Code: "legacy_access_scope_unavailable", Message: "原上网规则的区域与主局域网不一致，请先确认规则归属"}
		}
		if accessBindingCurrent(tree, section, scope) {
			continue
		}
		if err := bindAccessRule(tree, section, scope); err != nil {
			return false, err
		}
		changed = true
	}
	return changed, nil
}

func accessBindingCurrent(tree uci.Tree, section string, scope *models.LANScope) bool {
	if scope == nil || scope.Device == "" || len(scope.FirewallZones) != 1 {
		return false
	}
	wan, err := accessWANZone(tree)
	if err != nil {
		return false
	}
	for key, expected := range map[string]string{"quickstart_network": scope.NetworkID, "device": scope.Device, "src": scope.FirewallZones[0], "dest": wan, "direction": "in"} {
		actual, _ := tree.GetLast("firewall", section, key)
		if actual != expected {
			return false
		}
	}
	return true
}

// Empty ownership preserves the original explicit device/group pause rules.
type accessOwnerKey struct{}

func accessOwner(ctx context.Context) string {
	owner, _ := ctx.Value(accessOwnerKey{}).(string)
	return owner
}

// The pre-ownership rule is indistinguishable from an old manual pause.
// Keep it until an explicit access action replaces/removes it. A new manual
// rule has a generation marker and must not resurrect an old quota warning.
func legacyDeviceAccessRule(ctx context.Context, configDir, mac string) (bool, error) {
	if _, err := os.Stat(filepath.Join(configDir, "firewall")); os.IsNotExist(err) {
		return false, nil
	}
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("firewall", true); err != nil {
		return false, err
	}
	sections, _ := tree.GetSections("firewall", "rule")
	for _, section := range sections {
		sourceMAC, _ := tree.GetLast("firewall", section, "src_mac")
		if normalizeLanSpeedLimitedDeviceMAC(sourceMAC) != normalizeLanSpeedLimitedDeviceMAC(mac) || !accessRuleInScope(tree, section, selectedLANScope(ctx)) {
			continue
		}
		name, _ := tree.GetLast("firewall", section, "name")
		target, _ := tree.GetLast("firewall", section, "target")
		if _, recognized := buildBlockedDeviceRule(sourceMAC, name, target); !recognized {
			continue
		}
		owner, _ := tree.GetLast("firewall", section, "quickstart_owner")
		version, _ := tree.GetLast("firewall", section, "quickstart_access_version")
		if owner == "" && version != "1" {
			return true, nil
		}
	}
	return false, nil
}

// Physical legacy ownership does not depend on an inventory row or quota file.
func legacyAccessReviewForDevice(ctx context.Context, id string) (bool, error) {
	if !strings.HasPrefix(id, "mac:") {
		return false, nil
	}
	mac := normalizeInventoryMAC(strings.TrimPrefix(id, "mac:"))
	if mac == "" {
		return false, nil
	}
	return legacyDeviceAccessRule(ctx, deviceRestrictionConfigDir(), mac)
}

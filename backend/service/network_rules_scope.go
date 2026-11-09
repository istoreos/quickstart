package service

import (
	"context"
	"errors"
	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
	"os"
)

func scopedRuleDHCPHost(ctx context.Context, tree uci.Tree, section string) bool {
	if selectedLANScope(ctx) == nil {
		return true
	}
	_, err := scopedDHCPBinding(ctx, tree)
	if err != nil {
		return false
	}
	return dhcpHostInScope(tree, section, selectedLANScope(ctx))
}

func (store *systemNetworkRulesStore) nativeProvider() *nativeRateLimitProvider {
	if store.policy == nil {
		return nil
	}
	system, ok := store.policy.store.(*systemDevicePolicyStore)
	if !ok || system.rateLimit == nil {
		return nil
	}
	provider, _ := system.rateLimit.providers[nativePolicyProviderName].(*nativeRateLimitProvider)
	return provider
}

func (store *systemNetworkRulesStore) scopedRestrictionRules(ctx context.Context, devices map[string]*models.DeviceInventoryItem) ([]*models.NetworkRule, error) {
	rules := []*models.NetworkRule{}
	if store.policy != nil {
		if _, system := store.policy.store.(*systemDevicePolicyStore); !system {
			return rules, nil
		}
	}
	tree := uci.NewTree(deviceRestrictionConfigDir())
	var failure error
	if err := tree.LoadConfig("firewall", true); err != nil {
		failure = err
	} else {
		sections, _ := tree.GetSections("firewall", "rule")
		seen := map[string]bool{}
		for _, section := range sections {
			if !accessRuleInScope(tree, section, selectedLANScope(ctx)) {
				continue
			}
			mac, _ := tree.GetLast("firewall", section, "src_mac")
			name, _ := tree.GetLast("firewall", section, "name")
			target, _ := tree.GetLast("firewall", section, "target")
			if _, ok := buildBlockedDeviceRule(mac, name, target); !ok {
				continue
			}
			owner, _ := tree.GetLast("firewall", section, "quickstart_owner")
			source := "device"
			if owner == "quota" {
				source = "traffic_quota"
			}
			id := stableNetworkRuleID("access", normalizeInventoryMAC(mac), owner)
			if seen[id] {
				continue
			}
			seen[id] = true
			rule := &models.NetworkRule{ID: id, Kind: "access", MAC: mac, Status: "active", Source: source, Summary: "暂停上网"}
			if owner == "quota" {
				rule.NextAction = "edit_quota"
			}
			attachNetworkRuleDevice(rule, devices[normalizeInventoryMAC(mac)])
			rules = append(rules, rule)
		}
	}
	native := store.nativeProvider()
	if native == nil {
		return rules, failure
	}
	if native.registry != nil {
		if _, err := os.Stat(native.registry.configDir + "/quickstart-netpolicy"); errors.Is(err, os.ErrNotExist) {
			return rules, failure
		}
	}
	selected,err:=native.forContext(ctx);if err!=nil{return rules,err}
 if err:=selected.probe(ctx);err!=nil{return rules,err}
 snapshot, err := selected.snapshot(ctx)
	if err != nil {
		return rules, err
	}
	for _, policy := range snapshot.Policies {
		if policy.Owner != native.owner || policy.RateLimit == nil || !policy.RateLimit.Enabled {
			continue
		}
		mac := normalizeInventoryMAC(policy.Device.Value)
		rule := &models.NetworkRule{ID: stableNetworkRuleID("speed", mac, policy.ID), Kind: "speed", MAC: mac, Status: policy.State, Source: "device", Provider: nativePolicyProviderName, Summary: "设备限速"}
		attachNetworkRuleDevice(rule, devices[mac])
		rules = append(rules, rule)
	}
	return rules, failure
}

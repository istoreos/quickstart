package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

func scopedDHCPSectionName(scope *models.LANScope, legacy string) string {
	if scope == nil || scope.NetworkID == defaultLAN {
		return legacy
	}
	sum := sha256.Sum256([]byte(scope.NetworkID + "\x00" + legacy))
	return "qs_" + hex.EncodeToString(sum[:12])
}

// A DHCP section name is independent of the logical network name. Never create
// a missing binding while serving a read or silently choose between instances.
func scopedDHCPBinding(ctx context.Context, tree uci.Tree) (models.LANDHCPBinding, error) {
	aliases := map[string]string{}
	if scope := selectedLANScope(ctx); scope != nil {
		for _, binding := range scope.DHCPBindings {
			aliases[binding.Instance] = binding.RuntimeInstance
		}
	}
	bindings := lanDHCPBindings(tree, selectedLAN(ctx), aliases)
	if len(bindings) != 1 || !dhcpTreeHasSection(tree, "dnsmasq", bindings[0].Instance) {
		return models.LANDHCPBinding{}, &PolicyError{Code: "dhcp_binding_unavailable", Message: "当前网络没有唯一的 DHCP 服务配置"}
	}
	return bindings[0], nil
}

func scopedIPv4(ctx context.Context, tree uci.Tree) (string, netip.Prefix, error) {
	network := selectedLAN(ctx)
	addresses, _ := tree.Get("network", network, "ipaddr")
	mask, _ := tree.GetLast("network", network, "netmask")
	if len(addresses) != 1 {
		return "", netip.Prefix{}, errors.New("LAN must have one unambiguous IPv4 address")
	}
	raw := addresses[0]
	if prefix, err := netip.ParsePrefix(raw); err == nil && prefix.Addr().Is4() {
		return prefix.Addr().String(), prefix.Masked(), nil
	}
	prefix, err := prefixFromAddressAndMask(raw, mask)
	return raw, prefix, err
}

// Explicit metadata identifies Quickstart records; match_tag identifies existing
// OpenWrt records. Untagged legacy records retain their original default LAN.
func dhcpHostInScope(tree uci.Tree, section string, scope *models.LANScope) bool {
	if scope == nil {
		return true
	}
	if network, ok := tree.GetLast("dhcp", section, "quickstart_network"); ok {
		return network == scope.NetworkID
	}
	tags, _ := tree.Get("dhcp", section, "match_tag")
	if len(tags) > 0 {
		return len(tags) == 1 && strings.TrimPrefix(tags[0], "tag:") == scope.Device
	}
	if ip, ok := tree.GetLast("dhcp", section, "ip"); ok {
		if address, err := netip.ParseAddr(ip); err == nil && address.Is4() {
			return addressInPrefixes(address, scopeLANPrefixes(scope))
		}
	}
	return scope.NetworkID == defaultLAN
}

func validateScopedReservation(input StaticAssignmentWriteInput, state gatewayPolicySnapshot) *models.DevicePolicyError {
	if !input.BindIP || state.Scope == nil {
		return nil
	}
	address, err := netip.ParseAddr(input.AssignedIP)
	valid := false
	for _, prefix := range state.Prefixes {
		if !prefix.Addr().Is4() || !prefix.Contains(address) {
			continue
		}
		base := uint64(ipv4Ordinal(prefix.Masked().Addr()))
		broadcast := base + (uint64(1) << uint(32-prefix.Bits())) - 1
		value := uint64(ipv4Ordinal(address))
		valid = value > base && value < broadcast
		if valid {
			break
		}
	}
	if err != nil || !address.Is4() || !valid {
		return &models.DevicePolicyError{Code: "validation_failed", Message: "reserved IPv4 address must be a usable address in the selected LAN"}
	}
	if address.String() == state.LAN.LanAddr {
		return &models.DevicePolicyError{Code: "address_conflict", Message: "reserved IPv4 address belongs to the router"}
	}
	for _, target := range buildGatewayTargetIndex(state).targets {
		if target.Public != nil && target.Public.Gateway == address.String() {
			return &models.DevicePolicyError{Code: "address_conflict", Message: "reserved IPv4 address belongs to an Internet Path gateway"}
		}
	}
	if state.DHCP != nil && state.DHCP.FloatIP != nil && (state.DHCP.FloatIP.SetIP == address.String() || state.DHCP.FloatIP.CheckIP == address.String()) {
		return &models.DevicePolicyError{Code: "address_conflict", Message: "reserved IPv4 address belongs to a floating gateway"}
	}
	return nil
}

func scopedDHCPLeaseCount(path string, prefix netip.Prefix) int64 {
	leases, err := readDeviceInventoryDHCPv4(path, time.Now())
	if err != nil {
		return 0
	}
	seen := map[string]bool{}
	for _, lease := range leases {
		address, err := netip.ParseAddr(lease.IPv4)
		if err == nil && prefix.Contains(address) {
			seen[strings.ToLower(lease.MAC)] = true
		}
	}
	return int64(len(seen))
}

func sharedDHCPNetworks(tree uci.Tree, selected models.LANDHCPBinding) []string {
	networks := []string{}
	sections, _ := tree.GetSections("dhcp", "dhcp")
	for _, section := range sections {
		network, _ := tree.GetLast("dhcp", section, "interface")
		if network == "" {
			network = section
		}
		for _, binding := range lanDHCPBindings(tree, network, map[string]string{selected.Instance: selected.RuntimeInstance}) {
			if binding.Instance == selected.Instance {
				networks = append(networks, network)
				break
			}
		}
	}
	return uniqueSortedNonEmpty(networks)
}

func setDHCPGatewayOptions(tree uci.Tree, section, gateway, routerAddress string) error {
	options, _ := tree.Get("dhcp", section, "dhcp_option")
	kept := make([]string, 0, len(options)+2)
	for _, option := range options {
		code := strings.TrimSpace(strings.SplitN(option, ",", 2)[0])
		if code != "3" && code != "6" && code != "option:router" && code != "option:dns-server" {
			kept = append(kept, option)
		}
	}
	if gateway != "" && gateway != routerAddress {
		kept = append(kept, gatewayAndDNSOptions(gateway)...)
	}
	tree.Del("dhcp", section, "dhcp_option")
	if len(kept) > 0 && !tree.SetType("dhcp", section, "dhcp_option", uci.TypeList, kept...) {
		return errors.New("set default Internet Path")
	}
	return nil
}

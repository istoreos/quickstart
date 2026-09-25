package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/digineo/go-uci"
)

type defaultGatewayPolicyStore struct {
	lanStatus LanStatusReader
	dhcp      DhcpConfigStore
}

var (
	gatewayPolicyTakeSnapshot = snapshotDhcpConfig
	gatewayPolicyMutate       = func(plan gatewayPolicyExecutionPlan) error {
		return mutateGatewayPolicyConfigAt(filepath.Dir(dhcpConfigPath), plan)
	}
	gatewayPolicyReload  = reloadAndVerifyDnsmasq
	gatewayPolicyRestore = restoreDhcpConfig
)

func NewDefaultGatewayPolicyStore() gatewayPolicyStore {
	return &defaultGatewayPolicyStore{lanStatus: NewDefaultLanStatusReader(), dhcp: NewDefaultDhcpConfigStore()}
}

func (store *defaultGatewayPolicyStore) ReadState(ctx context.Context) (gatewayPolicySnapshot, error) {
	lan, err := store.lanStatus.ReadLanStatus(ctx)
	if err != nil {
		return gatewayPolicySnapshot{}, err
	}
	dhcp, err := store.dhcp.LoadLanState(ctx)
	if err != nil {
		return gatewayPolicySnapshot{}, err
	}
	if err := uci.LoadConfig("dhcp", true); err != nil {
		return gatewayPolicySnapshot{}, err
	}
	sections, _ := uci.GetSections("dhcp", "host")
	hosts := make([]gatewayPolicyHost, 0, len(sections))
	for _, section := range sections {
		mac, _ := uci.GetLast("dhcp", section, "mac")
		tag, _ := uci.GetLast("dhcp", section, "tag")
		hosts = append(hosts, gatewayPolicyHost{SectionName: section, MAC: mac, TagName: tag})
	}
	data, err := os.ReadFile(dhcpConfigPath)
	if err != nil {
		return gatewayPolicySnapshot{}, err
	}
	prefixes, prefixErr := readSystemLANPrefixes()
	if prefixErr != nil {
		prefixes = nil
	}
	sum := sha256.Sum256(data)
	return gatewayPolicySnapshot{
		LAN: lan, DHCP: dhcp, Hosts: hosts, Prefixes: prefixes, Version: hex.EncodeToString(sum[:]),
	}, nil
}

func (store *defaultGatewayPolicyStore) Apply(ctx context.Context, plan gatewayPolicyExecutionPlan) error {
	if plan.Public == nil || plan.Target == nil || len(plan.Public.Changes) == 0 {
		return nil
	}
	if plan.Public.Action == "assign" {
		if err := preflightGatewayAssignment(ctx, plan); err != nil {
			return fmt.Errorf("dnsmasq preflight failed: %w", err)
		}
	}
	snapshot, err := gatewayPolicyTakeSnapshot()
	if err != nil {
		return fmt.Errorf("snapshot DHCP configuration: %w", err)
	}
	if err := gatewayPolicyMutate(plan); err != nil {
		return rollbackGatewayPolicy(ctx, snapshot, fmt.Errorf("write gateway policy: %w", err))
	}
	if err := gatewayPolicyReload(ctx); err != nil {
		return rollbackGatewayPolicy(ctx, snapshot, fmt.Errorf("reload dnsmasq: %w", err))
	}
	return nil
}

func rollbackGatewayPolicy(ctx context.Context, snapshot dhcpConfigSnapshot, applyErr error) error {
	if err := gatewayPolicyRestore(ctx, snapshot); err != nil {
		return fmt.Errorf("%v; rollback failed: %w", applyErr, err)
	}
	return applyErr
}

func preflightGatewayAssignment(ctx context.Context, plan gatewayPolicyExecutionPlan) error {
	input := StaticAssignmentWriteInput{Action: "add", AssignedMAC: plan.MAC}
	if plan.Target != nil {
		input.TagName = plan.Target.TagName
		input.TagTitle = plan.Target.TagTitle
	}
	normalized, err := normalizeStaticAssignmentInput(input)
	if err != nil {
		return err
	}
	if plan.Target != nil {
		for _, option := range plan.Target.Options {
			parts := splitDhcpOption(option)
			if len(parts) != 2 || (parts[0] != "3" && parts[0] != "6") {
				return errors.New("unsupported DHCP option")
			}
			if _, parseErr := netipParseIPv4(parts[1]); parseErr != nil {
				return parseErr
			}
		}
	}
	return preflightStaticAssignment(ctx, normalized)
}

func netipParseIPv4(raw string) (string, error) {
	address, err := netipParseAddr(raw)
	if err != nil || !address.Is4() {
		return "", errors.New("gateway DHCP option is not a valid IPv4 address")
	}
	return address.String(), nil
}

var netipParseAddr = func(raw string) (netip.Addr, error) { return netip.ParseAddr(raw) }

func mutateGatewayPolicyConfigAt(configDir string, plan gatewayPolicyExecutionPlan) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return err
	}
	if plan.Public.Action == "delete_target" {
		if plan.Target.TagName == "" {
			return errors.New("built-in gateway target cannot be deleted")
		}
		tree.DelSection("dhcp", plan.Target.TagName)
		return tree.Commit()
	}
	if plan.Public.Action != "assign" {
		return errors.New("unsupported gateway policy action")
	}
	if plan.Target.TagName != "" && plan.Target.Materialize {
		if !dhcpTreeHasSection(tree, "tag", plan.Target.TagName) {
			if err := tree.AddSection("dhcp", plan.Target.TagName, "tag"); err != nil {
				return err
			}
		}
		if plan.Target.TagTitle != "" && !tree.Set("dhcp", plan.Target.TagName, "tag_title", plan.Target.TagTitle) {
			return errors.New("set gateway target title")
		}
		if !tree.Set("dhcp", plan.Target.TagName, "AutoCreated", "1") {
			return errors.New("set gateway target ownership")
		}
		if len(plan.Target.Options) > 0 && !tree.SetType("dhcp", plan.Target.TagName, "dhcp_option", uci.TypeList, plan.Target.Options...) {
			return errors.New("set gateway target DHCP options")
		}
	}
	hostSection := ""
	sections, _ := tree.GetSections("dhcp", "host")
	for _, section := range sections {
		mac, _ := tree.GetLast("dhcp", section, "mac")
		if normalizeInventoryMAC(mac) == normalizeInventoryMAC(plan.MAC) {
			hostSection = section
			break
		}
	}
	if hostSection == "" {
		hostSection = "quickstart_" + strings.ToLower(strings.ReplaceAll(plan.MAC, ":", ""))
		tree.DelSection("dhcp", hostSection)
		if err := tree.AddSection("dhcp", hostSection, "host"); err != nil {
			return err
		}
		if !tree.Set("dhcp", hostSection, "mac", strings.ToUpper(plan.MAC)) {
			return errors.New("set gateway assignment MAC")
		}
	}
	if plan.Target.TagName == "" {
		tree.Del("dhcp", hostSection, "tag")
		tree.Del("dhcp", hostSection, "tag_title")
	} else {
		if !tree.Set("dhcp", hostSection, "tag", plan.Target.TagName) {
			return errors.New("set gateway assignment")
		}
		if plan.Target.TagTitle != "" && !tree.Set("dhcp", hostSection, "tag_title", plan.Target.TagTitle) {
			return errors.New("set gateway assignment title")
		}
	}
	return tree.Commit()
}

func dhcpTreeHasSection(tree uci.Tree, sectionType, sectionName string) bool {
	sections, _ := tree.GetSections("dhcp", sectionType)
	for _, section := range sections {
		if section == sectionName {
			return true
		}
	}
	return false
}

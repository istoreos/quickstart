package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/digineo/go-uci"
)

type defaultDeviceNetworkPolicyWriter struct{}

type networkPolicyApplyError struct {
	Stage  string
	Status string
	Err    error
}

func (failure *networkPolicyApplyError) Error() string { return failure.Err.Error() }

var (
	deviceNetworkPolicyPreflight = preflightStaticAssignment
	deviceNetworkPolicySnapshot  = snapshotDhcpConfig
	deviceNetworkPolicyMutate    = func(input StaticAssignmentWriteInput, route gatewayPolicyExecutionPlan) error {
		return mutateDeviceNetworkPolicyConfigAt(filepath.Dir(dhcpConfigPath), input, route)
	}
	deviceNetworkPolicyReload  = reloadAndVerifyDnsmasq
	deviceNetworkPolicyRestore = restoreDhcpConfig
)

func (writer *defaultDeviceNetworkPolicyWriter) Apply(ctx context.Context, input StaticAssignmentWriteInput, route gatewayPolicyExecutionPlan) error {
	if route.Target == nil {
		return &networkPolicyApplyError{Stage: "validate", Status: "failed", Err: errors.New("internet path target is unavailable")}
	}
	preflight := input
	preflight.TagName = route.Target.TagName
	preflight.TagTitle = route.Target.TagTitle
	if err := deviceNetworkPolicyPreflight(ctx, preflight); err != nil {
		return &networkPolicyApplyError{Stage: "validate", Status: "failed", Err: fmt.Errorf("dnsmasq preflight failed: %w", err)}
	}
	if accumulator := groupBatchAccumulatorFrom(ctx); accumulator != nil {
		accumulator.network = append(accumulator.network, groupBatchNetworkChange{input: input, route: route})
		return nil
	}
	snapshot, err := deviceNetworkPolicySnapshot()
	if err != nil {
		return &networkPolicyApplyError{Stage: "snapshot", Status: "failed", Err: fmt.Errorf("snapshot DHCP configuration: %w", err)}
	}
	if err := deviceNetworkPolicyMutate(input, route); err != nil {
		return rollbackDeviceNetworkPolicy(ctx, snapshot, "apply", fmt.Errorf("write device network policy: %w", err))
	}
	if groupBatchDefersReload(ctx) {
		return nil
	}
	if err := deviceNetworkPolicyReload(ctx); err != nil {
		return rollbackDeviceNetworkPolicy(ctx, snapshot, "verify", fmt.Errorf("reload dnsmasq: %w", err))
	}
	return nil
}

func rollbackDeviceNetworkPolicy(ctx context.Context, snapshot dhcpConfigSnapshot, stage string, applyErr error) error {
	if err := deviceNetworkPolicyRestore(ctx, snapshot); err != nil {
		return &networkPolicyApplyError{Stage: "recover", Status: "recovery_required", Err: fmt.Errorf("%v; rollback failed: %w", applyErr, err)}
	}
	return &networkPolicyApplyError{Stage: stage, Status: "rolled_back", Err: applyErr}
}

func mutateDeviceNetworkPolicyConfigAt(configDir string, input StaticAssignmentWriteInput, route gatewayPolicyExecutionPlan) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return err
	}
	if err := mutateDeviceNetworkPolicyTree(tree, input, route); err != nil {
		return err
	}
	return tree.Commit()
}

func mutateDeviceNetworkPoliciesConfigAt(configDir string, changes []groupBatchNetworkChange) error {
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return err
	}
	for _, change := range changes {
		if err := mutateDeviceNetworkPolicyTree(tree, change.input, change.route); err != nil {
			return err
		}
	}
	return tree.Commit()
}

func mutateDeviceNetworkPolicyTree(tree uci.Tree, input StaticAssignmentWriteInput, route gatewayPolicyExecutionPlan) error {
	if route.Target.Materialize && route.Target.TagName != "" {
		if !dhcpTreeHasSection(tree, "tag", route.Target.TagName) {
			if err := tree.AddSection("dhcp", route.Target.TagName, "tag"); err != nil {
				return err
			}
		}
		if route.Target.TagTitle != "" && !tree.Set("dhcp", route.Target.TagName, "tag_title", route.Target.TagTitle) {
			return errors.New("set gateway target title")
		}
		if !tree.Set("dhcp", route.Target.TagName, "AutoCreated", "1") {
			return errors.New("set gateway target ownership")
		}
		if len(route.Target.Options) > 0 && !tree.SetType("dhcp", route.Target.TagName, "dhcp_option", uci.TypeList, route.Target.Options...) {
			return errors.New("set gateway target options")
		}
	}
	hostSections := make([]string, 0, 1)
	sections, _ := tree.GetSections("dhcp", "host")
	for _, section := range sections {
		mac, _ := tree.GetLast("dhcp", section, "mac")
		if normalizeInventoryMAC(mac) == normalizeInventoryMAC(input.AssignedMAC) {
			hostSections = append(hostSections, section)
		}
	}
	section := ""
	if len(hostSections) > 0 {
		section = hostSections[0]
		for _, duplicate := range hostSections[1:] {
			tree.DelSection("dhcp", duplicate)
		}
	}
	needsHost := input.BindIP || input.Hostname != "" || route.Target.TagName != ""
	if !needsHost {
		if section != "" {
			tree.DelSection("dhcp", section)
		}
		return nil
	}
	if section == "" {
		section = "quickstart_" + strings.ToLower(strings.ReplaceAll(input.AssignedMAC, ":", ""))
		tree.DelSection("dhcp", section)
		if err := tree.AddSection("dhcp", section, "host"); err != nil {
			return err
		}
	}
	for _, option := range []string{"enabled", "mac", "ip", "name", "tag", "tag_title"} {
		tree.Del("dhcp", section, option)
	}
	if !tree.Set("dhcp", section, "enabled", "1") || !tree.Set("dhcp", section, "mac", strings.ToUpper(input.AssignedMAC)) {
		return errors.New("set device identity")
	}
	if input.BindIP && !tree.Set("dhcp", section, "ip", input.AssignedIP) {
		return errors.New("set reserved IPv4 address")
	}
	if input.Hostname != "" && !tree.Set("dhcp", section, "name", input.Hostname) {
		return errors.New("set DHCP hostname")
	}
	if route.Target.TagName != "" {
		if !tree.Set("dhcp", section, "tag", route.Target.TagName) {
			return errors.New("set internet path")
		}
		if route.Target.TagTitle != "" && !tree.Set("dhcp", section, "tag_title", route.Target.TagTitle) {
			return errors.New("set internet path title")
		}
	}
	return nil
}

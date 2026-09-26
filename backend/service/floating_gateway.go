package service

import (
	"context"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/istoreos/quickstart/backend/models"
)

type floatingGatewaySnapshot struct {
	Installed   bool
	Config      *models.FloatingGatewayConfig
	Prefixes    []netip.Prefix
	UsedIPs     map[string]bool
	Running     bool
	LocalHolder bool
	PeerHolder  bool
	Version     string
}

type floatingGatewayStore interface {
	Read(context.Context) (floatingGatewaySnapshot, error)
	Apply(context.Context, *models.FloatingGatewayConfig) error
}

type FloatingGatewayModule struct {
	mu      sync.Mutex
	store   floatingGatewayStore
	gateway *GatewayPolicyModule
}

func NewFloatingGatewayModule(store floatingGatewayStore, gateway *GatewayPolicyModule) *FloatingGatewayModule {
	return &FloatingGatewayModule{store: store, gateway: gateway}
}

func NewDefaultFloatingGatewayModule(gateway *GatewayPolicyModule) *FloatingGatewayModule {
	var inventory *DeviceInventoryModule
	if gateway != nil {
		inventory = gateway.inventory
	}
	return NewFloatingGatewayModule(&systemFloatingGatewayStore{inventory: inventory}, gateway)
}

func (module *FloatingGatewayModule) Get(ctx context.Context) (*models.FloatingGatewayResponse, error) {
	state, err := module.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	return floatingGatewayResponse(state, nil, false), nil
}

func (module *FloatingGatewayModule) Plan(ctx context.Context, request *models.FloatingGatewayApplyRequest) (*models.FloatingGatewayResponse, error) {
	state, plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	return floatingGatewayResponse(state, plan, false), nil
}

func (module *FloatingGatewayModule) Apply(ctx context.Context, request *models.FloatingGatewayApplyRequest) (*models.FloatingGatewayResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	state, plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	if plan.Error != nil || !plan.CanApply {
		return floatingGatewayResponse(state, plan, false), nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != state.Version {
		plan.Error = &models.DevicePolicyError{Code: "conflict", Message: "floating gateway configuration changed; review the plan again"}
		plan.CanApply = false
		return floatingGatewayResponse(state, plan, false), nil
	}
	if !plan.Changed {
		return floatingGatewayResponse(state, plan, false), nil
	}
	if err := module.store.Apply(ctx, normalizedFloatingConfig(request.Config)); err != nil {
		plan.Error = &models.DevicePolicyError{Code: "apply_failed", Message: err.Error()}
		return floatingGatewayResponse(state, plan, false), nil
	}
	updated, err := module.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	return floatingGatewayResponse(updated, plan, true), nil
}

func (module *FloatingGatewayModule) DrillPlan(ctx context.Context) (*models.FloatingGatewayDrillResponse, error) {
	state, err := module.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	plan := &models.FloatingGatewayDrillPlan{
		Steps:          []string{"记录当前持有者与配置快照", "显式停止优先服务节点", "观察接管节点获得虚拟 IP", "恢复优先服务节点并观察回切策略"},
		StopConditions: []string{"管理连接中断", "两台节点同时持有虚拟 IP", "外网检测连续失败"},
		Recovery:       []string{"恢复配置快照", "启动两端 floatip 服务", "确认虚拟 IP 只有一个持有者"},
	}
	if !state.Installed || !state.Config.Enabled {
		plan.Error = &models.DevicePolicyError{Code: "dependency_not_installed", Message: "install and enable floating gateway on both nodes before a drill"}
	} else {
		plan.CanRun = false
		plan.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "pair verification is required before running a disruptive drill"}
	}
	return &models.FloatingGatewayDrillResponse{Result: plan}, nil
}

func (module *FloatingGatewayModule) plan(ctx context.Context, request *models.FloatingGatewayApplyRequest) (floatingGatewaySnapshot, *models.FloatingGatewayPlan, error) {
	state, err := module.store.Read(ctx)
	if err != nil {
		return state, nil, err
	}
	if state.Config == nil {
		state.Config = &models.FloatingGatewayConfig{PeerIPs: []string{}, HealthTimeoutSec: 5}
	}
	plan := &models.FloatingGatewayPlan{Changes: []string{}, Warnings: []string{}, AffectedDevices: []*models.GatewayReference{}, Version: state.Version}
	if request == nil || request.Config == nil {
		plan.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "config is required"}
		return state, plan, nil
	}
	config := normalizedFloatingConfig(request.Config)
	if !state.Installed {
		plan.Error = &models.DevicePolicyError{Code: "dependency_not_installed", Message: "floating gateway component is not installed"}
		return state, plan, nil
	}
	if validation := validateFloatingGatewayConfig(config, state); validation != nil {
		plan.Error = validation
		return state, plan, nil
	}
	plan.Changed = !floatingGatewayConfigsEqual(state.Config, config)
	if state.Config.Enabled != config.Enabled {
		plan.Changes = append(plan.Changes, "change_enabled_state")
	}
	if state.Config.Role != config.Role {
		plan.Changes = append(plan.Changes, "change_node_responsibility")
	}
	if state.Config.VirtualIP != config.VirtualIP {
		plan.Changes = append(plan.Changes, "change_virtual_ip")
	}
	if strings.Join(state.Config.PeerIPs, ",") != strings.Join(config.PeerIPs, ",") {
		plan.Changes = append(plan.Changes, "change_peer_nodes")
	}
	if state.Config.HealthURL != config.HealthURL || state.Config.HealthTimeoutSec != config.HealthTimeoutSec {
		plan.Changes = append(plan.Changes, "change_health_check")
	}
	if state.Config.Enabled && (!config.Enabled || state.Config.VirtualIP != config.VirtualIP) {
		plan.AffectedDevices = module.references(ctx, state.Config.VirtualIP)
		if len(plan.AffectedDevices) > 0 {
			plan.Error = &models.DevicePolicyError{Code: "conflict", Message: "migrate devices that use this floating gateway before disabling or changing its address"}
			return state, plan, nil
		}
	}
	plan.CanApply = true
	plan.RequiresDrill = config.Enabled && plan.Changed
	if plan.RequiresDrill {
		plan.Warnings = append(plan.Warnings, "run_a_controlled_failover_drill_after_pairing")
	}
	return state, plan, nil
}

func (module *FloatingGatewayModule) references(ctx context.Context, ip string) []*models.GatewayReference {
	if module.gateway == nil || ip == "" {
		return []*models.GatewayReference{}
	}
	targets, err := module.gateway.ListTargets(ctx)
	if err != nil || targets.Result == nil {
		return []*models.GatewayReference{}
	}
	for _, target := range targets.Result.Targets {
		if target.Kind != "floating" || target.Gateway != ip {
			continue
		}
		refs, refErr := module.gateway.References(ctx, target.ID)
		if refErr == nil && refs.Result != nil {
			return refs.Result.References
		}
	}
	return []*models.GatewayReference{}
}

func validateFloatingGatewayConfig(config *models.FloatingGatewayConfig, state floatingGatewaySnapshot) *models.DevicePolicyError {
	if !config.Enabled {
		return nil
	}
	if config.Role != "preferred" && config.Role != "takeover" {
		return &models.DevicePolicyError{Code: "validation_failed", Message: "choose preferred service node or failover takeover node"}
	}
	virtual, err := netip.ParseAddr(config.VirtualIP)
	if err != nil || !virtual.Is4() {
		return &models.DevicePolicyError{Code: "validation_failed", Message: "virtual gateway must be a valid IPv4 address"}
	}
	inLAN := false
	for _, prefix := range state.Prefixes {
		if prefix.Contains(virtual) {
			inLAN = true
			break
		}
	}
	if !inLAN {
		return &models.DevicePolicyError{Code: "validation_failed", Message: "virtual gateway must be in a LAN subnet"}
	}
	// Reusing the configured address is only safe while this node is already
	// serving it.  When the service is disabled, an inventory hit means another
	// LAN client currently owns the address and enabling would create a clash.
	if state.UsedIPs[virtual.String()] && (!state.Config.Enabled || state.Config.VirtualIP != virtual.String()) {
		return &models.DevicePolicyError{Code: "conflict", Message: "virtual gateway address is already in use"}
	}
	if config.Role == "takeover" && len(config.PeerIPs) == 0 {
		return &models.DevicePolicyError{Code: "validation_failed", Message: "failover takeover node requires the preferred node address"}
	}
	for _, raw := range config.PeerIPs {
		peer, parseErr := netip.ParseAddr(raw)
		if parseErr != nil || !peer.Is4() || peer == virtual {
			return &models.DevicePolicyError{Code: "validation_failed", Message: "peer node address is invalid"}
		}
		peerInLAN := false
		for _, prefix := range state.Prefixes {
			if prefix.Contains(peer) {
				peerInLAN = true
				break
			}
		}
		if !peerInLAN {
			return &models.DevicePolicyError{Code: "validation_failed", Message: "peer node must be in a LAN subnet"}
		}
	}
	if config.HealthURL != "" {
		parsed, parseErr := url.ParseRequestURI(config.HealthURL)
		if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || strings.ContainsRune(config.HealthURL, '\'') || !validSafeLabel(config.HealthURL, 2048) {
			return &models.DevicePolicyError{Code: "validation_failed", Message: "health URL must be an absolute HTTP or HTTPS URL"}
		}
	}
	if config.HealthTimeoutSec < 1 || config.HealthTimeoutSec > 30 {
		return &models.DevicePolicyError{Code: "validation_failed", Message: "health timeout must be between 1 and 30 seconds"}
	}
	return nil
}

func normalizedFloatingConfig(config *models.FloatingGatewayConfig) *models.FloatingGatewayConfig {
	if config == nil {
		return &models.FloatingGatewayConfig{PeerIPs: []string{}}
	}
	copyValue := *config
	copyValue.VirtualIP = normalizeFloatingAddress(copyValue.VirtualIP)
	copyValue.HealthURL = strings.TrimSpace(copyValue.HealthURL)
	if copyValue.HealthTimeoutSec == 0 {
		copyValue.HealthTimeoutSec = 5
	}
	peers := make([]string, 0, len(copyValue.PeerIPs))
	seen := map[string]bool{}
	for _, peer := range copyValue.PeerIPs {
		peer = strings.TrimSpace(peer)
		if peer != "" && !seen[peer] {
			seen[peer] = true
			peers = append(peers, peer)
		}
	}
	sort.Strings(peers)
	copyValue.PeerIPs = peers
	return &copyValue
}

// normalizeFloatingAddress keeps the public/domain model address-only even
// when an existing floatip UCI config uses the supported address/prefix form.
// floatip derives the LAN prefix when a plain address is written back.
func normalizeFloatingAddress(raw string) string {
	raw = strings.TrimSpace(raw)
	if prefix, err := netip.ParsePrefix(raw); err == nil {
		return prefix.Addr().String()
	}
	return raw
}

func floatingGatewayConfigsEqual(left, right *models.FloatingGatewayConfig) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Enabled == right.Enabled && left.Role == right.Role && left.VirtualIP == right.VirtualIP && strings.Join(left.PeerIPs, ",") == strings.Join(right.PeerIPs, ",") && left.HealthURL == right.HealthURL && left.HealthTimeoutSec == right.HealthTimeoutSec
}

func floatingGatewayResponse(state floatingGatewaySnapshot, plan *models.FloatingGatewayPlan, changed bool) *models.FloatingGatewayResponse {
	capability := "available"
	status := "disabled"
	holder := "none"
	peerState := "unverifiable"
	reason := ""
	if !state.Installed {
		capability, status, reason = "not_installed", "disabled", "dependency_not_installed"
	} else if state.Config.Enabled && !state.Running {
		status, reason = "error", "service_not_running"
	} else if state.Config.Enabled && state.LocalHolder {
		status, holder = "healthy", "local"
	} else if state.Config.Enabled && state.PeerHolder {
		status, holder, peerState = "healthy", "peer", "healthy"
	} else if state.Config.Enabled {
		status, holder = "starting", "unknown"
	}
	return &models.FloatingGatewayResponse{Result: &models.FloatingGatewayResult{Config: state.Config, Status: &models.FloatingGatewayStatus{
		Capability: capability, State: status, Holder: holder, ServiceRunning: state.Running, PeerState: peerState, ExternalState: "unverifiable", Reason: reason,
	}, Plan: plan, Changed: changed}}
}

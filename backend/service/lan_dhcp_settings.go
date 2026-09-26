package service

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

var errLanDHCPRecoveryRequired = errors.New("DHCP settings recovery required")

var lanDHCPLeaseTimePattern = regexp.MustCompile(`^[1-9][0-9]*[mh]$`)

type lanDHCPSettingsSnapshot struct {
	Settings           *models.LanDHCPSettings
	Prefix             netip.Prefix
	RouterAddress      string
	Authority          string
	Version            string
	AffectedDevices    int64
	ProtectedAddresses map[string]string
	Targets            map[string]string
}

type lanDHCPSettingsExecutionPlan struct {
	Public         *models.LanDHCPSettingsResult
	State          lanDHCPSettingsSnapshot
	Desired        *models.LanDHCPSettings
	DefaultGateway string
}

type lanDHCPSettingsStore interface {
	Read(context.Context) (lanDHCPSettingsSnapshot, error)
	Apply(context.Context, lanDHCPSettingsExecutionPlan) error
}

type LanDHCPSettingsModule struct {
	mu           sync.Mutex
	store        lanDHCPSettingsStore
	transactions *TaskTransactionJournal
}

func NewLanDHCPSettingsModule(store lanDHCPSettingsStore) *LanDHCPSettingsModule {
	return &LanDHCPSettingsModule{store: store, transactions: newMemoryTaskTransactionJournal()}
}

func NewDefaultLanDHCPSettingsModule(gateway *GatewayPolicyModule, routerContext *RouterContextModule) *LanDHCPSettingsModule {
	module := NewLanDHCPSettingsModule(&defaultLanDHCPSettingsStore{gateway: gateway, routerContext: routerContext})
	module.transactions = NewDefaultTaskTransactionJournal()
	return module
}

func (module *LanDHCPSettingsModule) Get(ctx context.Context) (*models.LanDHCPSettingsResponse, error) {
	state, err := module.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	editable := state.Authority == "local" || state.Authority == "none_detected"
	result := &models.LanDHCPSettingsResult{
		Settings: state.Settings, Version: state.Version, Editable: editable, AffectedDevices: state.AffectedDevices,
		Conflicts: []*models.LanDHCPConflict{}, ReloadServices: []string{},
	}
	if !editable {
		result.ReadOnlyReason = "DHCP settings are read-only because another or uncertain server owns address allocation"
	}
	return &models.LanDHCPSettingsResponse{Result: result}, nil
}

func (module *LanDHCPSettingsModule) Plan(ctx context.Context, request *models.LanDHCPSettingsApplyRequest) (*models.LanDHCPSettingsResponse, error) {
	plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
}

func (module *LanDHCPSettingsModule) Apply(ctx context.Context, request *models.LanDHCPSettingsApplyRequest) (*models.LanDHCPSettingsResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	if plan.Public.Error != nil || !plan.Public.CanApply {
		return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != plan.State.Version {
		plan.Public.CanApply = false
		plan.Public.Error = &models.DevicePolicyError{Code: "conflict", Message: "DHCP settings changed; plan again"}
		return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
	}
	fingerprint, _ := json.Marshal(request)
	begin, beginErr := module.transactions.Begin(ctx, "lan_dhcp", request.IdempotencyKey, string(fingerprint))
	if beginErr != nil {
		plan.Public.Error = &models.DevicePolicyError{Code: "transaction_unavailable", Message: "could not create transaction record"}
		return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
	}
	plan.Public.Transaction = begin.Transaction
	if begin.Conflict {
		plan.Public.Error = &models.DevicePolicyError{Code: "conflict", Message: "idempotency key was already used for another DHCP change"}
		return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
	}
	if begin.Replay {
		plan.Public.Changed = false
		return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
	}
	if lanDHCPSettingsEqual(plan.State.Settings, plan.Desired) {
		_ = module.transactions.Advance(ctx, begin.Transaction, "verify", "unchanged", "")
		return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
	}
	_ = module.transactions.Advance(ctx, begin.Transaction, "apply", "in_progress", "restore_task_snapshot")
	if err := module.store.Apply(ctx, plan); err != nil {
		status, recovery := "rolled_back", "retry"
		if errors.Is(err, errLanDHCPRecoveryRequired) {
			status, recovery = "recovery_required", "restore_task_snapshot"
		}
		_ = module.transactions.Advance(ctx, begin.Transaction, "rollback", status, recovery)
		plan.Public.Error = &models.DevicePolicyError{Code: status, Message: err.Error()}
		return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
	}
	plan.Public.Changed = true
	_ = module.transactions.Advance(ctx, begin.Transaction, "verify", "committed", "")
	return &models.LanDHCPSettingsResponse{Result: plan.Public}, nil
}

func (module *LanDHCPSettingsModule) plan(ctx context.Context, request *models.LanDHCPSettingsApplyRequest) (lanDHCPSettingsExecutionPlan, error) {
	state, err := module.store.Read(ctx)
	if err != nil {
		return lanDHCPSettingsExecutionPlan{}, err
	}
	result := &models.LanDHCPSettingsResult{
		Settings: state.Settings, Version: state.Version, Editable: state.Authority == "local" || state.Authority == "none_detected",
		AffectedDevices: state.AffectedDevices, Conflicts: []*models.LanDHCPConflict{}, ReloadServices: []string{"dnsmasq"},
	}
	plan := lanDHCPSettingsExecutionPlan{Public: result, State: state}
	if !result.Editable {
		result.ReadOnlyReason = "address allocation is owned by another or uncertain DHCP server"
		result.Error = &models.DevicePolicyError{Code: "dhcp_authority_unavailable", Message: result.ReadOnlyReason}
		return plan, nil
	}
	if request == nil || request.Settings == nil {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "DHCP settings are required"}
		return plan, nil
	}
	desired := *request.Settings
	desired.PoolStart, desired.PoolEnd, desired.LeaseTime, desired.DefaultTargetID = strings.TrimSpace(desired.PoolStart), strings.TrimSpace(desired.PoolEnd), strings.TrimSpace(desired.LeaseTime), strings.TrimSpace(desired.DefaultTargetID)
	plan.Desired = &desired
	result.Settings = &desired
	start, startErr := netip.ParseAddr(desired.PoolStart)
	end, endErr := netip.ParseAddr(desired.PoolEnd)
	if startErr != nil || endErr != nil || !start.Is4() || !end.Is4() || !state.Prefix.Contains(start) || !state.Prefix.Contains(end) || start.Compare(end) > 0 {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "address pool must be an ordered IPv4 range inside the LAN subnet"}
		return plan, nil
	}
	networkOrdinal := ipv4Ordinal(state.Prefix.Masked().Addr())
	hostCount := uint64(1) << uint(32-state.Prefix.Bits())
	broadcastOrdinal := uint64(networkOrdinal) + hostCount - 1
	if uint64(ipv4Ordinal(start)) <= uint64(networkOrdinal) || uint64(ipv4Ordinal(end)) >= broadcastOrdinal {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "address pool cannot include the network or broadcast address"}
		return plan, nil
	}
	if !lanDHCPLeaseTimePattern.MatchString(desired.LeaseTime) {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "lease time must use minutes or hours, for example 30m or 12h"}
		return plan, nil
	}
	if addressInRange(state.RouterAddress, start, end) {
		result.Conflicts = append(result.Conflicts, &models.LanDHCPConflict{Address: state.RouterAddress, Kind: "router"})
	}
	for address, kind := range state.ProtectedAddresses {
		if address != state.RouterAddress && addressInRange(address, start, end) {
			result.Conflicts = append(result.Conflicts, &models.LanDHCPConflict{Address: address, Kind: kind})
		}
	}
	sort.Slice(result.Conflicts, func(i, j int) bool { return result.Conflicts[i].Address < result.Conflicts[j].Address })
	if len(result.Conflicts) > 0 {
		result.Error = &models.DevicePolicyError{Code: "address_conflict", Message: "address pool overlaps router or protected infrastructure addresses"}
		return plan, nil
	}
	gateway, ok := state.Targets[desired.DefaultTargetID]
	gatewayAddress, gatewayErr := netip.ParseAddr(gateway)
	if !ok || gatewayErr != nil || !gatewayAddress.Is4() || !state.Prefix.Contains(gatewayAddress) {
		result.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "default Internet Path is unavailable"}
		return plan, nil
	}
	plan.DefaultGateway = gateway
	if state.Settings != nil && state.Settings.Enabled && !desired.Enabled {
		result.RecoveryGuidance = "Reconnect to the router by static LAN address and re-enable DHCP, or enable the intended external DHCP server."
		if !request.ConfirmDisable {
			result.Error = &models.DevicePolicyError{Code: "confirmation_required", Message: "confirm DHCP shutdown after reviewing affected devices and recovery guidance"}
			return plan, nil
		}
	}
	result.CanApply = true
	return plan, nil
}

func addressInRange(raw string, start, end netip.Addr) bool {
	address, err := netip.ParseAddr(raw)
	return err == nil && address.Is4() && address.Compare(start) >= 0 && address.Compare(end) <= 0
}

func lanDHCPSettingsEqual(left, right *models.LanDHCPSettings) bool {
	return left != nil && right != nil && *left == *right
}

type defaultLanDHCPSettingsStore struct {
	gateway       *GatewayPolicyModule
	routerContext *RouterContextModule
}

var (
	lanDHCPSettingsTakeSnapshot = snapshotDhcpConfig
	lanDHCPSettingsMutate       = func(plan lanDHCPSettingsExecutionPlan) error {
		return mutateLanDHCPSettingsAt(filepath.Dir(dhcpConfigPath), plan)
	}
	lanDHCPSettingsReload  = reloadAndVerifyDnsmasq
	lanDHCPSettingsRestore = restoreDhcpConfig
)

func (store *defaultLanDHCPSettingsStore) Read(ctx context.Context) (lanDHCPSettingsSnapshot, error) {
	tree := uci.NewTree(filepath.Dir(dhcpConfigPath))
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return lanDHCPSettingsSnapshot{}, err
	}
	if err := tree.LoadConfig("network", true); err != nil {
		return lanDHCPSettingsSnapshot{}, err
	}
	lanAddress, _ := tree.GetLast("network", "lan", "ipaddr")
	netmask, _ := tree.GetLast("network", "lan", "netmask")
	prefix, err := prefixFromAddressAndMask(lanAddress, netmask)
	if err != nil {
		return lanDHCPSettingsSnapshot{}, err
	}
	startRaw, _ := tree.GetLast("dhcp", "lan", "start")
	limitRaw, _ := tree.GetLast("dhcp", "lan", "limit")
	poolStart, poolEnd := poolFromOffsets(prefix, startRaw, limitRaw)
	leaseTime, _ := tree.GetLast("dhcp", "lan", "leasetime")
	if leaseTime == "" {
		leaseTime = "12h"
	}
	ignore, _ := tree.GetLast("dhcp", "lan", "ignore")
	dhcpv4, _ := tree.GetLast("dhcp", "lan", "dhcpv4")
	authority := "local"
	if store.routerContext != nil {
		if response, contextErr := store.routerContext.Get(ctx, "lan"); contextErr == nil && response != nil && response.Result != nil {
			authority = response.Result.DHCPAuthority
		}
	}
	targets := map[string]string{"self": lanAddress}
	defaultTarget := "self"
	var dhcpState *LanDhcpState
	if store.gateway != nil {
		if gatewayState, gatewayErr := store.gateway.store.ReadState(ctx); gatewayErr == nil {
			dhcpState = gatewayState.DHCP
			index := buildGatewayTargetIndex(gatewayState)
			for id, target := range index.targets {
				if target.Public.Supported {
					effectiveGateway := gatewayTargetEffectiveGateway(id, index, gatewayState)
					if effectiveGateway != "" {
						targets[id] = effectiveGateway
					}
				}
			}
			currentGateway := detectDhcpGateway(gatewayState.LAN, gatewayState.DHCP.DhcpOptions)
			for id, gateway := range targets {
				if gateway == currentGateway {
					defaultTarget = id
				}
			}
		}
	}
	protected := map[string]string{lanAddress: "router"}
	for id, gateway := range targets {
		if id != "self" && id != "default" && gateway != "" {
			protected[gateway] = "gateway_target"
		}
	}
	if dhcpState != nil && dhcpState.FloatIP != nil {
		if dhcpState.FloatIP.SetIP != "" {
			protected[dhcpState.FloatIP.SetIP] = "floating_gateway"
		}
		if dhcpState.FloatIP.CheckIP != "" {
			protected[dhcpState.FloatIP.CheckIP] = "gateway_node"
		}
	}
	sections, _ := tree.GetSections("dhcp", "host")
	for _, section := range sections {
		if address, ok := tree.GetLast("dhcp", section, "ip"); ok && net.ParseIP(address) != nil {
			protected[address] = "address_reservation"
		}
	}
	raw, err := os.ReadFile(dhcpConfigPath)
	if err != nil {
		return lanDHCPSettingsSnapshot{}, err
	}
	sum := sha256.Sum256(raw)
	return lanDHCPSettingsSnapshot{
		Settings: &models.LanDHCPSettings{Enabled: dhcpIPv4ServingEnabled(ignore, dhcpv4), PoolStart: poolStart, PoolEnd: poolEnd, LeaseTime: leaseTime, DefaultTargetID: defaultTarget},
		Prefix:   prefix, RouterAddress: lanAddress, Authority: authority, Version: hex.EncodeToString(sum[:]), AffectedDevices: countDHCPLeases("/tmp/dhcp.leases"), ProtectedAddresses: protected, Targets: targets,
	}, nil
}

func (store *defaultLanDHCPSettingsStore) Apply(ctx context.Context, plan lanDHCPSettingsExecutionPlan) error {
	snapshot, err := lanDHCPSettingsTakeSnapshot()
	if err != nil {
		return fmt.Errorf("snapshot DHCP settings: %w", err)
	}
	if err := lanDHCPSettingsMutate(plan); err != nil {
		return rollbackLanDHCPSettings(ctx, snapshot, err)
	}
	if err := lanDHCPSettingsReload(ctx); err != nil {
		return rollbackLanDHCPSettings(ctx, snapshot, err)
	}
	return nil
}

func rollbackLanDHCPSettings(ctx context.Context, snapshot dhcpConfigSnapshot, cause error) error {
	if restoreErr := lanDHCPSettingsRestore(ctx, snapshot); restoreErr != nil {
		return fmt.Errorf("%w: %v; restore failed: %v", errLanDHCPRecoveryRequired, cause, restoreErr)
	}
	return fmt.Errorf("DHCP settings apply failed and original configuration was restored: %w", cause)
}

func mutateLanDHCPSettingsAt(configDir string, plan lanDHCPSettingsExecutionPlan) error {
	if plan.Desired == nil {
		return errors.New("DHCP settings are required")
	}
	tree := uci.NewTree(configDir)
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return err
	}
	start := ipv4Ordinal(netip.MustParseAddr(plan.Desired.PoolStart)) - ipv4Ordinal(plan.State.Prefix.Masked().Addr())
	end := ipv4Ordinal(netip.MustParseAddr(plan.Desired.PoolEnd)) - ipv4Ordinal(plan.State.Prefix.Masked().Addr())
	if !tree.Set("dhcp", "lan", "start", strconv.FormatUint(uint64(start), 10)) ||
		!tree.Set("dhcp", "lan", "limit", strconv.FormatUint(uint64(end-start+1), 10)) ||
		!tree.Set("dhcp", "lan", "leasetime", plan.Desired.LeaseTime) {
		return errors.New("set DHCP address pool")
	}
	if plan.Desired.Enabled {
		tree.Del("dhcp", "lan", "ignore")
		tree.Del("dhcp", "lan", "dhcpv4")
	} else if !tree.Set("dhcp", "lan", "ignore", "1") {
		return errors.New("disable DHCP")
	}
	tree.Del("dhcp", "lan", "dhcp_option")
	if plan.DefaultGateway != "" && plan.DefaultGateway != plan.State.RouterAddress && !tree.SetType("dhcp", "lan", "dhcp_option", uci.TypeList, gatewayAndDNSOptions(plan.DefaultGateway)...) {
		return errors.New("set default Internet Path")
	}
	return tree.Commit()
}

func prefixFromAddressAndMask(address, mask string) (netip.Prefix, error) {
	ip := net.ParseIP(address).To4()
	netmask := net.ParseIP(mask).To4()
	if ip == nil || netmask == nil {
		return netip.Prefix{}, errors.New("LAN IPv4 address or netmask is invalid")
	}
	ones, bits := net.IPMask(netmask).Size()
	if bits != 32 || ones < 1 {
		return netip.Prefix{}, errors.New("LAN IPv4 netmask is invalid")
	}
	return netip.PrefixFrom(netip.AddrFrom4([4]byte{ip[0], ip[1], ip[2], ip[3]}), ones).Masked(), nil
}

func poolFromOffsets(prefix netip.Prefix, startRaw, limitRaw string) (string, string) {
	start, _ := strconv.ParseUint(startRaw, 10, 32)
	limit, _ := strconv.ParseUint(limitRaw, 10, 32)
	if start == 0 {
		start = 100
	}
	if limit == 0 {
		limit = 150
	}
	base := ipv4Ordinal(prefix.Masked().Addr())
	return ordinalIPv4(uint32(uint64(base) + start)).String(), ordinalIPv4(uint32(uint64(base) + start + limit - 1)).String()
}

func ipv4Ordinal(address netip.Addr) uint32 {
	bytes := address.As4()
	return uint32(bytes[0])<<24 | uint32(bytes[1])<<16 | uint32(bytes[2])<<8 | uint32(bytes[3])
}

func ordinalIPv4(value uint32) netip.Addr {
	return netip.AddrFrom4([4]byte{byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value)})
}

func countDHCPLeases(path string) int64 {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	seen := map[string]bool{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 {
			seen[strings.ToLower(fields[1])] = true
		}
	}
	return int64(len(seen))
}

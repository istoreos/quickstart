package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

const defaultLAN = "lan"

type lanScopeContextKey struct{}

// Scope is immutable for the duration of one request. Never put a current LAN
// in process-wide state: concurrent requests and schedulers have different LANs.
func withLANScope(ctx context.Context, scope *models.LANScope) context.Context {
	return context.WithValue(ctx, lanScopeContextKey{}, scope)
}

func selectedLANScope(ctx context.Context) *models.LANScope {
	scope, _ := ctx.Value(lanScopeContextKey{}).(*models.LANScope)
	return scope
}

func selectedLAN(ctx context.Context) string {
	if scope := selectedLANScope(ctx); scope != nil {
		return scope.NetworkID
	}
	return defaultLAN
}

type lanScopeReader interface {
	Read(context.Context) ([]*models.LANScope, error)
}

// LANScopeModule hides OpenWrt network, runtime-device, DHCP and zone binding.
// Discovery is read-only. Writes always resolve fresh bindings.
type LANScopeModule struct {
	mu       sync.Mutex
	reader   lanScopeReader
	now      func() time.Time
	cached   []*models.LANScope
	cachedAt time.Time
}

func NewLANScopeModule() *LANScopeModule {
	return &LANScopeModule{reader: systemLANScopeReader{configDir: "/etc/config"}, now: time.Now}
}

func (module *LANScopeModule) List(ctx context.Context) ([]*models.LANScope, error) {
	return module.list(ctx, false)
}

func (module *LANScopeModule) list(ctx context.Context, fresh bool) ([]*models.LANScope, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	if !fresh && module.cached != nil && module.now().Sub(module.cachedAt) < 2*time.Second {
		return cloneLANScopes(module.cached), nil
	}
	scopes, err := module.reader.Read(ctx)
	if err != nil {
		return nil, err
	}
	module.cached, module.cachedAt = scopes, module.now()
	return cloneLANScopes(scopes), nil
}

func (module *LANScopeModule) Resolve(ctx context.Context, network string) (*models.LANScope, error) {
	return module.resolve(ctx, network, false)
}

func (module *LANScopeModule) resolve(ctx context.Context, network string, fresh bool) (*models.LANScope, error) {
	if !validRouterContextLAN.MatchString(network) {
		return nil, &PolicyError{Code: "invalid_network", Message: "无效的局域网标识"}
	}
	scopes, err := module.list(ctx, fresh)
	if err != nil {
		return nil, err
	}
	for _, scope := range scopes {
		if scope.NetworkID == network || lanContainsAlias(scope.Aliases, network) {
			return scope, nil
		}
	}
	return nil, &PolicyError{Code: "network_not_found", Message: "局域网不存在或当前不可管理，请重新选择网络"}
}

func cloneLANScopes(scopes []*models.LANScope) []*models.LANScope {
	cloned := make([]*models.LANScope, 0, len(scopes))
	for _, original := range scopes {
		copy := *original
		copy.Aliases = append([]string{}, original.Aliases...)
		copy.IPv4Prefixes = append([]string{}, original.IPv4Prefixes...)
		copy.IPv6Prefixes = append([]string{}, original.IPv6Prefixes...)
		copy.FirewallZones = append([]string{}, original.FirewallZones...)
		copy.DHCPBindings = append([]models.LANDHCPBinding{}, original.DHCPBindings...)
		cloned = append(cloned, &copy)
	}
	return cloned
}

func lanContainsAlias(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type lanRuntimeInterface struct {
	ubusNetworkInterface
	Up bool `json:"up"`
}

type systemLANScopeReader struct {
	configDir string
	runtime   func(context.Context) ([]lanRuntimeInterface, error)
}

func (reader systemLANScopeReader) Read(ctx context.Context) ([]*models.LANScope, error) {
	tree := uci.NewTree(reader.configDir)
	for _, config := range []string{"network", "dhcp", "firewall"} {
		if err := tree.LoadConfig(config, true); err != nil {
			return nil, err
		}
	}
	read := reader.runtime
	if read == nil {
		read = func(ctx context.Context) ([]lanRuntimeInterface, error) {
			var dump struct {
				Interfaces []lanRuntimeInterface `json:"interface"`
			}
			err := UbusCallWithObject(ctx, "network.interface dump", &dump)
			return dump.Interfaces, err
		}
	}
	runtime, err := read(ctx)
	if err != nil {
		return nil, err
	}
	aliases := map[string]string{}
	if reader.runtime == nil {
		output, err := exec.CommandContext(ctx, "uci", "-c", reader.configDir, "-X", "show", "dhcp").Output()
		if err != nil {
			return nil, err
		}
		aliases = dnsmasqInstanceAliases(string(output))
	}
	return buildLANScopes(tree, runtime, aliases), nil
}

func buildLANScopes(tree uci.Tree, runtime []lanRuntimeInterface, instanceAliases ...map[string]string) []*models.LANScope {
	runtimeByName := map[string]lanRuntimeInterface{}
	for _, network := range runtime {
		runtimeByName[network.Name] = network
	}
	zones := map[string][]string{}
	upstream := map[string]bool{"wan": true, "wan6": true, "loopback": true, "docker": true}
	zoneSections, _ := tree.GetSections("firewall", "zone")
	for _, section := range zoneSections {
		name, _ := tree.GetLast("firewall", section, "name")
		networks, _ := tree.Get("firewall", section, "network")
		for _, network := range networks {
			zones[network] = append(zones[network], name)
			if name == "wan" || name == "docker" || lanZoneIsUplink(tree, section, networks, runtimeByName) {
				upstream[network] = true
			}
		}
	}
	sections, _ := tree.GetSections("network", "interface")
	sort.SliceStable(sections, func(i, j int) bool {
		if sections[i] == defaultLAN || sections[j] == defaultLAN {
			return sections[i] == defaultLAN
		}
		return sections[i] < sections[j]
	})
	scopes := make([]*models.LANScope, 0)
	byDevice := map[string]*models.LANScope{}
	for _, section := range sections {
		if upstream[section] || !validRouterContextLAN.MatchString(section) {
			continue
		}
		proto, _ := tree.GetLast("network", section, "proto")
		if proto != "static" && proto != "none" && proto != "dhcp" && proto != "dhcpv6" {
			continue
		}
		network := runtimeByName[section]
		device := network.L3Device
		if device == "" {
			device = network.Device
		}
		if device == "" {
			device, _ = tree.GetLast("network", section, "device")
		}
		if device == "" {
			device, _ = tree.GetLast("network", section, "ifname")
		}
		scope := &models.LANScope{
			NetworkID: section, DisplayName: section, Device: device,
			Aliases: []string{}, State: "down", IPv4Prefixes: []string{}, IPv6Prefixes: []string{},
			FirewallZones: uniqueSortedNonEmpty(zones[section]), DHCPBindings: lanDHCPBindings(tree, section, instanceAliases...),
		}
		if network.Up {
			scope.State = "up"
		}
		for _, address := range network.Ipv4 {
			scope.IPv4Prefixes = appendLANPrefix(scope.IPv4Prefixes, address)
		}
		for _, address := range network.Ipv6 {
			scope.IPv6Prefixes = appendLANPrefix(scope.IPv6Prefixes, address)
		}
		for _, assignment := range network.Ipv6PA {
			if assignment != nil {
				scope.IPv6Prefixes = appendLANPrefix(scope.IPv6Prefixes, &ubusNetworkInterfaceAddress{Address: assignment.Address, Mask: assignment.Mask})
			}
		}
		if !network.Up {
			scope.ReadOnlyReason = "network_down"
		} else if len(scope.IPv4Prefixes)+len(scope.IPv6Prefixes) == 0 {
			scope.ReadOnlyReason = "network_address_unavailable"
		}
		if device == "" || strings.ContainsAny(device, "/ \t\r\n") || strings.HasPrefix(device, "@") {
			scope.ReadOnlyReason = "network_device_unavailable"
		} else if primary := byDevice[device]; primary != nil {
			primary.Aliases = append(primary.Aliases, section)
			primary.IPv4Prefixes = uniqueSortedNonEmpty(append(primary.IPv4Prefixes, scope.IPv4Prefixes...))
			primary.IPv6Prefixes = uniqueSortedNonEmpty(append(primary.IPv6Prefixes, scope.IPv6Prefixes...))
			primary.FirewallZones = uniqueSortedNonEmpty(append(primary.FirewallZones, scope.FirewallZones...))
			// Aliases are visible as one link. Distinct DHCP bindings cannot be
			// assumed interchangeable for network configuration writes.
			if len(scope.DHCPBindings) > 0 || len(scope.IPv4Prefixes) > 0 {
				primary.ReadOnlyReason = "shared_network_device"
			}
			continue
		} else {
			byDevice[device] = scope
		}
		scopes = append(scopes, scope)
	}
	for _, scope := range scopes {
		index, _ := os.ReadFile(filepath.Join("/sys/class/net", scope.Device, "ifindex"))
		data, _ := json.Marshal(struct {
			Scope   *models.LANScope
			IfIndex string
		}{scope, strings.TrimSpace(string(index))})
		sum := sha256.Sum256(data)
		scope.Version = hex.EncodeToString(sum[:])
	}
	return scopes
}

func appendLANPrefix(prefixes []string, address *ubusNetworkInterfaceAddress) []string {
	if address == nil {
		return prefixes
	}
	parsed, err := netip.ParseAddr(address.Address)
	if err != nil || address.Mask < 0 || address.Mask > parsed.BitLen() {
		return prefixes
	}
	return uniqueSortedNonEmpty(append(prefixes, netip.PrefixFrom(parsed, address.Mask).Masked().String()))
}

// go-uci identifies anonymous sections by selector, while procd and dnsmasq's
// instance filter use libuci's generated section ID. Preserve both identities.
func dnsmasqInstanceAliases(output string) map[string]string {
	aliases := map[string]string{}
	index := 0
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "dhcp.") || !strings.HasSuffix(line, "=dnsmasq") {
			continue
		}
		name := strings.TrimSuffix(strings.TrimPrefix(line, "dhcp."), "=dnsmasq")
		aliases[fmt.Sprintf("@dnsmasq[%d]", index)] = name
		aliases[name] = name
		index++
	}
	return aliases
}

func lanDHCPBindings(tree uci.Tree, network string, instanceAliases ...map[string]string) []models.LANDHCPBinding {
	bindings := []models.LANDHCPBinding{}
	sections, _ := tree.GetSections("dhcp", "dhcp")
	instances, _ := tree.GetSections("dhcp", "dnsmasq")
	for _, section := range sections {
		bound, _ := tree.GetLast("dhcp", section, "interface")
		if bound != network && !(bound == "" && section == network) {
			continue
		}
		instance, _ := tree.GetLast("dhcp", section, "instance")
		if len(instanceAliases) > 0 && !dhcpTreeHasSection(tree, "dnsmasq", instance) {
			for _, selector := range instances {
				if instance == instanceAliases[0][selector] {
					instance = selector
					break
				}
			}
		}
		if instance == "" && len(instances) == 1 {
			instance = instances[0]
		}
		if index, err := strconv.Atoi(instance); err == nil && index >= 0 && index < len(instances) {
			instance = instances[index]
		}
		runtimeInstance := instance
		if strings.HasPrefix(instance, "@") {
			runtimeInstance = ""
		}
		if len(instanceAliases) > 0 && instanceAliases[0][instance] != "" {
			runtimeInstance = instanceAliases[0][instance]
		}
		leaseFile, _ := tree.GetLast("dhcp", instance, "leasefile")
		if leaseFile == "" {
			leaseFile = "/tmp/dhcp.leases"
		}
		bindings = append(bindings, models.LANDHCPBinding{Section: section, Instance: instance, RuntimeInstance: runtimeInstance, LeaseFile: leaseFile})
	}
	return bindings
}

func (scope *LANScopeModule) requireWritable(ctx context.Context) error {
	selected := selectedLANScope(ctx)
	if selected == nil {
		return errors.New("LAN scope is required")
	}
	current, err := scope.resolve(ctx, selected.NetworkID, true)
	if err != nil {
		return err
	}
	if current.Version != selected.Version {
		return &PolicyError{Code: "network_changed", Message: "网络配置已改变，请重新预览"}
	}
	if current.ReadOnlyReason != "" {
		return &PolicyError{Code: current.ReadOnlyReason, Message: "当前网络尚不具备此操作所需的独立配置"}
	}
	return nil
}

// A renamed WAN needs positive firewall and routing evidence. A default route
// alone is also normal on an observed/bypass LAN and must not hide that LAN.
func lanZoneIsUplink(tree uci.Tree, zone string, networks []string, runtime map[string]lanRuntimeInterface) bool {
	name, _ := tree.GetLast("firewall", zone, "name")
	input, _ := tree.GetLast("firewall", zone, "input")
	forward, _ := tree.GetLast("firewall", zone, "forward")
	rejects := func(value string) bool { return value == "REJECT" || value == "DROP" }
	if name == "lan" || !rejects(input) || !rejects(forward) || lanContainsAlias(networks, defaultLAN) {
		return false
	}
	sections, _ := tree.GetSections("firewall", "forwarding")
	incoming := false
	for _, section := range sections {
		src, _ := tree.GetLast("firewall", section, "src")
		dest, _ := tree.GetLast("firewall", section, "dest")
		if src != "" && src != name && dest == name {
			incoming = true
		}
	}
	if !incoming {
		return false
	}
	dhcp, _ := tree.GetSections("dhcp", "dhcp")
	for _, section := range dhcp {
		network, _ := tree.GetLast("dhcp", section, "interface")
		ignore, _ := tree.GetLast("dhcp", section, "ignore")
		if ignore != "1" && lanContainsAlias(networks, network) {
			return false
		}
	}
	for _, network := range networks {
		for _, route := range runtime[network].Route {
			if route != nil && route.Mask == 0 && (route.Target == "0.0.0.0" || route.Target == "::") {
				return true
			}
		}
	}
	return false
}

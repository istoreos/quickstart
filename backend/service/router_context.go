package service

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

const defaultRouterContextLAN = "lan"

var validRouterContextLAN = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

type RouterContextFacts struct {
	LAN                     string
	LocalAddress            string
	UpstreamGateways        []string
	LocalDHCPConfigured     bool
	LocalDHCPHealthy        bool
	ObservedExternalServers []string
}

type RouterContextProbe interface {
	Probe(context.Context, string) (*RouterContextFacts, error)
}

type RouterContextModule struct {
	probe RouterContextProbe
}

func NewRouterContextModule(probe RouterContextProbe) *RouterContextModule {
	return &RouterContextModule{probe: probe}
}

func NewDefaultRouterContextModule() *RouterContextModule {
	return NewRouterContextModule(&defaultRouterContextProbe{})
}

func (module *RouterContextModule) Get(ctx context.Context, lan string) (*models.RouterContextResponse, error) {
	lan = strings.TrimSpace(lan)
	if lan == "" {
		lan = defaultRouterContextLAN
	}
	if !validRouterContextLAN.MatchString(lan) {
		return nil, errors.New("invalid LAN identifier")
	}
	facts, err := module.probe.Probe(ctx, lan)
	if err != nil {
		return &models.RouterContextResponse{Result: buildRouterContextError(lan)}, nil
	}
	return &models.RouterContextResponse{Result: BuildRouterContext(*facts)}, nil
}

func BuildRouterContext(facts RouterContextFacts) *models.RouterContext {
	facts.UpstreamGateways = uniqueSortedNonEmpty(facts.UpstreamGateways)
	facts.ObservedExternalServers = uniqueSortedNonEmpty(facts.ObservedExternalServers)
	result := &models.RouterContext{
		LAN:                 facts.LAN,
		TopologyPosition:    topologyPosition(facts.UpstreamGateways),
		LocalAddress:        facts.LocalAddress,
		UpstreamGateways:    facts.UpstreamGateways,
		LocalDHCPConfigured: facts.LocalDHCPConfigured,
		LocalDHCPHealthy:    facts.LocalDHCPHealthy,
		ExternalDHCPServers: facts.ObservedExternalServers,
		Evidence:            make([]*models.RouterContextEvidence, 0, 5),
	}
	result.Evidence = append(result.Evidence,
		&models.RouterContextEvidence{Kind: "lan_address", Source: "network_runtime", Value: facts.LocalAddress, Confidence: "observed"},
		&models.RouterContextEvidence{Kind: "local_dhcp_config", Source: "dhcp_config", Value: fmt.Sprintf("configured=%t", facts.LocalDHCPConfigured), Confidence: "configured"},
	)
	for _, gateway := range facts.UpstreamGateways {
		result.Evidence = append(result.Evidence, &models.RouterContextEvidence{Kind: "upstream_gateway", Source: "network_runtime", Value: gateway, Confidence: "observed"})
	}
	for _, server := range facts.ObservedExternalServers {
		result.Evidence = append(result.Evidence, &models.RouterContextEvidence{Kind: "external_dhcp_server", Source: "network_runtime", Value: server, Confidence: "observed"})
	}

	switch {
	case facts.LocalDHCPConfigured && !facts.LocalDHCPHealthy && len(facts.ObservedExternalServers) == 0:
		result.DHCPAuthority = "error"
		result.RouteEditability = readOnlyRoute("local_dhcp_unhealthy", "本机 DHCP 服务异常，请先检查服务；不会自动开启或重启 DHCP。")
	case facts.LocalDHCPConfigured && len(facts.ObservedExternalServers) > 0:
		result.DHCPAuthority = "ambiguous"
		result.RouteEditability = readOnlyRoute("multiple_dhcp_evidence", "检测到本机和外部 DHCP 证据，请先排除重复 DHCP 服务。")
	case facts.LocalDHCPConfigured && facts.LocalDHCPHealthy:
		result.DHCPAuthority = "local"
		result.RouteEditability = &models.RouteEditability{Editable: true, Reason: "local_dhcp_authority"}
	case len(facts.ObservedExternalServers) == 1:
		result.DHCPAuthority = "external_observed"
		result.RouteEditability = readOnlyRoute("external_dhcp_authority", "请在主路由的 DHCP 设置中调整路线；可复制本机地址定位此设备。")
	case len(facts.ObservedExternalServers) > 1:
		result.DHCPAuthority = "ambiguous"
		result.RouteEditability = readOnlyRoute("multiple_external_dhcp_servers", "检测到多个 DHCP 服务证据，请先确认唯一的地址分配设备。")
	default:
		result.DHCPAuthority = "none_detected"
		result.RouteEditability = readOnlyRoute("dhcp_authority_not_detected", "当前无法确认地址分配设备；请检查主路由，不会自动开启本机 DHCP。")
	}
	return result
}

func buildRouterContextError(lan string) *models.RouterContext {
	return &models.RouterContext{
		LAN:                 lan,
		TopologyPosition:    "unknown",
		UpstreamGateways:    []string{},
		ExternalDHCPServers: []string{},
		DHCPAuthority:       "error",
		RouteEditability:    readOnlyRoute("router_context_unavailable", "暂时无法判断 DHCP 分配权，请重试；设备浏览和资料编辑仍可使用。"),
		Evidence:            []*models.RouterContextEvidence{},
	}
}

func readOnlyRoute(reason, guidance string) *models.RouteEditability {
	return &models.RouteEditability{Editable: false, Reason: reason, Guidance: guidance}
}

func topologyPosition(gateways []string) string {
	switch len(gateways) {
	case 0:
		return "lan_gateway_candidate"
	case 1:
		return "downstream_router"
	default:
		return "ambiguous"
	}
}

func uniqueSortedNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

type defaultRouterContextProbe struct{}

var routerContextDnsmasqHealthy = func(ctx context.Context) bool {
	return exec.CommandContext(ctx, "/etc/init.d/dnsmasq", "running").Run() == nil
}

func (probe *defaultRouterContextProbe) Probe(ctx context.Context, lan string) (*RouterContextFacts, error) {
	var network ubusNetworkInterface
	if err := UbusCallWithObject(ctx, fmt.Sprintf("network.interface.%s status", lan), &network); err != nil {
		return nil, err
	}
	facts := &RouterContextFacts{LAN: lan}
	if len(network.Ipv4) > 0 {
		facts.LocalAddress = network.Ipv4[0].Address
	}
	for _, route := range network.Route {
		if route != nil && route.Target == "0.0.0.0" && route.Mask == 0 {
			facts.UpstreamGateways = append(facts.UpstreamGateways, route.Nexthop)
		}
	}
	for _, key := range []string{"dhcpserver", "dhcp_server", "serverid", "server_id"} {
		if value, ok := network.Data[key].(string); ok {
			facts.ObservedExternalServers = append(facts.ObservedExternalServers, value)
		}
	}

	tree := uci.NewTree("/etc/config")
	if err := tree.LoadConfig("dhcp", true); err != nil {
		return nil, err
	}
	facts.LocalDHCPConfigured = routerContextLocalDHCPConfigured(tree, lan)
	if facts.LocalDHCPConfigured {
		facts.LocalDHCPHealthy = routerContextDnsmasqHealthy(ctx)
	}
	return facts, nil
}

func routerContextLocalDHCPConfigured(tree uci.Tree, lan string) bool {
	sections, _ := tree.GetSections("dhcp", "dhcp")
	for _, section := range sections {
		boundLAN, _ := tree.GetLast("dhcp", section, "interface")
		if section != lan && boundLAN != lan {
			continue
		}
		ignore, _ := tree.GetLast("dhcp", section, "ignore")
		if ignore == "1" {
			return false
		}
		dhcpv4, _ := tree.GetLast("dhcp", section, "dhcpv4")
		switch strings.ToLower(strings.TrimSpace(dhcpv4)) {
		case "disabled", "relay":
			return false
		default:
			// An omitted mode is the dnsmasq-compatible legacy default;
			// server and hybrid modes both provide local DHCPv4 service.
			return true
		}
	}
	return false
}

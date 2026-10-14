package service

import (
	"context"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

// rateLimitProvider is the seam between product policy and a concrete traffic
// shaper. Implementations own provider-specific persistence, reload and runtime
// inspection; callers only deal in devices and desired rates.
type rateLimitProvider interface {
	Name() string
	IdentityKind() string
	SupportsIPv6() bool
	Inspect(context.Context, rateLimitTarget) (rateLimitProviderObservation, error)
	Apply(context.Context, rateLimitTarget, models.DeviceSpeedPolicy) error
}

type rateLimitRouteResolver interface {
	Resolve(context.Context, string) (rateLimitRoute, error)
}

type rateLimitRuntimeInspector interface {
	Loaded(context.Context, string) (bool, error)
	Offload(context.Context) (string, error)
}

type rateLimitTarget struct {
	DeviceID    string
	MAC         string
	DisplayName string
	CurrentIPv4 string
	ReservedIP  string
	HasIPv6     bool
	Route       rateLimitRoute
}

type rateLimitRoute struct {
	TargetID string
	Kind     string
	Gateway  string
	Local    bool
}

type rateLimitProviderObservation struct {
	Policy     models.DeviceSpeedPolicy
	Configured bool
	Loaded     bool
	Verified   bool
	Offload    string
}

// RateLimitModule is deliberately a deep module: it hides route ownership,
// stable-address requirements and provider details behind two operations.
type RateLimitModule struct {
	providers map[string]rateLimitProvider
	selected  func(context.Context) string
	routes    rateLimitRouteResolver
	now       func() time.Time
}

func NewRateLimitModule(provider rateLimitProvider, routes rateLimitRouteResolver) *RateLimitModule {
	providers := map[string]rateLimitProvider{}
	if provider != nil {
		providers[provider.Name()] = provider
	}
	return &RateLimitModule{providers: providers, selected: func(context.Context) string {
		if provider == nil {
			return ""
		}
		return provider.Name()
	}, routes: routes, now: time.Now}
}

func NewDefaultRateLimitModule(gateway *GatewayPolicyModule) *RateLimitModule {
	module := NewRateLimitModule(newEqosRateLimitProvider(deviceRestrictionConfigDir(), &systemRateLimitRuntimeInspector{}), &gatewayRateLimitRouteResolver{gateway: gateway})
	module.providers["bandix"] = NewBandixRateLimitProvider(defaultBandixBaseURL(), nil)
	module.providers[nativePolicyProviderName] = NewNativeRateLimitProvider(defaultNativePolicyBaseURL(), nil)
	module.selected = func(context.Context) string { return effectiveRateLimitProvider(defaultRateLimitProviderPath) }
	return module
}

func (module *RateLimitModule) Inspect(ctx context.Context, target rateLimitTarget) (*models.DeviceSpeedPolicy, *models.RateLimitEnforcement, error) {
	provider := module.provider(ctx)
	if module == nil || provider == nil {
		return &models.DeviceSpeedPolicy{}, unavailableRateLimit("provider_unavailable"), nil
	}
	target.Route = rateLimitRoute{TargetID: "unknown"}
	if module.routes != nil {
		route, err := module.routes.Resolve(ctx, target.MAC)
		if err != nil {
			return &models.DeviceSpeedPolicy{}, unavailableRateLimit("route_unavailable"), nil
		}
		target.Route = route
	}
	observation, err := provider.Inspect(ctx, target)
	if err != nil {
		return nil, nil, err
	}
	enforcement := module.enforcement(provider, target, observation)
	policy := observation.Policy
	return &policy, enforcement, nil
}

func (module *RateLimitModule) Apply(ctx context.Context, target rateLimitTarget, desired models.DeviceSpeedPolicy) error {
	provider := module.provider(ctx)
	if module == nil || provider == nil {
		return errors.New("rate limit provider is unavailable")
	}
	if module.routes != nil {
		route, err := module.routes.Resolve(ctx, target.MAC)
		if err != nil {
			return err
		}
		target.Route = route
	}
	if desired.Enabled {
		if !target.Route.Local {
			return &PolicyError{Code: "execution_node_unavailable", Message: "设备当前不通过本机上网，请在实际执行上网的网关上设置限速"}
		}
		if provider.IdentityKind() == "ipv4" && stableRateLimitIPv4(target) == "" {
			return &PolicyError{Code: "address_reservation_required", Message: "请先为设备预留当前 IPv4 地址，再启用限速"}
		}
	}
	return provider.Apply(ctx, target, desired)
}

func (module *RateLimitModule) provider(ctx context.Context) rateLimitProvider {
	if module == nil {
		return nil
	}
	name := "eqos"
	if module.selected != nil {
		selected := strings.TrimSpace(module.selected(ctx))
		if selected != "" {
			name = selected
		}
	}
	return module.providers[name]
}

func (module *RateLimitModule) enforcement(provider rateLimitProvider, target rateLimitTarget, observation rateLimitProviderObservation) *models.RateLimitEnforcement {
	addressState := "missing"
	if target.CurrentIPv4 != "" {
		addressState = "current_only"
	}
	if provider.IdentityKind() == "mac" && target.MAC != "" {
		addressState = "stable_identity"
	} else if stableRateLimitIPv4(target) != "" {
		addressState = "stable"
	}
	result := &models.RateLimitEnforcement{
		Provider: provider.Name(), TargetID: target.Route.TargetID, Gateway: target.Route.Gateway,
		ExecutionNode: "remote", State: "inactive", Configured: observation.Configured, Loaded: observation.Loaded,
		Verified: observation.Verified, CanApply: target.Route.Local && (addressState == "stable" || addressState == "stable_identity"), AddressState: addressState,
		IPv4: target.CurrentIPv4, IPv6State: "not_present", OffloadState: observation.Offload,
		ObservedAt: module.now().UTC().Format(time.RFC3339),
	}
	addressReady := addressState == "stable" || addressState == "stable_identity"
	if target.Route.Local {
		result.ExecutionNode = "local"
	}
	if target.HasIPv6 && !provider.SupportsIPv6() {
		result.IPv6State = "unsupported"
		result.Warnings = append(result.Warnings, "ipv6_not_limited")
	} else if target.HasIPv6 && provider.SupportsIPv6() && provider.IdentityKind() == "mac" {
		result.IPv6State = "covered"
	}
	if observation.Offload == "risk" {
		result.Warnings = append(result.Warnings, "flow_offload_may_bypass_limit")
	}
	switch {
	case observation.Configured && !target.Route.Local:
		result.State, result.Reason = "configured_wrong_node", "execution_node_unavailable"
	case observation.Configured && !addressReady:
		result.State, result.Reason = "configured_unstable", "address_reservation_required"
	case observation.Verified:
		result.State = "verified"
	case observation.Loaded:
		result.State, result.Reason = "loaded_unverified", "runtime_effect_unverified"
	case observation.Configured:
		result.State, result.Reason = "configured_not_loaded", "provider_not_loaded"
	case !target.Route.Local:
		result.State, result.Reason = "unavailable", "execution_node_unavailable"
	case !addressReady:
		result.State, result.Reason = "needs_address_reservation", "address_reservation_required"
	default:
		result.State = "ready"
	}
	return result
}

func stableRateLimitIPv4(target rateLimitTarget) string {
	current, currentErr := netip.ParseAddr(strings.TrimSpace(target.CurrentIPv4))
	reserved, reservedErr := netip.ParseAddr(strings.TrimSpace(target.ReservedIP))
	if currentErr != nil || reservedErr != nil || !current.Is4() || current != reserved {
		return ""
	}
	return current.String()
}

func unavailableRateLimit(reason string) *models.RateLimitEnforcement {
	return &models.RateLimitEnforcement{Provider: "none", ExecutionNode: "unknown", State: "unavailable", Reason: reason, AddressState: "missing", IPv6State: "unknown", OffloadState: "unknown"}
}

type gatewayRateLimitRouteResolver struct{ gateway *GatewayPolicyModule }

func (resolver *gatewayRateLimitRouteResolver) Resolve(ctx context.Context, mac string) (rateLimitRoute, error) {
	if resolver == nil || resolver.gateway == nil || resolver.gateway.store == nil {
		return rateLimitRoute{}, errors.New("gateway policy is unavailable")
	}
	state, err := resolver.gateway.store.ReadState(ctx)
	if err != nil {
		return rateLimitRoute{}, err
	}
	index := buildGatewayTargetIndex(state)
	targetID := "default"
	for _, host := range state.Hosts {
		if normalizeInventoryMAC(host.MAC) == normalizeInventoryMAC(mac) {
			targetID = index.targetIDForTag(host.TagName)
			break
		}
	}
	gateway := gatewayTargetEffectiveGateway(targetID, index, state)
	kind := "unknown"
	if target := index.targets[targetID]; target != nil {
		kind = target.Public.Kind
	}
	return rateLimitRoute{TargetID: targetID, Kind: kind, Gateway: gateway, Local: gateway != "" && gateway == state.LAN.LanAddr}, nil
}

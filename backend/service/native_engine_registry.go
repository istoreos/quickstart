package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
)

type nativeEngineBinding struct {
	NetworkID      string
	InstanceID     string
	Interface      string
	Port           int
	DataDir        string
	InterfaceIndex int
}

// Endpoints come exclusively from local service configuration. The browser
// selects a logical network; it never supplies a URL, port, or kernel device.
type nativeEngineRegistry struct{ configDir string }

func (registry *nativeEngineRegistry) Resolve(ctx context.Context) (nativeEngineBinding, error) {
	scope := selectedLANScope(ctx)
	if scope == nil {
		return nativeEngineBinding{}, errors.New("netpolicy_network_scope_required")
	}
	tree := uci.NewTree(registry.configDir)
	if err := tree.LoadConfig("quickstart-netpolicy", true); err != nil {
		return nativeEngineBinding{}, fmt.Errorf("netpolicy_configuration_unavailable: %w", err)
	}
	sections, _ := tree.GetSections("quickstart-netpolicy", "quickstart-netpolicy")
	bindings := []nativeEngineBinding{}
	for _, section := range sections {
		enabled, _ := tree.GetLast("quickstart-netpolicy", section, "enabled")
		if enabled != "1" && enabled != "true" && enabled != "yes" && enabled != "on" {
			continue
		}
		get := func(option, fallback string) string {
			value, _ := tree.GetLast("quickstart-netpolicy", section, option)
			if value == "" {
				return fallback
			}
			return value
		}
		binding := nativeEngineBinding{InstanceID: section, NetworkID: get("network", ""), Interface: get("interface", ""), DataDir: get("data_dir", "")}
		port := get("port", "")
		if section == "main" {
			if binding.NetworkID == "" {
				binding.NetworkID = defaultLAN
			}
			if binding.Interface == "" {
				binding.Interface = "br-lan"
			}
			if binding.DataDir == "" {
				binding.DataDir = "/usr/share/quickstart-netpolicy"
			}
			if port == "" {
				port = "8765"
			}
		}
		binding.Port, _ = strconv.Atoi(port)
		if !validRouterContextLAN.MatchString(binding.NetworkID) || !validRouterContextLAN.MatchString(section) || binding.Interface == "" || binding.Port < 1 || binding.Port > 65535 || !filepath.IsAbs(binding.DataDir) {
			if binding.NetworkID == scope.NetworkID {
				return nativeEngineBinding{}, errors.New("netpolicy_invalid_instance_configuration")
			}
			continue
		}
		binding.DataDir = filepath.Clean(binding.DataDir)
		if resolved, err := filepath.EvalSymlinks(binding.DataDir); err == nil {
			binding.DataDir = resolved
		}
		bindings = append(bindings, binding)
	}
	var selected *nativeEngineBinding
	for i := range bindings {
		binding := &bindings[i]
		if binding.NetworkID != scope.NetworkID {
			continue
		}
		if selected != nil {
			return nativeEngineBinding{}, errors.New("netpolicy_duplicate_network_binding")
		}
		selected = binding
	}
	if selected == nil {
		return nativeEngineBinding{}, errors.New("netpolicy_instance_unavailable")
	}
	for _, other := range bindings {
		if other.InstanceID == selected.InstanceID {
			continue
		}
		if other.Interface == selected.Interface || other.Port == selected.Port || other.DataDir == selected.DataDir {
			return nativeEngineBinding{}, errors.New("netpolicy_instance_resource_conflict")
		}
	}
	if selected.Interface != scope.Device {
		return nativeEngineBinding{}, errors.New("netpolicy_interface_binding_mismatch")
	}
	index, err := os.ReadFile(filepath.Join("/sys/class/net", selected.Interface, "ifindex"))
	if err != nil {
		return nativeEngineBinding{}, errors.New("netpolicy_interface_unavailable")
	}
	selected.InterfaceIndex, err = strconv.Atoi(strings.TrimSpace(string(index)))
	if err != nil || selected.InterfaceIndex <= 0 {
		return nativeEngineBinding{}, errors.New("netpolicy_interface_unavailable")
	}
	return *selected, nil
}

func newScopedNativeRateLimitProvider() *nativeRateLimitProvider {
	provider := NewNativeRateLimitProvider(defaultNativePolicyBaseURL(), nil).(*nativeRateLimitProvider)
	provider.registry = &nativeEngineRegistry{configDir: "/etc/config"}
	return provider
}

func (provider *nativeRateLimitProvider) forContext(ctx context.Context) (*nativeRateLimitProvider, error) {
	if provider.registry == nil {
		return provider, nil
	}
	if selectedLANScope(ctx) == nil {
		scope, err := NewLANScopeModule().Resolve(ctx, defaultLAN)
		if err != nil {
			return nil, err
		}
		ctx = withLANScope(ctx, scope)
	}
	binding, err := provider.registry.Resolve(ctx)
	if err != nil {
		return nil, err
	}
	selected := *provider
	selected.registry, selected.binding = nil, &binding
	selected.baseURL = fmt.Sprintf("http://127.0.0.1:%d", binding.Port)
	return &selected, nil
}

package service

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type componentDefinition struct{ id, key, pkg, provider string }

var featureComponentDefinitions = []componentDefinition{
	{"quickstart-native", "device_speed_limit", "quickstart-netpolicy", "quickstart-native"},
	{"eqos", "compat_speed_limit", "luci-app-eqos", "eqos"},
	{"floatip", "floating_gateway", "app-meta-floatip", ""},
}

type componentOperation struct{ state, action string }

// Component status must remain readable during an install. This bounded map
// holds only the three registered dependencies and never stores configuration.
type componentOperations struct {
	mu    sync.Mutex
	items map[string]componentOperation
}

func (module *CapabilityActionModule) operation(key string, value *componentOperation) componentOperation {
	module.operations.mu.Lock()
	defer module.operations.mu.Unlock()
	if module.operations.items == nil {
		module.operations.items = map[string]componentOperation{}
	}
	if value != nil {
		if value.state == "" {
			delete(module.operations.items, key)
		} else {
			module.operations.items[key] = *value
		}
	}
	return module.operations.items[key]
}

func (module *CapabilityActionModule) Components(ctx context.Context, selected string) *models.FeatureComponentsResponse {
	result := &models.FeatureComponentsResult{SchemaVersion: 1, CurrentProvider: selected, Components: []*models.FeatureComponent{}}
	for _, definition := range featureComponentDefinitions {
		status, err := module.store.Read(ctx, definition.key)
		if err != nil || status == nil {
			status = &models.Capability{State: "error", Reason: "status_unavailable", Actions: []*models.CapabilityAction{{Kind: "retry"}}}
		}
		op := module.operation(definition.key, nil)
		if op.state == "pending" && (op.action == "install" && (status.State == "disabled" || status.State == "available" || status.State == "error" && status.Reason == "engine_unavailable") || op.action == "enable" && status.State == "available") {
			op = module.operation(definition.key, &componentOperation{})
		}
		var installed *bool
		if system, ok := module.store.(*systemCapabilityActionStore); ok {
			if value, err := system.componentInstalled(definition.key); err == nil {
				installed = &value
			}
		}
		networkID := ""
		if selectedLANScope(ctx) != nil {
			networkID = selectedLAN(ctx)
		}
		result.Components = append(result.Components, &models.FeatureComponent{NetworkID: networkID, Installed: installed, ID: definition.id, CapabilityKey: definition.key, Package: definition.pkg, Provider: definition.provider, Selected: definition.provider != "" && definition.provider == selected, Status: status, Operation: op.state})
	}
	return &models.FeatureComponentsResponse{Result: result}
}

func componentPackageInstalled(name string, stat func(string) (os.FileInfo, error)) (bool, error) {
	for _, path := range []string{"/usr/lib/opkg/info/" + name + ".control", "/lib/apk/packages/" + name + ".list"} {
		if _, err := stat(path); err == nil {
			return true, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
	}
	return false, nil
}

func nativeComponentCapability(ctx context.Context, probe func(context.Context) (bool, bool, string), run capabilityCommandRunner) (*models.Capability, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	installed, ready, reason := probe(ctx)
	status := &models.Capability{}
	switch {
	case ready:
		status.State = "available"
	case !installed && reason == "dependency_not_installed":
		status.State, status.Reason = "not_installed", reason
	case reason == "kernel_not_supported" || reason == "interface_not_supported":
		status.State, status.Reason = "unsupported", reason
	case installed && reason == "engine_unavailable":
		_, err := run(ctx, "/etc/init.d/quickstart-netpolicy", "running")
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			status.State, status.Reason = "disabled", "service_stopped"
		} else {
			status.State, status.Reason = "error", "engine_unavailable"
		}
	default:
		status.State, status.Reason = "error", "status_unavailable"
	}
	decorateCapabilityActions(status, "quickstart-netpolicy")
	return status, nil
}

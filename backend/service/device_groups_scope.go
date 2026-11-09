package service

import (
	"context"
	"github.com/istoreos/quickstart/backend/models"
	"time"
)

// Runtime state belongs to a network; the policy/store/applier implementations
// remain shared. Only the root has a scheduler, with fresh contexts per run.
func (module *DeviceGroupModule) forContext(ctx context.Context) *DeviceGroupModule {
	if module.root != nil || selectedLAN(ctx) == defaultLAN {
		return module
	}
	network := selectedLAN(ctx)
	module.mu.Lock()
	defer module.mu.Unlock()
	if module.scopes == nil {
		module.scopes = map[string]*DeviceGroupModule{}
	}
	if child := module.scopes[network]; child != nil {
		return child
	}
	child := &DeviceGroupModule{store: module.store, applier: module.applier, inventory: module.inventory, now: module.now, root: module, network: network, events: []*models.DeviceGroupEvent{}, lastApplied: map[string]string{}}
	module.scopes[network] = child
	return child
}

func (module *DeviceGroupModule) AttachLANScopes(resolver *LANScopeModule) {
	module.mu.Lock()
	module.resolver = resolver
	hasSaved := false
	if store, ok := module.store.(*jsonDeviceGroupStore); ok {
		for network := range lanStateFiles(store.path) {
			if network != defaultLAN {
				hasSaved = true
			}
		}
	}
	module.mu.Unlock()
	if hasSaved {
		module.startScheduler()
	}
}

func (module *DeviceGroupModule) runScheduledNetworks(now time.Time) {
	module.mu.Lock()
	resolver := module.resolver
	module.mu.Unlock()
	networks := map[string]bool{defaultLAN: true}
	if store, ok := module.store.(*jsonDeviceGroupStore); ok {
		for network := range lanStateFiles(store.path) {
			networks[network] = true
		}
	}
	for network := range networks {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if resolver != nil {
			scope, err := resolver.resolve(ctx, network, true)
			if err != nil {
				cancel()
				continue
			}
			ctx = context.WithValue(withLANScope(ctx, scope), lanScopeResolverKey{}, resolver)
		} else if network != defaultLAN {
			cancel()
			continue
		}
		module.runDue(ctx, now)
		cancel()
	}
}

func (module *DeviceGroupModule) validateScopeMutation(ctx context.Context, request *models.DeviceGroupMutationRequest, current deviceGroupState) *models.DevicePolicyError {
	if selectedLANScope(ctx) == nil {
		return nil
	}
	incoming := []string{}
	policies := []*models.GroupPolicy{}
	switch request.Action {
	case "upsert_group":
		incoming = request.Group.Members
		policies = append(policies, request.Group.Policy)
	case "set_device_policy":
		if request.DevicePolicy != nil {
			incoming = append(incoming, request.DeviceID)
			policies = append(policies, request.DevicePolicy)
		}
	case "set_global_policy":
		policies = append(policies, request.GlobalPolicy)
	}
	if len(incoming) > 0 && module.inventory == nil {
		return &models.DevicePolicyError{Code: "inventory_unavailable", Message: "无法确认分组成员所属网络"}
	}
	if len(incoming) > 0 && module.inventory != nil {
		inventory, err := module.inventory.Snapshot(ctx)
		if err != nil || inventory == nil || inventory.Result == nil {
			return &models.DevicePolicyError{Code: "inventory_unavailable", Message: "无法确认分组成员所属网络"}
		}
		known := map[string]bool{}
		for _, device := range inventory.Result.Devices {
			if device != nil {
				known[device.DeviceID] = true
			}
		}
		// Existing orphan references can be retained for cleanup, but never create a
		// new cross-network reference. Execution still resolves each scoped device.
		retained := map[string]bool{}
		if request.Action == "upsert_group" {
			for _, group := range current.Groups {
				if group.ID == request.Group.ID {
					for _, id := range group.Members {
						retained[id] = true
					}
				}
			}
		}
		if request.Action == "set_device_policy" {
			_, retained[request.DeviceID] = current.DevicePolicies[request.DeviceID]
		}
		for _, id := range incoming {
			if !known[id] && !retained[id] {
				return &models.DevicePolicyError{Code: "device_network_mismatch", Message: "分组成员不属于当前网络"}
			}
		}
	}
	if applier, ok := module.applier.(*defaultGroupPolicyApplier); ok && applier.network != nil && applier.network.gateway != nil {
		state, err := applier.network.gateway.readState(ctx)
		if err != nil {
			return &models.DevicePolicyError{Code: "route_unavailable", Message: "无法确认当前网络的上网路线"}
		}
		targets := buildGatewayTargetIndex(state)
		for _, policy := range policies {
			if policy != nil && policy.TargetID != "" && targets.targets[policy.TargetID] == nil {
				return &models.DevicePolicyError{Code: "route_network_mismatch", Message: "所选路线不属于当前网络"}
			}
		}
	}
	return nil
}

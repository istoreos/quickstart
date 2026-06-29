package interfacewrite

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	ReadSnapshot(ctx context.Context) (Snapshot, error)
	NormalizeDevices(name string, devices []string) []string
	ApplyPlan(ctx context.Context, plan CommandPlan) error
}

type Apply interface {
	Apply(ctx context.Context, configs []string) error
}

type Service struct {
	store Store
	apply Apply
}

func NewService(store Store, apply Apply) *Service {
	return &Service{store: store, apply: apply}
}

func (svc *Service) ApplyConfigSet(ctx context.Context, input Input) (*models.SDKNormalResponse, error) {
	snapshot, err := svc.store.ReadSnapshot(ctx)
	if err != nil {
		return nil, err
	}

	plan := CommandPlan{
		DeleteCommands: BuildDeleteCommands(BuildDeletePlan(snapshot.Interfaces, input.Configs)),
	}

	for _, cfg := range input.Configs {
		if cfg == nil {
			continue
		}
		normalized := svc.store.NormalizeDevices(cfg.Name, cfg.Devices)
		bridged := len(cfg.Devices) > 1 || cfg.Name == "lan"
		deviceName := ""
		if bridged {
			deviceName = "br-" + cfg.Name
			plan.BridgeCommands = append(plan.BridgeCommands, BuildBridgePlan(cfg, normalized, snapshot.DeviceSections))
		} else if len(normalized) > 0 {
			deviceName = normalized[0]
		}

		plan.InterfaceCommands = append(plan.InterfaceCommands, BuildInterfacePlan(cfg, deviceName, bridged))
		if firewallCmds := BuildFirewallBindingPlan(cfg, snapshot.FirewallZones); len(firewallCmds) > 0 {
			plan.FirewallCommands = append(plan.FirewallCommands, firewallCmds)
		}
	}

	if err := svc.store.ApplyPlan(ctx, plan); err != nil {
		return nil, err
	}
	if err := svc.apply.Apply(ctx, []string{"firewall", "network"}); err != nil {
		return nil, err
	}

	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}

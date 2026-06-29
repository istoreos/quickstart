package service

import (
	"context"
	"errors"
	"strings"
)

type LanSpeedLimitWriteService struct {
	store LanSpeedLimitWriteStore
	apply LanSpeedLimitApply
}

func NewDefaultLanSpeedLimitWriteService() *LanSpeedLimitWriteService {
	return NewLanSpeedLimitWriteService(
		NewDefaultLanSpeedLimitWriteStore(),
		NewDefaultLanSpeedLimitApply(),
	)
}

func NewLanSpeedLimitWriteService(store LanSpeedLimitWriteStore, apply LanSpeedLimitApply) *LanSpeedLimitWriteService {
	return &LanSpeedLimitWriteService{
		store: store,
		apply: apply,
	}
}

func (svc *LanSpeedLimitWriteService) UpsertSpeedLimitRule(ctx context.Context, input SpeedLimitWriteInput) error {
	if input.Action == "" {
		return errors.New("action is required")
	}
	if input.Action != "delete" && strings.TrimSpace(input.MAC) == "" {
		return errors.New("mac is required")
	}
	input.MAC = normalizeLanSpeedLimitedDeviceMAC(input.MAC)

	eqosMatches, firewallMatches, err := svc.store.ReadRuleMatches(ctx)
	if err != nil {
		return err
	}

	plan := BuildSpeedLimitWritePlan(input, eqosMatches, firewallMatches)
	if len(plan.DeleteSections) == 0 && !plan.AddSpeedLimit && !plan.AddBlockRule {
		return nil
	}
	if err := svc.store.ApplyPlan(ctx, plan); err != nil {
		return err
	}
	return svc.apply.Apply(ctx, []string{"eqos", "firewall"})
}

func (svc *LanSpeedLimitWriteService) SetSpeedLimitModule(ctx context.Context, input SpeedLimitModuleInput) error {
	if input.DownloadSpeed == 0 {
		input.DownloadSpeed = 2000
	}
	if input.UploadSpeed == 0 {
		input.UploadSpeed = 200
	}
	if err := svc.store.ApplyModuleConfig(ctx, input); err != nil {
		return err
	}
	return svc.apply.Apply(ctx, []string{"eqos"})
}

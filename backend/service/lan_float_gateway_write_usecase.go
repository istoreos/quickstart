package service

import "context"

type LanFloatGatewayWriteService struct {
	store LanFloatGatewayWriteStore
	apply LanFloatGatewayApply
}

type lanFloatGatewayWriteFacade interface {
	SetFloatGateway(ctx context.Context, input FloatGatewayWriteInput) error
}

func NewLanFloatGatewayWriteService(store LanFloatGatewayWriteStore, apply LanFloatGatewayApply) *LanFloatGatewayWriteService {
	return &LanFloatGatewayWriteService{
		store: store,
		apply: apply,
	}
}

func NewDefaultLanFloatGatewayWriteService() *LanFloatGatewayWriteService {
	return NewLanFloatGatewayWriteService(
		NewDefaultLanFloatGatewayWriteStore(),
		NewDefaultLanFloatGatewayApply(),
	)
}

func (svc *LanFloatGatewayWriteService) SetFloatGateway(ctx context.Context, input FloatGatewayWriteInput) error {
	state, tags, hosts, err := svc.store.ReadState(ctx)
	if err != nil {
		return err
	}

	plan := FloatGatewayWriteExecutionPlan{
		FloatCommands: buildFloatGatewayWriteCommands(input),
	}
	if shouldCleanupFloatGatewayDhcp(state, input) {
		plan.CleanupPlan = buildFloatGatewayDhcpCleanupPlan(tags, hosts)
	}

	if err := svc.store.ApplyPlan(ctx, plan); err != nil {
		return err
	}
	return svc.apply.Apply(ctx, []string{"floatip", "dhcp", "dnsmasq"})
}

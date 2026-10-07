package service

import (
	"context"
	"errors"
	"net/netip"
	"net/url"
	"strings"
)

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
	if err := validateLegacyFloatGatewayInput(input); err != nil {
		return err
	}
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

func validateLegacyFloatGatewayInput(input FloatGatewayWriteInput) error {
	if input.Role != "main" && input.Role != "fallback" {
		return errors.New("floating gateway role must be main or fallback")
	}
	for _, value := range []string{input.SetIP, input.CheckIP} {
		address, err := netip.ParseAddr(strings.TrimSpace(value))
		if err != nil || !address.Is4() {
			return errors.New("floating gateway addresses must be valid IPv4 addresses")
		}
	}
	if input.Role == "main" {
		if input.CheckURL != "" {
			parsed, err := url.ParseRequestURI(input.CheckURL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || !validSafeLabel(input.CheckURL, 2048) {
				return errors.New("floating gateway check URL must be an absolute HTTP or HTTPS URL")
			}
		}
		if input.CheckURLTimeout < 1 || input.CheckURLTimeout > 30 {
			return errors.New("floating gateway check timeout must be between 1 and 30 seconds")
		}
	}
	return nil
}

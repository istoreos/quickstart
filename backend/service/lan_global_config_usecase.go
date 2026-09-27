package service

import (
	"context"
	"errors"

	"github.com/istoreos/quickstart/backend/models"
)

type LanDhcpStateReader interface {
	LoadLanState(context.Context) (*LanDhcpState, error)
}

type LanGlobalConfigService struct {
	LanStatusReader  LanStatusReader
	DhcpStore        LanDhcpStateReader
	FloatIPReader    FloatIPReader
	SpeedLimitReader SpeedLimitReader
}

func NewLanGlobalConfigService() *LanGlobalConfigService {
	return &LanGlobalConfigService{
		LanStatusReader:  NewDefaultLanStatusReader(),
		DhcpStore:        NewDefaultDhcpConfigStore(),
		FloatIPReader:    NewDefaultFloatIPReader(),
		SpeedLimitReader: NewDefaultSpeedLimitReader(),
	}
}

func (svc *LanGlobalConfigService) GetGlobalConfigs(ctx context.Context) (*models.LANCtrlGlobalConfigResponse, error) {
	lanStatus, err := svc.LanStatusReader.ReadLanStatus(ctx)
	if err != nil {
		return nil, err
	}

	dhcpState, err := svc.DhcpStore.LoadLanState(ctx)
	if err != nil {
		dhcpState = &LanDhcpState{}
	}

	floatState, floatErr := svc.FloatIPReader.ReadFloatIPStatus(ctx)

	speedState, speedErr := svc.SpeedLimitReader.ReadSpeedLimitStatus(ctx)
	selectedProvider := effectiveRateLimitProvider(defaultRateLimitProviderPath)
	if selectedProvider == nativePolicyProviderName {
		installed, available, reason := nativeProviderCapability(ctx)
		speedState.Installed = installed
		speedState.Enabled = available
		if !available {
			speedErr = errors.New(reason)
		}
	} else if selectedProvider == "bandix" {
		installed, supported, _ := bandixProviderCapability()
		speedState.Installed = installed
		speedState.Enabled = installed && supported
		if !supported {
			speedErr = errors.New("kernel_not_supported")
		}
	}

	plan := BuildAutoDhcpPlan(lanStatus, dhcpState)

	return &models.LANCtrlGlobalConfigResponse{
		Result: &models.LANCtrlGlobalConfig{
			Capabilities: buildDeviceManagementCapabilities(floatState, floatErr, speedState, speedErr),
			DhcpTags:     buildGlobalDhcpTags(lanStatus, dhcpState),
			DhcpGlobal:   buildDhcpGlobalConfig(lanStatus, plan),
			FloatGateway: toFloatGatewayModel(floatState),
			SpeedLimit:   toSpeedLimitModel(speedState),
		},
	}, nil
}

func buildDeviceManagementCapabilities(floatState FloatIPStatus, floatErr error, speedState SpeedLimitStatus, speedErr error) *models.DeviceManagementCapabilities {
	access := &models.DeviceManagementCapability{State: "available"}
	speed := buildDeviceManagementCapability(speedState.Installed, speedState.Enabled, speedErr)
	decorateCapabilityActions(speed, "app-meta-eqos")
	floating := buildDeviceManagementCapability(floatState.Installed, floatState.Enabled, floatErr)
	decorateCapabilityActions(floating, "app-meta-floatip")
	traffic := &models.DeviceManagementCapability{State: "available"}
	items := map[string]*models.Capability{
		"internet_access": access, "device_speed_limit": speed, "floating_gateway": floating, "traffic_insights": traffic,
	}
	return &models.DeviceManagementCapabilities{Items: items, InternetAccess: access, FloatGateway: floating, SpeedLimit: speed, TrafficInsights: traffic}
}

func buildDeviceManagementCapability(installed, enabled bool, err error) *models.DeviceManagementCapability {
	capability := &models.DeviceManagementCapability{}
	switch {
	case err != nil:
		capability.State = "error"
		capability.Reason = "status_unavailable"
	case !installed:
		capability.State = "not_installed"
		capability.Reason = "dependency_not_installed"
	case !enabled:
		capability.State = "disabled"
	default:
		capability.State = "available"
	}
	return capability
}

func decorateCapabilityActions(capability *models.DeviceManagementCapability, installTarget string) {
	if capability == nil {
		return
	}
	switch capability.State {
	case "not_installed":
		capability.Actions = []*models.CapabilityAction{{Kind: "install", Target: installTarget, RequiresConfirmation: true}}
	case "disabled":
		capability.Actions = []*models.CapabilityAction{{Kind: "enable", RequiresConfirmation: true}}
	case "error":
		capability.Actions = []*models.CapabilityAction{{Kind: "retry"}}
	}
}

func buildGlobalDhcpTags(lanStatus LanStatusSnapshot, state *LanDhcpState) []*models.LANCtrlDhcpTagInfo {
	plan := BuildAutoDhcpPlan(lanStatus, state)
	dhcpTags := append([]*models.LANCtrlDhcpTagInfo(nil), toModelDhcpTags(plan.Tags)...)

	if state == nil {
		return dhcpTags
	}

	if state.FloatIP != nil && state.FloatIP.Enabled {
		tagName := ipToDhcpTag(state.FloatIP.SetIP)
		if tagName != "" && !hasDhcpTag(dhcpTags, tagName) {
			dhcpTags = append(dhcpTags, &models.LANCtrlDhcpTagInfo{
				TagTitle:    "floatip",
				TagName:     tagName,
				AutoCreated: true,
				Gateway:     state.FloatIP.SetIP,
				DhcpOption:  []string{"3," + state.FloatIP.SetIP, "6," + state.FloatIP.SetIP},
			})
		}

		tagName = ipToDhcpTag(state.FloatIP.CheckIP)
		if tagName != "" {
			dhcpTags = append(dhcpTags, &models.LANCtrlDhcpTagInfo{
				TagTitle:    "bypass",
				TagName:     tagName,
				AutoCreated: true,
				Gateway:     state.FloatIP.CheckIP,
				DhcpOption:  []string{"3," + state.FloatIP.CheckIP, "6," + state.FloatIP.CheckIP},
			})
		}
	}

	for _, tag := range state.Tags {
		if hasDhcpTag(dhcpTags, tag.TagName) {
			continue
		}
		dhcpTags = append(dhcpTags, toModelDhcpTags([]DhcpTagRecord{tag})...)
	}

	return dhcpTags
}

func buildDhcpGlobalConfig(lanStatus LanStatusSnapshot, plan DhcpTagPlan) *models.LANDhcpGlobalConfig {
	dhcpGateway := plan.DhcpGateway
	if dhcpGateway == lanStatus.LanAddr {
		dhcpGateway = ""
	}

	config := &models.LANDhcpGlobalConfig{
		DhcpEnabled: plan.DhcpEnabled,
		DhcpGateway: dhcpGateway,
		GatewaySels: make([]*models.LANDhcpGatewaySel, 0, 2),
	}

	config.GatewaySels = append(config.GatewaySels, &models.LANDhcpGatewaySel{
		Title:   "myself",
		Gateway: "",
	})

	if dhcpGateway == "" || dhcpGateway == lanStatus.Nexthop {
		if lanStatus.Nexthop != "" {
			config.GatewaySels = append(config.GatewaySels, &models.LANDhcpGatewaySel{
				Title:   "parent",
				Gateway: lanStatus.Nexthop,
			})
		}
		return config
	}

	config.GatewaySels = append(config.GatewaySels, &models.LANDhcpGatewaySel{
		Gateway: dhcpGateway,
	})
	return config
}

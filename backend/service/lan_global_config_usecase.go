package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
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

	floatState, err := svc.FloatIPReader.ReadFloatIPStatus(ctx)
	if err != nil {
		return nil, err
	}

	speedState, err := svc.SpeedLimitReader.ReadSpeedLimitStatus(ctx)
	if err != nil {
		return nil, err
	}

	plan := BuildAutoDhcpPlan(lanStatus, dhcpState)

	return &models.LANCtrlGlobalConfigResponse{
		Result: &models.LANCtrlGlobalConfig{
			DhcpTags:     buildGlobalDhcpTags(lanStatus, dhcpState),
			DhcpGlobal:   buildDhcpGlobalConfig(lanStatus, plan),
			FloatGateway: toFloatGatewayModel(floatState),
			SpeedLimit:   toSpeedLimitModel(speedState),
		},
	}, nil
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

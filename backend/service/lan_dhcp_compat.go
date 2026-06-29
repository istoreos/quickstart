package service

import (
	"context"
	"fmt"
	"net"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

// Get DHCP info, only support the one LAN, not support LAN2, LAN3...
// TODO bug at uci.GetLast may return old value or not return value if revert in Golang
func getDhcpOfLAN(ctx context.Context, lanStatus *ubusLanStatus) (dhcpTags []*models.LANCtrlDhcpTagInfo, dhcpGateway string, isDhcpIgnore bool) {
	dhcpTags = make([]*models.LANCtrlDhcpTagInfo, 0)
	state, err := NewDefaultDhcpConfigStore().LoadLanState(ctx)
	if err != nil {
		state = &LanDhcpState{}
	}

	plan := BuildAutoDhcpPlan(
		LanStatusSnapshot{
			LanAddr:          lanStatus.lanAddr,
			Nexthop:          lanStatus.nexthop,
			IsDefaultGateway: lanStatus.isDefaultGateway,
		},
		state,
	)
	dhcpTags = append(dhcpTags, toModelDhcpTags(plan.Tags)...)
	dhcpGateway = plan.DhcpGateway
	isDhcpIgnore = state.DhcpIgnore

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
				AutoCreated: true,
				TagName:     tagName,
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

	return
}

func hasDhcpTag(dhcpTags []*models.LANCtrlDhcpTagInfo, tag string) bool {
	for _, dhcpTag := range dhcpTags {
		if tag == dhcpTag.TagName {
			return true
		}
	}
	return false
}

func ipToDhcpTag(ipstr string) string {
	if ipstr == "" {
		return ""
	}
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return ""
	}
	return fmt.Sprintf("t_auto_%x", utils.Ipv4ToLong(ip))
}

func DhcpTagsConfig(ctx context.Context, input DhcpTagConfigInput) error {
	svc := NewLanDhcpService(NewDefaultDhcpConfigStore(), NewDefaultLanStatusReader())
	return svc.SetDhcpTags(ctx, input)
}

func DhcpGatewayConfig(ctx context.Context, input DhcpGatewayInput) error {
	svc := NewLanDhcpService(NewDefaultDhcpConfigStore(), NewDefaultLanStatusReader())
	return svc.SetDhcpGateway(ctx, input)
}

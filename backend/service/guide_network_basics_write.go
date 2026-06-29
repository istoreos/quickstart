package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	networkbasics "github.com/linkease/quick-start/istore-backend/modules/guidecore/networkbasics"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideSetDNSConfigInput struct {
	InterfaceName string
	DNSProto      string
	ManualDNSIP   []string
}

type GuideSetWANInterfaceInput struct {
	InterfaceName string
	WanProto      string
	StaticIP      string
	SubnetMask    string
	Gateway       string
}

type GuideSetPPPoEInput struct {
	Account  string
	Password string
}

type GuideSetLANConfigInput struct {
	LanIP      string
	NetMask    string
	EnableDhcp bool
	DhcpStart  string
	DhcpEnd    string
}

type GuideNetworkBasicsWriter interface {
	SetDNSConfig(ctx context.Context, input GuideSetDNSConfigInput) error
	SetWANInterfaceMode(ctx context.Context, input GuideSetWANInterfaceInput) error
	SetPPPoE(ctx context.Context, input GuideSetPPPoEInput) error
	SetLANConfig(ctx context.Context, input GuideSetLANConfigInput) ([]string, error)
}

type GuideNetworkBasicsApply interface {
	Apply(ctx context.Context, pending []string) error
}

var writeGuideNetworkBasicsDNS = func(ctx context.Context, it string, dnsProto string, dnsIPs []string) error {
	return uciSetDNSWithoutCommit(ctx, it, dnsProto, dnsIPs)
}

var writeGuideNetworkBasicsInterface = func(ctx context.Context, input GuideSetWANInterfaceInput) error {
	return uciSetInterfaceWithoutCommit(ctx, input.InterfaceName, input.WanProto, input.SubnetMask, input.StaticIP, input.Gateway)
}

var writeGuideNetworkBasicsPPPoE = func(ctx context.Context, input GuideSetPPPoEInput) error {
	return uciSetPppoeWithoutCommit(ctx, input.Account, input.Password)
}

var writeGuideNetworkBasicsLAN = func(input GuideSetLANConfigInput) error {
	return LanSetting(input.LanIP, input.NetMask, true)
}

var writeGuideNetworkBasicsLanDHCPRange = func(ctx context.Context, start int, limit int) error {
	return utils.BatchRun(ctx, networkbasics.BuildLanDHCPRangeCommands(start, limit), 0)
}

var writeGuideNetworkBasicsBatchRun = func(ctx context.Context, cmdList []string) error {
	return utils.BatchRun(ctx, cmdList, 0)
}

var writeGuideNetworkBasicsEnableLanDHCP = func(ctx context.Context) {
	enabledLanDHCPServer(ctx)
}

var writeGuideNetworkBasicsSetLanMasq = func(ctx context.Context, enable bool) bool {
	return uciSetLanMasq(ctx, enable)
}

var writeGuideNetworkBasicsDeleteLan6 = func(ctx context.Context) error {
	return writeGuideNetworkBasicsBatchRun(ctx, []string{"uci -q delete network.lan6 || true"})
}

var applyGuideNetworkBasicsPending = func(ctx context.Context, pending []string) error {
	return utils.UciCommitAndApply(ctx, pending)
}

type defaultGuideNetworkBasicsWriter struct{}

func newDefaultGuideNetworkBasicsWriter() *defaultGuideNetworkBasicsWriter {
	return &defaultGuideNetworkBasicsWriter{}
}

func (writer *defaultGuideNetworkBasicsWriter) SetDNSConfig(ctx context.Context, input GuideSetDNSConfigInput) error {
	return writeGuideNetworkBasicsDNS(ctx, input.InterfaceName, input.DNSProto, input.ManualDNSIP)
}

func (writer *defaultGuideNetworkBasicsWriter) SetWANInterfaceMode(ctx context.Context, input GuideSetWANInterfaceInput) error {
	return writeGuideNetworkBasicsInterface(ctx, input)
}

func (writer *defaultGuideNetworkBasicsWriter) SetPPPoE(ctx context.Context, input GuideSetPPPoEInput) error {
	return writeGuideNetworkBasicsPPPoE(ctx, input)
}

func (writer *defaultGuideNetworkBasicsWriter) SetLANConfig(ctx context.Context, input GuideSetLANConfigInput) ([]string, error) {
	if input.EnableDhcp {
		startInt, limitInt, err := utils.CalcStartAndLimit(input.DhcpStart, input.DhcpEnd, input.NetMask)
		if err != nil {
			return nil, err
		}
		if err := writeGuideNetworkBasicsLanDHCPRange(ctx, startInt, limitInt); err != nil {
			return nil, err
		}
	}
	if err := writeGuideNetworkBasicsLAN(input); err != nil {
		return nil, err
	}
	pending := []string{}
	if input.EnableDhcp {
		writeGuideNetworkBasicsEnableLanDHCP(ctx)
		pending = append(pending, "dhcp")
	}
	pending = append(pending, "network")
	return pending, nil
}

type defaultGuideNetworkBasicsApply struct{}

func newDefaultGuideNetworkBasicsApply() *defaultGuideNetworkBasicsApply {
	return &defaultGuideNetworkBasicsApply{}
}

func (apply *defaultGuideNetworkBasicsApply) Apply(ctx context.Context, pending []string) error {
	return applyGuideNetworkBasicsPending(ctx, pending)
}

func buildGuideNetworkBasicsPendingForWANMode(req interface{}, includeMasq bool) []string {
	enableLanDhcp := false
	switch typed := req.(type) {
	case models.GuideClientModeRequest:
		if typed.EnableLanDhcp {
			enableLanDhcp = true
		}
	case models.GuidePppoeRequest:
		if typed.EnableLanDhcp {
			enableLanDhcp = true
		}
	}
	return networkbasics.BuildWANModePendingConfigs(includeMasq, enableLanDhcp)
}

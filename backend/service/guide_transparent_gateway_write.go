package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideTransparentGatewayWriter interface {
	SetDHCP(ctx context.Context, enable bool) error
	SetInterface(ctx context.Context, staticIP string, subnetMask string, gateway string) error
	SetDNS(ctx context.Context, dnsIP string) error
	SetLan6(ctx context.Context, enable bool) error
	SetNat(ctx context.Context, enable bool) bool
}

type GuideTransparentGatewayApply interface {
	Apply(ctx context.Context, pending []string) error
}

var writeGuideTransparentGatewayBatchRun = func(ctx context.Context, cmdList []string) error {
	return utils.BatchRun(ctx, cmdList, 0)
}

var writeGuideTransparentGatewayInterface = func(ctx context.Context, staticIP string, subnetMask string, gateway string) error {
	return uciSetInterfaceWithoutCommit(ctx, "lan", "static", subnetMask, staticIP, gateway)
}

var writeGuideTransparentGatewayDNS = func(ctx context.Context, dnsIP string) error {
	return uciSetDNSWithoutCommit(ctx, "lan", "manual", []string{dnsIP})
}

var writeGuideTransparentGatewayNat = func(ctx context.Context, enable bool) bool {
	return uciSetLanMasq(ctx, enable)
}

var applyGuideTransparentGatewayPending = func(ctx context.Context, pending []string) error {
	return utils.UciCommitAndApply(ctx, pending)
}

type defaultGuideTransparentGatewayWriter struct{}

func newDefaultGuideTransparentGatewayWriter() *defaultGuideTransparentGatewayWriter {
	return &defaultGuideTransparentGatewayWriter{}
}

func (writer *defaultGuideTransparentGatewayWriter) SetDHCP(ctx context.Context, enable bool) error {
	if enable {
		return writeGuideTransparentGatewayBatchRun(ctx, []string{
			"uci -q del dhcp.lan.ignore || true",
			"uci set dhcp.lan.dhcpv4=server",
			"uci set dhcp.lan.start=100",
			"uci set dhcp.lan.limit=150",
			"uci set dhcp.lan.leasetime=12h",
		})
	}
	return writeGuideTransparentGatewayBatchRun(ctx, []string{
		"uci set dhcp.lan.ignore=1",
	})
}

func (writer *defaultGuideTransparentGatewayWriter) SetInterface(ctx context.Context, staticIP string, subnetMask string, gateway string) error {
	return writeGuideTransparentGatewayInterface(ctx, staticIP, subnetMask, gateway)
}

func (writer *defaultGuideTransparentGatewayWriter) SetDNS(ctx context.Context, dnsIP string) error {
	return writeGuideTransparentGatewayDNS(ctx, dnsIP)
}

func (writer *defaultGuideTransparentGatewayWriter) SetLan6(ctx context.Context, enable bool) error {
	if enable {
		return writeGuideTransparentGatewayBatchRun(ctx, []string{
			"uci -q batch <<-EOF >/dev/null",
			"set network.lan6=interface",
			"set network.lan6.proto=dhcpv6",
			"set network.lan6.device=@lan",
			"EOF",
			"",
		})
	}
	return writeGuideTransparentGatewayBatchRun(ctx, []string{
		"uci -q delete network.lan6 || true",
	})
}

func (writer *defaultGuideTransparentGatewayWriter) SetNat(ctx context.Context, enable bool) bool {
	return writeGuideTransparentGatewayNat(ctx, enable)
}

type defaultGuideTransparentGatewayApply struct{}

func newDefaultGuideTransparentGatewayApply() *defaultGuideTransparentGatewayApply {
	return &defaultGuideTransparentGatewayApply{}
}

func (apply *defaultGuideTransparentGatewayApply) Apply(ctx context.Context, pending []string) error {
	return applyGuideTransparentGatewayPending(ctx, pending)
}

func buildGuideTransparentGatewayPending(req models.GuideGatewayRouterRequest, includeFirewall bool) []string {
	pending := []string{}
	if includeFirewall {
		pending = append(pending, "firewall")
	}
	pending = append(pending, "dhcp", "network")
	return pending
}

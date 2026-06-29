package service

import (
	"context"

	"github.com/digineo/go-uci"
)

type GuideTransparentGatewayReader interface {
	ReadTransparentGateway(ctx context.Context) *GuideTransparentGatewaySnapshot
}

var readGuideTransparentGatewayIPAndMask = func(net string) (string, string) {
	return uciGetIPAndMask(net)
}

var readGuideTransparentGatewayConfigLast = func(config, section, option string) (string, bool) {
	uci.LoadConfig(config, true)
	return uci.GetLast(config, section, option)
}

var readGuideTransparentGatewayLanDHCPEnabled = func() bool {
	return isLanDHCPServerEnabled()
}

type defaultGuideTransparentGatewayReader struct{}

func newDefaultGuideTransparentGatewayReader() *defaultGuideTransparentGatewayReader {
	return &defaultGuideTransparentGatewayReader{}
}

func (reader *defaultGuideTransparentGatewayReader) ReadTransparentGateway(ctx context.Context) *GuideTransparentGatewaySnapshot {
	snapshot := &GuideTransparentGatewaySnapshot{}
	snapshot.StaticLanIP, snapshot.SubnetMask = readGuideTransparentGatewayIPAndMask("lan")
	if value, ok := readGuideTransparentGatewayConfigLast("network", "lan", "gateway"); ok {
		snapshot.Gateway = value
	}
	if value, ok := readGuideTransparentGatewayConfigLast("network", "lan", "dns"); ok {
		snapshot.StaticDNSIP = value
	}
	snapshot.EnableDhcp = readGuideTransparentGatewayLanDHCPEnabled()
	return snapshot
}

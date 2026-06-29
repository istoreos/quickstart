package service

import (
	"context"
	"fmt"

	"github.com/digineo/go-uci"
)

type GuideNetworkBasicsReader interface {
	ReadDefaultOutboundInterface(ctx context.Context) (*GuideDefaultOutboundInterfaceSnapshot, error)
	ReadDNSConfig(ctx context.Context) (*GuideDNSConfigSnapshot, error)
	ReadWANConfig(ctx context.Context) *GuideWANConfigSnapshot
	ReadWANRuntime(ctx context.Context, interfaceName string) (*GuideWANRuntimeSnapshot, error)
	ReadLANConfig(ctx context.Context) *GuideLANConfigSnapshot
}

var readGuideNetworkBasicsOutboundInterface = func() (*DefaultInterface, error) {
	return outboundInterface()
}

var readGuideNetworkBasicsNetDNSClient = func(net string) (string, []string) {
	return uciGetNetDNSClient(net)
}

var readGuideNetworkBasicsIPAndMask = func(net string) (string, string) {
	return uciGetIPAndMask(net)
}

var readGuideNetworkBasicsLanDHCPEnabled = func() bool {
	return isLanDHCPServerEnabled()
}

var readGuideNetworkBasicsConfigLast = func(config, section, option string) (string, bool) {
	uci.LoadConfig(config, true)
	return uci.GetLast(config, section, option)
}

var readGuideNetworkBasicsWANStatus = func(ctx context.Context, interfaceName string, blk *ubusNetworkInterfaceStatus) error {
	return UbusCallWithObject(ctx, fmt.Sprintf("network.interface.%s status", interfaceName), blk)
}

type defaultGuideNetworkBasicsReader struct{}

func newDefaultGuideNetworkBasicsReader() *defaultGuideNetworkBasicsReader {
	return &defaultGuideNetworkBasicsReader{}
}

func (reader *defaultGuideNetworkBasicsReader) ReadDefaultOutboundInterface(ctx context.Context) (*GuideDefaultOutboundInterfaceSnapshot, error) {
	defaultIf, err := readGuideNetworkBasicsOutboundInterface()
	if err != nil {
		return nil, err
	}
	return &GuideDefaultOutboundInterfaceSnapshot{
		InterfaceName: defaultIf.interfaceName,
		DeviceName:    defaultIf.deviceName,
		Proto:         defaultIf.proto,
	}, nil
}

func (reader *defaultGuideNetworkBasicsReader) ReadDNSConfig(ctx context.Context) (*GuideDNSConfigSnapshot, error) {
	defaultIf, err := reader.ReadDefaultOutboundInterface(ctx)
	if err != nil {
		return nil, err
	}
	dnsProto, manualDNSIP := readGuideNetworkBasicsNetDNSClient(defaultIf.InterfaceName)
	return &GuideDNSConfigSnapshot{
		InterfaceName: defaultIf.InterfaceName,
		DNSProto:      dnsProto,
		ManualDNSIP:   manualDNSIP,
	}, nil
}

func (reader *defaultGuideNetworkBasicsReader) ReadWANConfig(ctx context.Context) *GuideWANConfigSnapshot {
	snapshot := &GuideWANConfigSnapshot{}
	if value, ok := readGuideNetworkBasicsConfigLast("network", "wan", "proto"); ok {
		snapshot.Exists = value != ""
		snapshot.WanProto = value
	}
	snapshot.StaticIP, snapshot.SubnetMask = readGuideNetworkBasicsIPAndMask("wan")
	if value, ok := readGuideNetworkBasicsConfigLast("network", "wan", "gateway"); ok {
		snapshot.Gateway = value
	}
	snapshot.DNSProto, snapshot.ManualDNSIP = readGuideNetworkBasicsNetDNSClient("wan")
	if value, ok := readGuideNetworkBasicsConfigLast("network", "wan", "username"); ok {
		snapshot.PPPoEAccount = value
	}
	if value, ok := readGuideNetworkBasicsConfigLast("network", "wan", "password"); ok {
		snapshot.PPPoEPassword = value
	}
	return snapshot
}

func (reader *defaultGuideNetworkBasicsReader) ReadWANRuntime(ctx context.Context, interfaceName string) (*GuideWANRuntimeSnapshot, error) {
	var blk ubusNetworkInterfaceStatus
	if err := readGuideNetworkBasicsWANStatus(ctx, interfaceName, &blk); err != nil {
		return nil, err
	}
	result := &GuideWANRuntimeSnapshot{}
	if len(blk.Ipv4) > 0 {
		result.StaticIP = blk.Ipv4[0].Address
		result.SubnetMask = LenToSubNetMask(blk.Ipv4[0].Mask)
	}
	for _, route := range blk.Route {
		if route.Target == "0.0.0.0" && route.Mask == 0 {
			result.Gateway = route.Nexthop
			break
		}
	}
	return result, nil
}

func (reader *defaultGuideNetworkBasicsReader) ReadLANConfig(ctx context.Context) *GuideLANConfigSnapshot {
	snapshot := &GuideLANConfigSnapshot{}
	snapshot.LanIP, snapshot.NetMask = readGuideNetworkBasicsIPAndMask("lan")
	snapshot.EnableDhcp = readGuideNetworkBasicsLanDHCPEnabled()
	if snapshot.EnableDhcp {
		startStr, _ := readGuideNetworkBasicsConfigLast("dhcp", "lan", "start")
		limitStr, _ := readGuideNetworkBasicsConfigLast("dhcp", "lan", "limit")
		snapshot.DhcpStart, snapshot.DhcpEnd = buildGuideNetworkBasicsLANRange(snapshot.LanIP, startStr, limitStr)
	}
	return snapshot
}

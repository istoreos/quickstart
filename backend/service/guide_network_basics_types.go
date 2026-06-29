package service

import networkbasics "github.com/linkease/quick-start/istore-backend/modules/guidecore/networkbasics"

type GuideDefaultOutboundInterfaceSnapshot struct {
	InterfaceName string
	DeviceName    string
	Proto         string
}

type GuideDNSConfigSnapshot struct {
	InterfaceName string
	DNSProto      string
	ManualDNSIP   []string
}

type GuideWANConfigSnapshot struct {
	Exists        bool
	WanProto      string
	StaticIP      string
	SubnetMask    string
	Gateway       string
	DNSProto      string
	ManualDNSIP   []string
	PPPoEAccount  string
	PPPoEPassword string
}

type GuideWANRuntimeSnapshot struct {
	StaticIP   string
	SubnetMask string
	Gateway    string
}

type GuideLANConfigSnapshot struct {
	LanIP      string
	NetMask    string
	EnableDhcp bool
	DhcpStart  string
	DhcpEnd    string
}

func buildGuideNetworkBasicsLANRange(lanIP string, startStr string, limitStr string) (string, string) {
	return networkbasics.BuildLANRange(lanIP, startStr, limitStr)
}

package service

type GuideTransparentGatewaySnapshot struct {
	StaticLanIP string
	SubnetMask  string
	Gateway     string
	StaticDNSIP string
	EnableDhcp  bool
}

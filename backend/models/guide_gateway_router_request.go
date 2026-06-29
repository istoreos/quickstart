package models

// swagger:model guideGatewayRouterRequest
type GuideGatewayRouterRequest struct {

	// DHCPv6客户端
	Dhcp6c bool `json:"dhcp6c,omitempty"`

	// 是否开启dhcp
	EnableDhcp bool `json:"enableDhcp,omitempty"`

	// 是否开启NAT
	EnableNat bool `json:"enableNat,omitempty"`

	// 网关地址
	Gateway string `json:"gateway,omitempty"`

	// DNS服务器IP
	StaticDNSIP string `json:"staticDnsIp,omitempty"`

	// 静态IP地址
	StaticLanIP string `json:"staticLanIp,omitempty"`

	// 子网掩码
	SubnetMask string `json:"subnetMask,omitempty"`
}

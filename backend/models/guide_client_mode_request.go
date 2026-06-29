package models

// swagger:model guideClientModeRequest
type GuideClientModeRequest struct {

	// DNS 配置方式
	// Enum: [manual auto]
	DNSProto string `json:"dnsProto,omitempty"`

	// 启用LAN口的DHCP服务，用于从旁路由模式恢复
	EnableLanDhcp bool `json:"enableLanDhcp,omitempty"`

	// gateway
	Gateway string `json:"gateway,omitempty"`

	// manual Dns Ip
	ManualDNSIP []string `json:"manualDnsIp"`

	// 静态IP地址
	StaticIP string `json:"staticIp,omitempty"`

	// 子网掩码
	SubnetMask string `json:"subnetMask,omitempty"`

	// WAN 接口配置方式
	// Enum: [static dhcp]
	WanProto string `json:"wanProto,omitempty"`
}

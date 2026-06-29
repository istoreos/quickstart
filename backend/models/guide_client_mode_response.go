package models

// swagger:model guideClientModeResponse
type GuideClientModeResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideClientModeResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideClientModeResponseResult
type GuideClientModeResponseResult struct {

	// DNS 配置方式
	// Enum: [static dhcp]
	DNSProto string `json:"dnsProto,omitempty"`

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

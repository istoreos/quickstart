package models

// swagger:model networkStatusResponse
type NetworkStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NetworkStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NetworkStatusResponseResult
type NetworkStatusResponseResult struct {

	// default interface
	DefaultInterface string `json:"defaultInterface,omitempty"`

	// dns list
	DNSList []string `json:"dnsList"`

	// DNS 配置方式
	// Enum: [manual auto]
	DNSProto string `json:"dnsProto,omitempty"`

	// 网关 地址
	Gateway string `json:"gateway,omitempty"`

	// ipv4 地址
	Ipv4addr string `json:"ipv4addr,omitempty"`

	// ipv4 掩码长度
	Ipv4mask int32 `json:"ipv4mask,omitempty"`

	// ipv6 地址
	Ipv6addr string `json:"ipv6addr,omitempty"`

	// 网络状态
	// Enum: [netDetecting netSuccess dnsFailed netFailed softSourceFailed]
	NetworkInfo string `json:"networkInfo,omitempty"`

	// 工作模式
	// Enum: [pppoe static dhcp]
	Proto string `json:"proto,omitempty"`

	// 在线时间，eg 5h24m33s
	Uptime string `json:"uptime,omitempty"`

	// 在线时间, 毫秒
	UptimeStamp int64 `json:"uptimeStamp,omitempty"`
}

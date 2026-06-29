package models

// swagger:model guideDnsConfigResponse
type GuideDNSConfigResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideDNSConfigResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideDNSConfigResponseResult
type GuideDNSConfigResponseResult struct {

	// DNS 配置方式
	// Enum: [manual auto]
	DNSProto string `json:"dnsProto,omitempty"`

	// interface name
	InterfaceName string `json:"interfaceName,omitempty"`

	// manual Dns Ip
	ManualDNSIP []string `json:"manualDnsIp"`
}

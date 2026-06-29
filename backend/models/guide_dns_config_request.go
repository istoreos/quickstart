package models

// swagger:model guideDnsConfigRequest
type GuideDNSConfigRequest struct {

	// DNS 配置方式
	// Enum: [manual auto]
	DNSProto string `json:"dnsProto,omitempty"`

	// interface name
	InterfaceName string `json:"interfaceName,omitempty"`

	// manual Dns Ip
	ManualDNSIP []string `json:"manualDnsIp"`
}

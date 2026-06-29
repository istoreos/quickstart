package models

// swagger:model guideDdnsRequest
type GuideDdnsRequest struct {

	// 域名
	Domain string `json:"domain,omitempty"`

	// ipv4 or ipv6
	IPVersion string `json:"ipVersion,omitempty"`

	// 密码
	Password string `json:"password,omitempty"`

	// 服务提供商
	// Enum: [ali dnspod oray]
	ServiceName string `json:"serviceName,omitempty"`

	// 用户名
	UserName string `json:"userName,omitempty"`
}

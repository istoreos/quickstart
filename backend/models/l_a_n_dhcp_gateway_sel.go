package models

// swagger:model lANDhcpGatewaySel
type LANDhcpGatewaySel struct {

	// 具体的网关IP
	Gateway string `json:"gateway,omitempty"`

	// 显示标题。比如 default: 默认网关 parent: 上级路由 myself: 本设备 bypass: 旁路由 floatip: 浮动网关
	Title string `json:"title,omitempty"`
}

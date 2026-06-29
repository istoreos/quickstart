package models

// swagger:model guidePppoeRequest
type GuidePppoeRequest struct {

	// 拨号账号
	Account string `json:"account,omitempty"`

	// 启用LAN口的DHCP服务，用于从旁路由模式恢复
	EnableLanDhcp bool `json:"enableLanDhcp,omitempty"`

	// 拨号密码
	Password string `json:"password,omitempty"`
}

package models

// swagger:model nasSambaCreateRequest
type NasSambaCreateRequest struct {

	// 允许旧协议
	AllowLegacy bool `json:"allowLegacy,omitempty"`

	// 密码
	Password string `json:"password,omitempty"`

	// 服务根目录
	RootPath string `json:"rootPath,omitempty"`

	// 共享名
	ShareName string `json:"shareName,omitempty"`

	// 用户名
	Username string `json:"username,omitempty"`
}

package models

// swagger:model shareProtocolSambaConfig
type ShareProtocolSambaConfig struct {

	// 允许旧协议与身份验证(不安全)
	AllowLegacy bool `json:"allowLegacy,omitempty"`

	// 描述
	Description string `json:"description,omitempty"`

	// 禁用 Netbios
	DisableNetbios bool `json:"disableNetbios,omitempty"`

	// 启用 macOS 兼容共享
	EnableMacosCompatible bool `json:"enableMacosCompatible,omitempty"`

	// 工作组
	Workgroup string `json:"workgroup,omitempty"`
}

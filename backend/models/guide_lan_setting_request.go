package models

// swagger:model guideLanSettingRequest
type GuideLanSettingRequest struct {

	// 结束地址
	DhcpEnd string `json:"dhcpEnd,omitempty"`

	// 起始地址
	DhcpStart string `json:"dhcpStart,omitempty"`

	// 是否修改dhcp服务器设置
	EnableDhcp bool `json:"enableDhcp,omitempty"`

	// 内网地址
	LanIP string `json:"lanIp,omitempty"`

	// 子网掩码
	NetMask string `json:"netMask,omitempty"`
}

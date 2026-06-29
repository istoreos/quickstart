package models

// swagger:model guideLanSettingResponse
type GuideLanSettingResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideLanSettingResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideLanSettingResponseResult
type GuideLanSettingResponseResult struct {

	// 结束地址
	DhcpEnd string `json:"dhcpEnd,omitempty"`

	// 起始地址
	DhcpStart string `json:"dhcpStart,omitempty"`

	// dhcp服务已启用
	EnableDhcp bool `json:"enableDhcp,omitempty"`

	// 内网地址
	LanIP string `json:"lanIp,omitempty"`

	// 子网掩码
	NetMask string `json:"netMask,omitempty"`
}

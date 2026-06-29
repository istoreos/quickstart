package models

// swagger:model lANCtrlDhcpTagInfo
type LANCtrlDhcpTagInfo struct {

	// 系统自动生成，不能删除
	AutoCreated bool `json:"autoCreated,omitempty"`

	// ['3,192.168.100.3','6,192.168.100.3']
	DhcpOption []string `json:"dhcpOption"`

	// 如果是咱们自定义的设备，则返回网关信息
	Gateway string `json:"gateway,omitempty"`

	// 比如 LAN, LAN2, LAN3 等局域网接口
	LanName string `json:"lanName,omitempty"`

	// 标签值，提交的时候用这个值
	TagName string `json:"tagName,omitempty"`

	// 展示标题，需要在前段把英文翻译成中文
	TagTitle string `json:"tagTitle,omitempty"`
}

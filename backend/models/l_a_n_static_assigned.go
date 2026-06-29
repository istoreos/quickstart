package models

// swagger:model lANStaticAssigned
type LANStaticAssigned struct {

	// modify|delete|add
	Action string `json:"action,omitempty"`

	// assigned IP
	AssignedIP string `json:"assignedIP,omitempty"`

	// assigned mac
	AssignedMac string `json:"assignedMac,omitempty"`

	// 请求数据的时候有用，是否启用 mac/ip 绑定
	BindIP bool `json:"bindIP,omitempty"`

	// 网关信息
	DhcpGateway string `json:"dhcpGateway,omitempty"`

	// hostname
	Hostname string `json:"hostname,omitempty"`

	// 标签名
	TagName string `json:"tagName,omitempty"`

	// 标签名字，如果标签名字为空，则展示标签值。需要前端把标签英文名字变成中文
	TagTitle string `json:"tagTitle,omitempty"`
}

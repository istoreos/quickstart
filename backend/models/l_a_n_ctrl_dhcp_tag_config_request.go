package models

// swagger:model lANCtrlDhcpTagConfigRequest
type LANCtrlDhcpTagConfigRequest struct {

	// modify|delete|add
	Action string `json:"action,omitempty"`

	// ['3,192.168.100.3','6,192.168.100.3']
	DhcpOption []string `json:"dhcpOption"`

	// 标签名
	TagName string `json:"tagName,omitempty"`

	// 标签
	TagTitle string `json:"tagTitle,omitempty"`
}

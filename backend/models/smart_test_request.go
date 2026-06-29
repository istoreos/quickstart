package models

// swagger:model SmartTestRequest
type SmartTestRequest struct {

	// 磁盘路径
	DevicePath string `json:"devicePath,omitempty"`

	// 检查类型
	// Enum: [offline short long conveyance]
	Type string `json:"type,omitempty"`
}

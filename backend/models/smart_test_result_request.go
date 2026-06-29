package models

// swagger:model SmartTestResultRequest
type SmartTestResultRequest struct {

	// 磁盘路径
	DevicePath string `json:"devicePath,omitempty"`

	// 检查类型，写死为selftest,error目前没有使用场景
	// Enum: [error selftest]
	Type string `json:"type,omitempty"`
}

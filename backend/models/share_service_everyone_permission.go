package models

// swagger:model ShareServiceEveryonePermission
type ShareServiceEveryonePermission struct {

	// 只读权限
	ReadOnly bool `json:"readOnly,omitempty"`

	// 读写权限
	Readwrite bool `json:"readwrite,omitempty"`
}

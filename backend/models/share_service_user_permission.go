package models

// swagger:model shareServiceUserPermission
type ShareServiceUserPermission struct {

	// 只读权限
	Ro bool `json:"ro,omitempty"`

	// 读写权限
	Rw bool `json:"rw,omitempty"`

	// 用户名
	UserName string `json:"userName,omitempty"`
}

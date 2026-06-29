package models

// swagger:model shareUserCreateRequest
type ShareUserCreateRequest struct {

	// 密码
	Password string `json:"password,omitempty"`

	// 用户名
	UserName string `json:"userName,omitempty"`
}

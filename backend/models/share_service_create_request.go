package models

// swagger:model shareServiceCreateRequest
type ShareServiceCreateRequest struct {

	// 用户名
	Name string `json:"name,omitempty"`

	// 目录路径
	Path string `json:"path,omitempty"`

	// 支持 samba
	Samba bool `json:"samba,omitempty"`

	// users
	Users []*ShareServiceUserPermission `json:"users"`

	// 支持 webdav
	Webdav bool `json:"webdav,omitempty"`
}

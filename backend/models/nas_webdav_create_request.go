package models

// swagger:model nasWebdavCreateRequest
type NasWebdavCreateRequest struct {

	// 密码
	Password string `json:"password,omitempty"`

	// 服务根目录
	RootPath string `json:"rootPath,omitempty"`

	// 用户名
	Username string `json:"username,omitempty"`
}

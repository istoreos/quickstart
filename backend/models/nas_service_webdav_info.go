package models

// swagger:model nasServiceWebdavInfo
type NasServiceWebdavInfo struct {

	// password
	Password string `json:"password,omitempty"`

	// path
	Path string `json:"path,omitempty"`

	// port
	Port string `json:"port,omitempty"`

	// username
	Username string `json:"username,omitempty"`
}

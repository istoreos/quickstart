package models

// swagger:model nasWebdavStatusResponse
type NasWebdavStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasWebdavStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasWebdavStatusResponseResult
type NasWebdavStatusResponseResult struct {

	// password
	Password string `json:"password,omitempty"`

	// path
	Path string `json:"path,omitempty"`

	// port
	Port string `json:"port,omitempty"`

	// username
	Username string `json:"username,omitempty"`

	// webdav服务地址
	WebdavURL string `json:"webdavUrl,omitempty"`
}

package models

// swagger:model nasWebdavCreateResponse
type NasWebdavCreateResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasWebdavCreateResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasWebdavCreateResponseResult
type NasWebdavCreateResponseResult struct {

	// password
	Password string `json:"password,omitempty"`

	// 用户名
	Username string `json:"username,omitempty"`

	// webdav服务地址
	WebdavURL string `json:"webdavUrl,omitempty"`
}

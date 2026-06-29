package models

// swagger:model guidePppoeStatusResponse
type GuidePppoeStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuidePppoeStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuidePppoeStatusResponseResult
type GuidePppoeStatusResponseResult struct {

	// 拨号账号
	Account string `json:"account,omitempty"`

	// 拨号密码
	Password string `json:"password,omitempty"`
}

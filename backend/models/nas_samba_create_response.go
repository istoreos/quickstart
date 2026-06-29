package models

// swagger:model nasSambaCreateResponse
type NasSambaCreateResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasSambaCreateResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasSambaCreateResponseResult
type NasSambaCreateResponseResult struct {

	// samba服务地址
	SambaURL string `json:"sambaUrl,omitempty"`

	// 用户名
	Username string `json:"username,omitempty"`
}

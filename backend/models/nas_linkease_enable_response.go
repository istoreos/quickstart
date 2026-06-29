package models

// swagger:model nasLinkeaseEnableResponse
type NasLinkeaseEnableResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasLinkeaseEnableResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasLinkeaseEnableResponseResult
type NasLinkeaseEnableResponseResult struct {

	// linkease端口
	Port string `json:"port,omitempty"`
}

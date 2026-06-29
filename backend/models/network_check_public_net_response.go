package models

// swagger:model networkCheckPublicNetResponse
type NetworkCheckPublicNetResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NetworkCheckPublicNetResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NetworkCheckPublicNetResponseResult
type NetworkCheckPublicNetResponseResult struct {

	// 公网地址
	Address string `json:"address,omitempty"`
}

package models

// swagger:model networkHomeBoxEnableResponse
type NetworkHomeBoxEnableResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NetworkHomeBoxEnableResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NetworkHomeBoxEnableResponseResult
type NetworkHomeBoxEnableResponseResult struct {

	// homebox端口
	Port string `json:"port,omitempty"`
}

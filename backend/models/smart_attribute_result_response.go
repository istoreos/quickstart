package models

// swagger:model SmartAttributeResultResponse
type SmartAttributeResultResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SmartAttributeResultResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SmartAttributeResultResponseResult
type SmartAttributeResultResponseResult struct {

	// result
	Result string `json:"result,omitempty"`
}

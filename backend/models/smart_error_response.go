package models

// swagger:model SmartErrorResponse
type SmartErrorResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SmartErrorResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SmartErrorResponseResult
type SmartErrorResponseResult struct {

	// result
	Result string `json:"result,omitempty"`
}

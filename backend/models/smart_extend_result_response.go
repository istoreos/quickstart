package models

// swagger:model SmartExtendResultResponse
type SmartExtendResultResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SmartExtendResultResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SmartExtendResultResponseResult
type SmartExtendResultResponseResult struct {

	// result
	Result string `json:"result,omitempty"`
}

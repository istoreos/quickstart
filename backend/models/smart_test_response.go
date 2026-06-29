package models

// swagger:model SmartTestResponse
type SmartTestResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SmartTestResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SmartTestResponseResult
type SmartTestResponseResult struct {

	// result
	Result string `json:"result,omitempty"`
}

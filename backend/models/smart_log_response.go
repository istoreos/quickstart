package models

// swagger:model SmartLogResponse
type SmartLogResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SmartLogResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SmartLogResponseResult
type SmartLogResponseResult struct {

	// result
	Result string `json:"result,omitempty"`
}

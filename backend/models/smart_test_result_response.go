package models

// swagger:model SmartTestResultResponse
type SmartTestResultResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SmartTestResultResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SmartTestResultResponseResult
type SmartTestResultResponseResult struct {

	// result
	Result string `json:"result,omitempty"`
}

package models

// swagger:model systemCheckUpdateResponse
type SystemCheckUpdateResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemCheckUpdateResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SystemCheckUpdateResponseResult
type SystemCheckUpdateResponseResult struct {

	// msg
	Msg string `json:"msg,omitempty"`

	// need update
	NeedUpdate bool `json:"needUpdate,omitempty"`
}

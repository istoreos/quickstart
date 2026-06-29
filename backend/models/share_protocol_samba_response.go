package models

// swagger:model shareProtocolSambaResponse
type ShareProtocolSambaResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *ShareProtocolSambaConfig `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}

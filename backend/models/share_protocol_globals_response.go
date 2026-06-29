package models

// swagger:model shareProtocolGlobalsResponse
type ShareProtocolGlobalsResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *ShareProtocolGlobalsResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model ShareProtocolGlobalsResponseResult
type ShareProtocolGlobalsResponseResult struct {

	// services
	Services []*ShareProtocolGlobalsConfig `json:"services"`
}

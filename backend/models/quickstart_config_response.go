package models

// swagger:model QuickstartConfigResponse
type QuickstartConfigResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *QuickstartConfigResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model QuickstartConfigResponseResult
type QuickstartConfigResponseResult struct {

	// key
	Key string `json:"key,omitempty"`

	// type
	// Enum: [list option]
	Type string `json:"type,omitempty"`

	// values
	Values []string `json:"values"`
}

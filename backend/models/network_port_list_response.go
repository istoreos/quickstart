package models

// swagger:model networkPortListResponse
type NetworkPortListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NetworkPortListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NetworkPortListResponseResult
type NetworkPortListResponseResult struct {

	// ports
	Ports []*NetworkPortInfo `json:"ports"`
}

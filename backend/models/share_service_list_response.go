package models

// swagger:model shareServiceListResponse
type ShareServiceListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *ShareServiceListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model ShareServiceListResponseResult
type ShareServiceListResponseResult struct {

	// services
	Services []*ShareServiceInfo `json:"services"`
}

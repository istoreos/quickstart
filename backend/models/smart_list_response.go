package models

// swagger:model SmartListResponse
type SmartListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SmartListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SmartListResponseResult
type SmartListResponseResult struct {

	// disks
	Disks []*SmartInfo `json:"disks"`
}

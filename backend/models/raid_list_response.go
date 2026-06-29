package models

// swagger:model raidListResponse
type RaidListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *RaidListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model RaidListResponseResult
type RaidListResponseResult struct {

	// disks
	Disks []*NasDiskInfo `json:"disks"`
}

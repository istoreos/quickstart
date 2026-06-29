package models

// swagger:model nasDiskPartitionMountResponse
type NasDiskPartitionMountResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *PartitionInfo `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}

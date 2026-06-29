package models

// swagger:model nasDiskPartitionFormatResponse
type NasDiskPartitionFormatResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *PartitionInfo `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}

package models

// swagger:model nasDiskStatusResponse
type NasDiskStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasDiskStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasDiskStatusResponseResult
type NasDiskStatusResponseResult struct {

	// disks
	Disks []*NasDiskInfo `json:"disks"`
}

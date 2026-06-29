package models

// swagger:model nasDiskInitDiskResponse
type NasDiskInitDiskResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasDiskInfo `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}

package models

// swagger:model nasSandboxDisksResponse
type NasSandboxDisksResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasSandboxDisksResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasSandboxDisksResponseResult
type NasSandboxDisksResponseResult struct {

	// disks
	Disks []*NasDiskInfo `json:"disks"`
}

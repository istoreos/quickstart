package models

// swagger:model NasDiskMountPointResponse
type NasDiskMountPointResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasDiskMountPointResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasDiskMountPointResponseResult
type NasDiskMountPointResponseResult struct {

	// mountpoint
	Mountpoint string `json:"mountpoint,omitempty"`
}

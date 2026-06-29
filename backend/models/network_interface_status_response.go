package models

// swagger:model networkInterfaceStatusResponse
type NetworkInterfaceStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NetworkInterfaceStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NetworkInterfaceStatusResponseResult
type NetworkInterfaceStatusResponseResult struct {

	// interfaces
	Interfaces []*NetworkInterfaceInfo `json:"interfaces"`
}

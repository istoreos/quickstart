package models

// swagger:model NetworkInterfaceGetConfigResponse
type NetworkInterfaceGetConfigResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NetworkInterfaceGetConfigResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NetworkInterfaceGetConfigResponseResult
type NetworkInterfaceGetConfigResponseResult struct {

	// devices
	Devices []*NetworkPortInfo `json:"devices"`

	// interfaces
	Interfaces []*NetworkInterfaceInfo `json:"interfaces"`
}

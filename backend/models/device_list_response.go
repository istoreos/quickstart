package models

// swagger:model deviceListResponse
type DeviceListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *DeviceListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model DeviceListResponseResult
type DeviceListResponseResult struct {

	// devices
	Devices []*DeviceInfo `json:"devices"`
}

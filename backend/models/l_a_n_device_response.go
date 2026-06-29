package models

// swagger:model lANDeviceResponse
type LANDeviceResponse struct {
	JSONResponse

	// result
	Result *LANDeviceResponseResult `json:"result,omitempty"`
}
// swagger:model LANDeviceResponseResult
type LANDeviceResponseResult struct {

	// devices
	Devices LANDevices `json:"devices,omitempty"`

	// dhcp tags
	DhcpTags LANCtrlDhcpTags `json:"dhcpTags,omitempty"`
}

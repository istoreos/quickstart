package models

// swagger:model wirelessListIfaceResponse
type WirelessListIfaceResponse struct {
	JSONResponse

	// result
	Result *WirelessListIfaceResponseResult `json:"result,omitempty"`
}
// swagger:model WirelessListIfaceResponseResult
type WirelessListIfaceResponseResult struct {

	// ifaces
	Ifaces []*WirelessIfaceInfo `json:"ifaces"`
}

package models

// swagger:model wirelessQuickSetupRequest
type WirelessQuickSetupRequest struct {

	// wifi2g
	Wifi2g *WirelessIfaceInfo `json:"wifi2g,omitempty"`

	// wifi5g
	Wifi5g *WirelessIfaceInfo `json:"wifi5g,omitempty"`
}

package models

// swagger:model wirelessEnableIfaceRequest
type WirelessEnableIfaceRequest struct {

	// enable
	Enable bool `json:"enable,omitempty"`

	// iface name
	IfaceName string `json:"ifaceName,omitempty"`
}

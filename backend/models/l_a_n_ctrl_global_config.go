package models

// swagger:model lANCtrlGlobalConfig
type LANCtrlGlobalConfig struct {

	// dhcp global
	DhcpGlobal *LANDhcpGlobalConfig `json:"dhcpGlobal,omitempty"`

	// dhcp tags
	DhcpTags LANCtrlDhcpTags `json:"dhcpTags,omitempty"`

	// float gateway
	FloatGateway *LANCtrlFloatGatewayModule `json:"floatGateway,omitempty"`

	// speed limit
	SpeedLimit *LANCtrlSpeedLimitModule `json:"speedLimit,omitempty"`
}

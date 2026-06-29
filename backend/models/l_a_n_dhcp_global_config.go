package models

// swagger:model lANDhcpGlobalConfig
type LANDhcpGlobalConfig struct {

	// dhcp enabled
	DhcpEnabled bool `json:"dhcpEnabled,omitempty"`

	// DHCP 网关
	DhcpGateway string `json:"dhcpGateway,omitempty"`

	// gateway sels
	GatewaySels []*LANDhcpGatewaySel `json:"gatewaySels"`
}

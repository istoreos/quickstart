package models

// swagger:model lANCtrlDhcpGatewayConfigRequest
type LANCtrlDhcpGatewayConfigRequest struct {

	// 是否启用 DHCP
	DhcpEnabled bool `json:"dhcpEnabled,omitempty"`

	// DHCP 网关地址
	DhcpGateway string `json:"dhcpGateway,omitempty"`
}

package models

// swagger:model NetworkInterfaceConfig
type NetworkInterfaceConfig struct {

	// devices
	Devices []string `json:"devices"`

	// 防火墙类型
	// Example: lan,wan
	FirewallType string `json:"firewallType,omitempty"`

	// name
	// Example: lan,wan,wan6
	Name string `json:"name,omitempty"`

	// 协议
	// Example: dhcp,static,pppoe
	Proto string `json:"proto,omitempty"`
}

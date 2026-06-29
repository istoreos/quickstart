package models

// swagger:model networkInterfaceInfo
type NetworkInterfaceInfo struct {

	// 接口数组
	DeviceNames []string `json:"deviceNames"`

	// 防火墙区域
	// Example: lan,wan
	FirewallType string `json:"firewallType,omitempty"`

	// ipv4地址
	IPV4Addr string `json:"ipv4Addr,omitempty"`

	// ipv6地址
	IPV6Addr string `json:"ipv6Addr,omitempty"`

	// name
	// Example: lan,wan,wan6
	Name string `json:"name,omitempty"`

	// 使用的接口
	// Example: eth0,eth1
	PortName string `json:"portName,omitempty"`

	// ports
	Ports []*NetworkPortInfo `json:"ports"`

	// 协议
	// Example: dhcp,static
	Proto string `json:"proto,omitempty"`
}

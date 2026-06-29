package models

// swagger:model deviceInfo
type DeviceInfo struct {

	// IPv4 地址
	Ipv4addr string `json:"ipv4addr,omitempty"`

	// IPv6 地址
	Ipv6addr string `json:"ipv6addr,omitempty"`

	// MAC 地址
	Mac string `json:"mac,omitempty"`

	// 设备名称
	Name string `json:"name,omitempty"`

	// 设备类型
	// Enum: [mobile pc other]
	Type string `json:"type,omitempty"`
}

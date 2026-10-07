package models

type DeviceManagementCapability = Capability

// swagger:model deviceManagementCapabilities
type DeviceManagementCapabilities struct {
	// canonical capability keys shared by every product surface
	Items map[string]*Capability `json:"items,omitempty"`

	// independent firewall-backed network access
	InternetAccess *DeviceManagementCapability `json:"internetAccess,omitempty"`

	// per-device and global speed limiting
	SpeedLimit *DeviceManagementCapability `json:"speedLimit"`

	// floating gateway configuration
	FloatGateway *DeviceManagementCapability `json:"floatGateway"`

	// bounded local traffic history
	TrafficInsights *DeviceManagementCapability `json:"trafficInsights,omitempty"`
}

package models

// swagger:model deviceManagementCapability
type DeviceManagementCapability struct {
	// state: available, disabled, not_installed, error
	State string `json:"state"`

	// machine-readable explanation when the capability is unavailable
	Reason string `json:"reason,omitempty"`
}

// swagger:model deviceManagementCapabilities
type DeviceManagementCapabilities struct {
	// per-device and global speed limiting
	SpeedLimit *DeviceManagementCapability `json:"speedLimit"`

	// floating gateway configuration
	FloatGateway *DeviceManagementCapability `json:"floatGateway"`
}

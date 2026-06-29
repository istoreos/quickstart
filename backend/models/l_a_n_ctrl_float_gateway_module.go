package models

// swagger:model lANCtrlFloatGatewayModule
type LANCtrlFloatGatewayModule struct {

	// check IP
	CheckIP string `json:"checkIP,omitempty"`

	// check Url
	CheckURL string `json:"checkUrl,omitempty"`

	// check Url timeout
	CheckURLTimeout int64 `json:"checkUrlTimeout,omitempty"`

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// installed
	Installed bool `json:"installed,omitempty"`

	// role
	Role string `json:"role,omitempty"`

	// set IP
	SetIP string `json:"setIP,omitempty"`
}

package models

// swagger:model devicePolicyCapability
type DevicePolicyCapability struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

// swagger:model deviceStaticPolicy
type DeviceStaticPolicy struct {
	Enabled    bool   `json:"enabled"`
	AssignedIP string `json:"assignedIP,omitempty"`
	BindIP     bool   `json:"bindIP"`
	Hostname   string `json:"hostname,omitempty"`
	TagName    string `json:"tagName,omitempty"`
	TagTitle   string `json:"tagTitle,omitempty"`
}

// swagger:model deviceSpeedPolicy
type DeviceSpeedPolicy struct {
	Enabled       bool  `json:"enabled"`
	UploadSpeed   int64 `json:"uploadSpeed"`
	DownloadSpeed int64 `json:"downloadSpeed"`
}

// swagger:model deviceAccessPolicy
type DeviceAccessPolicy struct {
	NetworkAccess bool `json:"networkAccess"`
}

// swagger:model devicePolicy
type DevicePolicy struct {
	DeviceID     string                             `json:"deviceId"`
	DisplayName  string                             `json:"displayName,omitempty"`
	MAC          string                             `json:"mac,omitempty"`
	CurrentIPv4  string                             `json:"currentIPv4,omitempty"`
	Static       *DeviceStaticPolicy                `json:"static"`
	Speed        *DeviceSpeedPolicy                 `json:"speed"`
	Access       *DeviceAccessPolicy                `json:"access"`
	Capabilities map[string]*DevicePolicyCapability `json:"capabilities"`
}

// swagger:model devicePolicyError
type DevicePolicyError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// swagger:model devicePolicyResult
type DevicePolicyResult struct {
	Policy  *DevicePolicy      `json:"policy,omitempty"`
	Changed bool               `json:"changed,omitempty"`
	Error   *DevicePolicyError `json:"error,omitempty"`
}

// swagger:model devicePolicyResponse
type DevicePolicyResponse struct {
	JSONResponse
	Result *DevicePolicyResult `json:"result,omitempty"`
}

// swagger:model devicePolicyApplyRequest
type DevicePolicyApplyRequest struct {
	DeviceID       string              `json:"deviceId"`
	Kind           string              `json:"kind"`
	IdempotencyKey string              `json:"idempotencyKey,omitempty"`
	Static         *DeviceStaticPolicy `json:"static,omitempty"`
	Speed          *DeviceSpeedPolicy  `json:"speed,omitempty"`
	Access         *DeviceAccessPolicy `json:"access,omitempty"`
}

// swagger:model devicePolicyRulesResult
type DevicePolicyRulesResult struct {
	Static []*LANStaticAssigned     `json:"static"`
	Speed  []*LANCtrlSpeedLimitItem `json:"speed"`
}

// swagger:model devicePolicyRulesResponse
type DevicePolicyRulesResponse struct {
	JSONResponse
	Result *DevicePolicyRulesResult `json:"result,omitempty"`
}

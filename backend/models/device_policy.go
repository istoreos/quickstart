package models

type CapabilityAction struct {
	Kind                 string `json:"kind"`
	Target               string `json:"target,omitempty"`
	RequiresConfirmation bool   `json:"requiresConfirmation,omitempty"`
}

// Capability is the shared truth contract for optional device-management features.
type Capability struct {
	State           string              `json:"state"`
	Reason          string              `json:"reason,omitempty"`
	Actions         []*CapabilityAction `json:"actions,omitempty"`
	DesiredRetained bool                `json:"desiredRetained,omitempty"`
}

type DevicePolicyCapability = Capability

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
	Version      string                             `json:"version"`
}

// swagger:model devicePolicyError
type DevicePolicyError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// swagger:model devicePolicyResult
type DevicePolicyResult struct {
	Policy      *DevicePolicy      `json:"policy,omitempty"`
	Changed     bool               `json:"changed,omitempty"`
	Error       *DevicePolicyError `json:"error,omitempty"`
	Transaction *TaskTransaction   `json:"transaction,omitempty"`
}

// swagger:model devicePolicyResponse
type DevicePolicyResponse struct {
	JSONResponse
	Result *DevicePolicyResult `json:"result,omitempty"`
}

// swagger:model devicePolicyApplyRequest
type DevicePolicyApplyRequest struct {
	DeviceID        string              `json:"deviceId"`
	Kind            string              `json:"kind"`
	IdempotencyKey  string              `json:"idempotencyKey,omitempty"`
	Static          *DeviceStaticPolicy `json:"static,omitempty"`
	Speed           *DeviceSpeedPolicy  `json:"speed,omitempty"`
	Access          *DeviceAccessPolicy `json:"access,omitempty"`
	ExpectedVersion string              `json:"expectedVersion,omitempty"`
}

type DeviceRestrictionPlanResult struct {
	DeviceID       string              `json:"deviceId"`
	Kind           string              `json:"kind"`
	CurrentSpeed   *DeviceSpeedPolicy  `json:"currentSpeed,omitempty"`
	DesiredSpeed   *DeviceSpeedPolicy  `json:"desiredSpeed,omitempty"`
	CurrentAccess  *DeviceAccessPolicy `json:"currentAccess,omitempty"`
	DesiredAccess  *DeviceAccessPolicy `json:"desiredAccess,omitempty"`
	Changes        []*PolicyPlanChange `json:"changes"`
	ReloadServices []string            `json:"reloadServices"`
	RecoveryAction string              `json:"recoveryAction"`
	Version        string              `json:"version"`
	RollbackPoint  string              `json:"rollbackPoint"`
	CanApply       bool                `json:"canApply"`
	Error          *DevicePolicyError  `json:"error,omitempty"`
}

type DeviceRestrictionPlanResponse struct {
	JSONResponse
	Result *DeviceRestrictionPlanResult `json:"result,omitempty"`
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

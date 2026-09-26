package models

type DeviceInternetPath struct {
	TargetID string        `json:"targetId"`
	Effect   *PolicyEffect `json:"effect"`
}

type DesiredNetworkPolicy struct {
	TargetID  string               `json:"targetId"`
	Static    *DeviceAddressPolicy `json:"static"`
	UpdatedAt string               `json:"updatedAt,omitempty"`
}

type AppliedNetworkConfiguration struct {
	TargetID      string `json:"targetId"`
	ConfigVersion string `json:"configVersion"`
	State         string `json:"state"`
	AppliedAt     string `json:"appliedAt,omitempty"`
}

type ObservedNetworkEffect struct {
	State      string `json:"state"`
	Reason     string `json:"reason,omitempty"`
	Source     string `json:"source,omitempty"`
	ObservedAt string `json:"observedAt,omitempty"`
}

type PolicyEffect struct {
	Desired         *DesiredNetworkPolicy        `json:"desired"`
	Applied         *AppliedNetworkConfiguration `json:"applied"`
	Observed        *ObservedNetworkEffect       `json:"observed"`
	NeedsAttention  bool                         `json:"needsAttention"`
	AttentionReason string                       `json:"attentionReason,omitempty"`
}

type DeviceAddressPolicy struct {
	Enabled    bool   `json:"enabled"`
	AssignedIP string `json:"assignedIP,omitempty"`
	BindIP     bool   `json:"bindIP"`
	Hostname   string `json:"hostname,omitempty"`
}

type DeviceNetworkPolicy struct {
	DeviceID string               `json:"deviceId"`
	Static   *DeviceAddressPolicy `json:"static"`
	Path     *DeviceInternetPath  `json:"path"`
	Targets  []*GatewayTarget     `json:"targets"`
	Version  string               `json:"version"`
}

type DeviceNetworkPolicyResult struct {
	Policy      *DeviceNetworkPolicy `json:"policy,omitempty"`
	Changed     bool                 `json:"changed"`
	Error       *DevicePolicyError   `json:"error,omitempty"`
	Transaction *TaskTransaction     `json:"transaction,omitempty"`
}

type DeviceNetworkPolicyResponse struct {
	JSONResponse
	Result *DeviceNetworkPolicyResult `json:"result,omitempty"`
}

type DeviceNetworkPolicyApplyRequest struct {
	DeviceID        string               `json:"deviceId"`
	IdempotencyKey  string               `json:"idempotencyKey,omitempty"`
	Static          *DeviceAddressPolicy `json:"static"`
	TargetID        string               `json:"targetId"`
	ExpectedVersion string               `json:"expectedVersion,omitempty"`
}

type PolicyPlanChange struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type DeviceNetworkPolicyPlanResult struct {
	DeviceID        string                `json:"deviceId"`
	Current         *DesiredNetworkPolicy `json:"current"`
	Desired         *DesiredNetworkPolicy `json:"desired"`
	Changes         []*PolicyPlanChange   `json:"changes"`
	ReloadServices  []string              `json:"reloadServices"`
	RequiresRenewal bool                  `json:"requiresRenewal"`
	RecoveryAction  string                `json:"recoveryAction"`
	Version         string                `json:"version"`
	RollbackPoint   string                `json:"rollbackPoint"`
	CanApply        bool                  `json:"canApply"`
	Error           *DevicePolicyError    `json:"error,omitempty"`
}

type DeviceNetworkPolicyPlanResponse struct {
	JSONResponse
	Result *DeviceNetworkPolicyPlanResult `json:"result,omitempty"`
}

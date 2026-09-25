package models

type GatewayTarget struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Kind           string   `json:"kind"`
	Gateway        string   `json:"gateway,omitempty"`
	DNS            []string `json:"dns,omitempty"`
	Supported      bool     `json:"supported"`
	Reasons        []string `json:"reasons,omitempty"`
	ReferenceCount int64    `json:"referenceCount"`
	DesiredState   string   `json:"desiredState"`
	EffectState    string   `json:"effectState"`
	Version        string   `json:"version"`
}

type GatewayTargetListResult struct {
	Targets []*GatewayTarget `json:"targets"`
	Version string           `json:"version"`
	Health  string           `json:"health"`
}

type GatewayTargetListResponse struct {
	JSONResponse
	Result *GatewayTargetListResult `json:"result,omitempty"`
}

type GatewayReference struct {
	DeviceID string `json:"deviceId,omitempty"`
	Scope    string `json:"scope"`
}

type GatewayReferencesResult struct {
	TargetID   string              `json:"targetId"`
	References []*GatewayReference `json:"references"`
}

type GatewayReferencesResponse struct {
	JSONResponse
	Result *GatewayReferencesResult `json:"result,omitempty"`
}

type GatewayAssignmentRequest struct {
	Action          string `json:"action"`
	DeviceID        string `json:"deviceId,omitempty"`
	TargetID        string `json:"targetId"`
	ExpectedVersion string `json:"expectedVersion,omitempty"`
}

type GatewayPlanChange struct {
	Kind        string `json:"kind"`
	Description string `json:"description"`
}

type GatewayAssignmentPlan struct {
	Action          string               `json:"action"`
	DeviceID        string               `json:"deviceId,omitempty"`
	TargetID        string               `json:"targetId"`
	CurrentTargetID string               `json:"currentTargetId,omitempty"`
	Version         string               `json:"version"`
	RollbackPoint   string               `json:"rollbackPoint"`
	Changes         []*GatewayPlanChange `json:"changes"`
	AffectedDevices []string             `json:"affectedDevices"`
	RequiresRenewal bool                 `json:"requiresRenewal"`
	CanApply        bool                 `json:"canApply"`
	Error           *DevicePolicyError   `json:"error,omitempty"`
}

type GatewayAssignmentPlanResponse struct {
	JSONResponse
	Result *GatewayAssignmentPlan `json:"result,omitempty"`
}

type GatewayAssignmentApplyResult struct {
	Plan        *GatewayAssignmentPlan `json:"plan"`
	Changed     bool                   `json:"changed"`
	EffectState string                 `json:"effectState"`
	Error       *DevicePolicyError     `json:"error,omitempty"`
}

type GatewayAssignmentApplyResponse struct {
	JSONResponse
	Result *GatewayAssignmentApplyResult `json:"result,omitempty"`
}

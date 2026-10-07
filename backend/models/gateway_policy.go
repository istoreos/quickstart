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
	GroupID  string `json:"groupId,omitempty"`
	Scope    string `json:"scope"`
}

type GatewayReferenceSummary struct {
	Devices      int64 `json:"devices"`
	Groups       int64 `json:"groups"`
	GlobalPolicy int64 `json:"globalPolicy"`
	LanDefault   int64 `json:"lanDefault"`
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

// GatewayTargetMutationRequest manages a user-facing Internet Path. DHCP tags
// and options intentionally remain private to the GatewayPolicy module.
type GatewayTargetMutationRequest struct {
	Action              string `json:"action"`
	TargetID            string `json:"targetId,omitempty"`
	Name                string `json:"name,omitempty"`
	Kind                string `json:"kind,omitempty"`
	Gateway             string `json:"gateway,omitempty"`
	ReplacementTargetID string `json:"replacementTargetId,omitempty"`
	ExpectedVersion     string `json:"expectedVersion,omitempty"`
	IdempotencyKey      string `json:"idempotencyKey,omitempty"`
}

type GatewayTargetMutationPlan struct {
	Action              string                   `json:"action"`
	Target              *GatewayTarget           `json:"target,omitempty"`
	ReplacementTargetID string                   `json:"replacementTargetId,omitempty"`
	ReferenceSummary    *GatewayReferenceSummary `json:"referenceSummary"`
	AffectedDevices     []string                 `json:"affectedDevices"`
	Version             string                   `json:"version"`
	RollbackPoint       string                   `json:"rollbackPoint"`
	CanApply            bool                     `json:"canApply"`
	Error               *DevicePolicyError       `json:"error,omitempty"`
}

type GatewayTargetMutationPlanResponse struct {
	JSONResponse
	Result *GatewayTargetMutationPlan `json:"result,omitempty"`
}

type GatewayTargetMutationApplyResult struct {
	Plan        *GatewayTargetMutationPlan `json:"plan"`
	Changed     bool                       `json:"changed"`
	Transaction *TaskTransaction           `json:"transaction,omitempty"`
	Error       *DevicePolicyError         `json:"error,omitempty"`
}

type GatewayTargetMutationApplyResponse struct {
	JSONResponse
	Result *GatewayTargetMutationApplyResult `json:"result,omitempty"`
}

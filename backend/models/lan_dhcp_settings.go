package models

type LanDHCPSettings struct {
	Enabled         bool   `json:"enabled"`
	PoolStart       string `json:"poolStart"`
	PoolEnd         string `json:"poolEnd"`
	LeaseTime       string `json:"leaseTime"`
	DefaultTargetID string `json:"defaultTargetId"`
}

type LanDHCPConflict struct {
	Address string `json:"address"`
	Kind    string `json:"kind"`
}

type LanDHCPSettingsResult struct {
	Settings         *LanDHCPSettings   `json:"settings,omitempty"`
	Version          string             `json:"version"`
	Editable         bool               `json:"editable"`
	ReadOnlyReason   string             `json:"readOnlyReason,omitempty"`
	AffectedDevices  int64              `json:"affectedDevices"`
	RecoveryGuidance string             `json:"recoveryGuidance,omitempty"`
	Conflicts        []*LanDHCPConflict `json:"conflicts"`
	ReloadServices   []string           `json:"reloadServices"`
	CanApply         bool               `json:"canApply"`
	Changed          bool               `json:"changed"`
	Transaction      *TaskTransaction   `json:"transaction,omitempty"`
	Error            *DevicePolicyError `json:"error,omitempty"`
}

type LanDHCPSettingsResponse struct {
	JSONResponse
	Result *LanDHCPSettingsResult `json:"result,omitempty"`
}

type LanDHCPSettingsApplyRequest struct {
	Settings        *LanDHCPSettings `json:"settings"`
	ConfirmDisable  bool             `json:"confirmDisable,omitempty"`
	ExpectedVersion string           `json:"expectedVersion,omitempty"`
	IdempotencyKey  string           `json:"idempotencyKey,omitempty"`
}

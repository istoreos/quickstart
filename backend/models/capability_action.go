package models

type CapabilityActionRequest struct {
	CapabilityKey string `json:"capabilityKey"`
	Action        string `json:"action"`
	DraftToken    string `json:"draftToken,omitempty"`
	Confirm       bool   `json:"confirm,omitempty"`
	ExpectedState string `json:"expectedState,omitempty"`
}

type CapabilityActionPlan struct {
	CapabilityKey        string             `json:"capabilityKey"`
	Action               string             `json:"action"`
	Current              *Capability        `json:"current,omitempty"`
	Target               string             `json:"target,omitempty"`
	DraftToken           string             `json:"draftToken,omitempty"`
	RequiredFreeBytes    int64              `json:"requiredFreeBytes,omitempty"`
	AvailableFreeBytes   int64              `json:"availableFreeBytes,omitempty"`
	RequiresConfirmation bool               `json:"requiresConfirmation"`
	CanApply             bool               `json:"canApply"`
	Error                *DevicePolicyError `json:"error,omitempty"`
}

type CapabilityActionResult struct {
	Plan      *CapabilityActionPlan `json:"plan,omitempty"`
	Changed   bool                  `json:"changed"`
	Cancelled bool                  `json:"cancelled,omitempty"`
	Current   *Capability           `json:"current,omitempty"`
	Error     *DevicePolicyError    `json:"error,omitempty"`
}

type CapabilityActionResponse struct {
	JSONResponse
	Result *CapabilityActionResult `json:"result,omitempty"`
}

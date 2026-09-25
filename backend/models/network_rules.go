package models

type NetworkRule struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	DeviceID    string `json:"deviceId,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
	MAC         string `json:"mac,omitempty"`
	IP          string `json:"ip,omitempty"`
	Status      string `json:"status"`
	Source      string `json:"source"`
	Summary     string `json:"summary"`
	Orphaned    bool   `json:"orphaned"`
	TargetID    string `json:"targetId,omitempty"`
}

type NetworkRulesResult struct {
	Rules   []*NetworkRule     `json:"rules"`
	Version string             `json:"version"`
	Error   *DevicePolicyError `json:"error,omitempty"`
}

type NetworkRulesResponse struct {
	JSONResponse
	Result *NetworkRulesResult `json:"result,omitempty"`
}

type NetworkRulesBulkRequest struct {
	Action          string   `json:"action"`
	RuleIDs         []string `json:"ruleIds"`
	ExpectedVersion string   `json:"expectedVersion,omitempty"`
}

type NetworkRulesBulkPlan struct {
	CanApply      bool               `json:"canApply"`
	Action        string             `json:"action"`
	RuleIDs       []string           `json:"ruleIds"`
	AffectedCount int                `json:"affectedCount"`
	Version       string             `json:"version"`
	RollbackScope []string           `json:"rollbackScope"`
	Error         *DevicePolicyError `json:"error,omitempty"`
}

type NetworkRulesBulkResult struct {
	Plan    *NetworkRulesBulkPlan `json:"plan,omitempty"`
	Changed bool                  `json:"changed"`
	Error   *DevicePolicyError    `json:"error,omitempty"`
}

type NetworkRulesBulkResponse struct {
	JSONResponse
	Result *NetworkRulesBulkResult `json:"result,omitempty"`
}

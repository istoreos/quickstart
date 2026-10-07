package models

type AdvancedCapability struct {
	State  CapabilityState `json:"state"`
	Reason string          `json:"reason,omitempty"`
	Detail string          `json:"detail,omitempty"`
}

type NetworkAuditEvent struct {
	ID       string `json:"id"`
	At       string `json:"at"`
	DeviceID string `json:"deviceId,omitempty"`
	Resource string `json:"resource"`
	Action   string `json:"action"`
	State    string `json:"state"`
	Reason   string `json:"reason,omitempty"`
}

type AdvancedNetworkResult struct {
	Capabilities map[string]*AdvancedCapability `json:"capabilities"`
	Events       []*NetworkAuditEvent           `json:"events"`
	EventLimit   int                            `json:"eventLimit"`
	Webhook      *WebhookConfig                 `json:"webhook"`
	Error        *DevicePolicyError             `json:"error,omitempty"`
}

type AdvancedNetworkResponse struct {
	JSONResponse
	Result *AdvancedNetworkResult `json:"result,omitempty"`
}

type ManagementProbeRequest struct {
	DeviceID string `json:"deviceId"`
	Address  string `json:"address"`
	Scheme   string `json:"scheme"`
	Port     int    `json:"port"`
}

type ManagementProbeResult struct {
	Reachable  bool               `json:"reachable"`
	StatusCode int                `json:"statusCode,omitempty"`
	URL        string             `json:"url,omitempty"`
	DurationMS int64              `json:"durationMs"`
	Error      *DevicePolicyError `json:"error,omitempty"`
}

type ManagementProbeResponse struct {
	JSONResponse
	Result *ManagementProbeResult `json:"result,omitempty"`
}

type WebhookConfig struct {
	Enabled bool     `json:"enabled"`
	URL     string   `json:"url,omitempty"`
	Events  []string `json:"events"`
}

type PolicyBundle struct {
	SchemaVersion  int                      `json:"schemaVersion"`
	Scope          string                   `json:"scope"`
	ExportedAt     string                   `json:"exportedAt"`
	GlobalPolicy   *GroupPolicy             `json:"globalPolicy,omitempty"`
	Groups         []*DeviceGroup           `json:"groups"`
	DevicePolicies map[string]*GroupPolicy  `json:"devicePolicies"`
	Quotas         map[string]*TrafficQuota `json:"quotas"`
}

type PolicyBundleResponse struct {
	JSONResponse
	Result *PolicyBundleResult `json:"result,omitempty"`
}

type PolicyBundleResult struct {
	Bundle  *PolicyBundle      `json:"bundle,omitempty"`
	Plan    *PolicyImportPlan  `json:"plan,omitempty"`
	Changed bool               `json:"changed,omitempty"`
	Error   *DevicePolicyError `json:"error,omitempty"`
}

type PolicyImportRequest struct {
	Bundle               *PolicyBundle `json:"bundle"`
	ExpectedGroupVersion string        `json:"expectedGroupVersion,omitempty"`
	PlanChecksum         string        `json:"planChecksum,omitempty"`
	Confirmed            bool          `json:"confirmed,omitempty"`
}

type PolicyImportPlan struct {
	CanApply          bool     `json:"canApply"`
	GroupVersion      string   `json:"groupVersion"`
	GroupCount        int      `json:"groupCount"`
	DevicePolicyCount int      `json:"devicePolicyCount"`
	QuotaCount        int      `json:"quotaCount"`
	Warnings          []string `json:"warnings"`
	Checksum          string   `json:"checksum"`
}

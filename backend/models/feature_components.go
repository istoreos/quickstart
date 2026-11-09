package models

// FeatureComponent describes a fixed product dependency, not the currently
// selected business provider. No arbitrary package/command can be supplied.
type FeatureComponent struct {
	NetworkID     string      `json:"networkId,omitempty"`
	Installed     *bool       `json:"installed,omitempty"`
	ID            string      `json:"id"`
	CapabilityKey string      `json:"capabilityKey"`
	Package       string      `json:"package"`
	Provider      string      `json:"provider,omitempty"`
	Selected      bool        `json:"selected"`
	Status        *Capability `json:"status"`
	Operation     string      `json:"operation,omitempty"`
}

type FeatureComponentsResult struct {
	SchemaVersion   int                 `json:"schemaVersion"`
	CurrentProvider string              `json:"currentProvider"`
	Components      []*FeatureComponent `json:"components"`
}

type FeatureComponentsResponse struct {
	JSONResponse
	Result *FeatureComponentsResult `json:"result"`
}

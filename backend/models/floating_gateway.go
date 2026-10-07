package models

type FloatingGatewayConfig struct {
	Enabled          bool     `json:"enabled"`
	Role             string   `json:"role,omitempty"`
	VirtualIP        string   `json:"virtualIP,omitempty"`
	PeerIPs          []string `json:"peerIPs"`
	HealthURL        string   `json:"healthURL,omitempty"`
	HealthTimeoutSec int64    `json:"healthTimeoutSec,omitempty"`
}

type FloatingGatewayStatus struct {
	Capability     string `json:"capability"`
	State          string `json:"state"`
	Holder         string `json:"holder"`
	ServiceRunning bool   `json:"serviceRunning"`
	PeerState      string `json:"peerState"`
	ExternalState  string `json:"externalState"`
	Reason         string `json:"reason,omitempty"`
}

type FloatingGatewayPlan struct {
	CanApply        bool                `json:"canApply"`
	Changed         bool                `json:"changed"`
	Changes         []string            `json:"changes"`
	Warnings        []string            `json:"warnings"`
	AffectedDevices []*GatewayReference `json:"affectedDevices"`
	Version         string              `json:"version"`
	RequiresDrill   bool                `json:"requiresDrill"`
	Error           *DevicePolicyError  `json:"error,omitempty"`
}

type FloatingGatewayResult struct {
	Config  *FloatingGatewayConfig `json:"config,omitempty"`
	Status  *FloatingGatewayStatus `json:"status,omitempty"`
	Plan    *FloatingGatewayPlan   `json:"plan,omitempty"`
	Changed bool                   `json:"changed"`
	Error   *DevicePolicyError     `json:"error,omitempty"`
}

type FloatingGatewayResponse struct {
	JSONResponse
	Result *FloatingGatewayResult `json:"result,omitempty"`
}

type FloatingGatewayApplyRequest struct {
	Config          *FloatingGatewayConfig `json:"config"`
	ExpectedVersion string                 `json:"expectedVersion,omitempty"`
}

type FloatingGatewayDrillPlan struct {
	CanRun         bool               `json:"canRun"`
	Steps          []string           `json:"steps"`
	StopConditions []string           `json:"stopConditions"`
	Recovery       []string           `json:"recovery"`
	Error          *DevicePolicyError `json:"error,omitempty"`
}

type FloatingGatewayDrillResponse struct {
	JSONResponse
	Result *FloatingGatewayDrillPlan `json:"result,omitempty"`
}

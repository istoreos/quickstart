package models

// swagger:model deviceInventoryAddress
type DeviceInventoryAddress struct {
	Family  int64  `json:"family"`
	Address string `json:"address"`
	Primary bool   `json:"primary,omitempty"`
}

// swagger:model deviceInventoryAddresses
type DeviceInventoryAddresses struct {
	Current    []*DeviceInventoryAddress `json:"current"`
	Historical []*DeviceInventoryAddress `json:"historical"`
}

// swagger:model deviceInventoryConnection
type DeviceInventoryConnection struct {
	Kind string `json:"kind"`
}

// swagger:model deviceInventoryIdentity
type DeviceInventoryIdentity struct {
	Kind  string `json:"kind"`
	Scope string `json:"scope"`
}

// swagger:model deviceClassification
type DeviceClassification struct {
	Brand        string `json:"brand,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Category     string `json:"category"`
	Source       string `json:"source"`
	Confidence   string `json:"confidence"`
}

// swagger:model deviceInventoryHealth
type DeviceInventoryHealth struct {
	State   string   `json:"state"`
	Reasons []string `json:"reasons,omitempty"`
}

// swagger:model deviceInventoryItem
type DeviceInventoryItem struct {
	DeviceID       string                     `json:"deviceId"`
	DisplayName    string                     `json:"displayName,omitempty"`
	Hostname       string                     `json:"hostname,omitempty"`
	Online         bool                       `json:"online"`
	PresenceState  string                     `json:"presenceState"`
	LastSeenAt     string                     `json:"lastSeenAt"`
	Mac            string                     `json:"mac"`
	Vendor         string                     `json:"vendor,omitempty"`
	Classification *DeviceClassification      `json:"classification"`
	Icon           *DeviceIconVisual          `json:"icon"`
	Identity       *DeviceInventoryIdentity   `json:"identity"`
	Addresses      *DeviceInventoryAddresses  `json:"addresses"`
	Connection     *DeviceInventoryConnection `json:"connection"`
}

type DeviceInventoryAddRequest struct {
	MAC   string `json:"mac"`
	Alias string `json:"alias,omitempty"`
}

// swagger:model deviceInventoryResult
type DeviceInventoryResult struct {
	Devices []*DeviceInventoryItem `json:"devices"`
	Health  *DeviceInventoryHealth `json:"health"`
}

// swagger:model deviceInventoryResponse
type DeviceInventoryResponse struct {
	JSONResponse
	Result *DeviceInventoryResult `json:"result,omitempty"`
}

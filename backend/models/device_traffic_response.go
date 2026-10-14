package models

// swagger:model deviceTrafficItem
type DeviceTrafficItem struct {
	DeviceID        string `json:"deviceId"`
	UploadSpeed     int64  `json:"uploadSpeed"`
	DownloadSpeed   int64  `json:"downloadSpeed"`
	UploadBytes     int64  `json:"uploadBytes"`
	DownloadBytes   int64  `json:"downloadBytes"`
	ConnectionCount int64  `json:"connectionCount"`
	State           string `json:"state"`
	SampledAt       string `json:"sampledAt,omitempty"`
}

// swagger:model deviceTrafficResult
type DeviceTrafficResult struct {
	Items  []*DeviceTrafficItem   `json:"items"`
	Health *DeviceInventoryHealth `json:"health"`
}

// swagger:model deviceTrafficResponse
type DeviceTrafficResponse struct {
	JSONResponse
	Result *DeviceTrafficResult `json:"result,omitempty"`
}

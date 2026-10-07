package models

type DeviceClassificationOverride struct {
	Active   bool   `json:"active"`
	Category string `json:"category,omitempty"`
	Scope    string `json:"scope"`
}

type DeviceClassificationResult struct {
	DeviceID       string                        `json:"deviceId"`
	Classification *DeviceClassification         `json:"classification,omitempty"`
	Override       *DeviceClassificationOverride `json:"override,omitempty"`
	Changed        bool                          `json:"changed"`
	Error          *DevicePolicyError            `json:"error,omitempty"`
}

type DeviceClassificationResponse struct {
	JSONResponse
	Result *DeviceClassificationResult `json:"result,omitempty"`
}

type DeviceClassificationApplyRequest struct {
	DeviceID string `json:"deviceId"`
	Action   string `json:"action"`
	Category string `json:"category,omitempty"`
}

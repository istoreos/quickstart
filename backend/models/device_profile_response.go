package models

type DeviceProfile struct {
	DeviceID         string                `json:"deviceId"`
	Alias            string                `json:"alias,omitempty"`
	OriginalHostname string                `json:"originalHostname,omitempty"`
	Scope            string                `json:"scope"`
	Classification   *DeviceClassification `json:"classification"`
	ManualBrand      string                `json:"manualBrand,omitempty"`
	ManualCategory   string                `json:"manualCategory,omitempty"`
	Icon             *DeviceIconVisual     `json:"icon"`
}

type DeviceIconVisual struct {
	Mode          string `json:"mode"`
	PreferenceKey string `json:"preferenceKey,omitempty"`
	ResolvedKey   string `json:"resolvedKey"`
	AssetKey      string `json:"assetKey"`
	Label         string `json:"label"`
	BrandLabel    string `json:"brandLabel,omitempty"`
}

type DeviceProfileResult struct {
	Profile     *DeviceProfile     `json:"profile,omitempty"`
	Changed     bool               `json:"changed"`
	Error       *DevicePolicyError `json:"error,omitempty"`
	Transaction *TaskTransaction   `json:"transaction,omitempty"`
}

type DeviceProfileResponse struct {
	JSONResponse
	Result *DeviceProfileResult `json:"result,omitempty"`
}

type DeviceProfileApplyRequest struct {
	DeviceID       string  `json:"deviceId"`
	IdempotencyKey string  `json:"idempotencyKey,omitempty"`
	Action         string  `json:"action"`
	Alias          *string `json:"alias,omitempty"`
	Brand          *string `json:"brand,omitempty"`
	Category       *string `json:"category,omitempty"`
	IconMode       *string `json:"iconMode,omitempty"`
	IconKey        *string `json:"iconKey,omitempty"`
}

package models

// swagger:model guideNeedSetupInfo
type GuideNeedSetupInfo struct {

	// Need setup or not
	Need bool `json:"need,omitempty"`

	// Have wifi setup dialog
	Wifi bool `json:"wifi,omitempty"`
}

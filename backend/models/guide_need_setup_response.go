package models

// swagger:model guideNeedSetupResponse
type GuideNeedSetupResponse struct {
	JSONResponse

	// result
	Result *GuideNeedSetupInfo `json:"result,omitempty"`
}

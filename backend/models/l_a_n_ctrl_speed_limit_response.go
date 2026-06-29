package models

// swagger:model lANCtrlSpeedLimitResponse
type LANCtrlSpeedLimitResponse struct {
	JSONResponse

	// result
	Result []*LANCtrlSpeedLimitItem `json:"result"`
}

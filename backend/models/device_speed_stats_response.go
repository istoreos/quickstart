package models

// swagger:model deviceSpeedStatsResponse
type DeviceSpeedStatsResponse struct {
	JSONResponse

	// result
	Result []*DeviceSpeedStat `json:"result"`
}

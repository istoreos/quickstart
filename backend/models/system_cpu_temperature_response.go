package models

// swagger:model systemCpuTemperatureResponse
type SystemCPUTemperatureResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemCPUTemperatureResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SystemCPUTemperatureResponseResult
type SystemCPUTemperatureResponseResult struct {

	// 温度
	Temperature int64 `json:"temperature,omitempty"`
}

package models

// swagger:model systemCpuStatusResponse
type SystemCPUStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemCPUStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SystemCPUStatusResponseResult
type SystemCPUStatusResponseResult struct {

	// 使用率
	Usage int64 `json:"usage,omitempty"`
}

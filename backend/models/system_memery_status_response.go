package models

// swagger:model systemMemeryStatusResponse
type SystemMemeryStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemMemeryStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SystemMemeryStatusResponseResult
type SystemMemeryStatusResponseResult struct {

	// available
	Available string `json:"available,omitempty"`

	// 可用百分比
	// Example: 57
	AvailablePercentage int64 `json:"availablePercentage,omitempty"`

	// total
	Total string `json:"total,omitempty"`
}

package models

// swagger:model appCheckResponse
type AppCheckResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *AppCheckResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model AppCheckResponseResult
type AppCheckResponseResult struct {

	// 插件名称
	Name string `json:"name,omitempty"`

	// 插件状态
	// Enum: [installed running stopped uninstalled not found]
	Status string `json:"status,omitempty"`
}

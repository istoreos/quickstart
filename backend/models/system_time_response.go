package models

// swagger:model systemTimeResponse
type SystemTimeResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemTimeResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SystemTimeResponseResult
type SystemTimeResponseResult struct {

	// 本地时间
	Localtime string `json:"localtime,omitempty"`

	// 运行时间
	Uptime int64 `json:"uptime,omitempty"`

	// 运行时间
	UptimeHuman string `json:"uptimeHuman,omitempty"`
}

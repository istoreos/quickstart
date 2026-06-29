package models

// swagger:model SystemStatusResponse
type SystemStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SystemStatusResponseResult
type SystemStatusResponseResult struct {

	// 温度
	CPUTemperature int64 `json:"cpuTemperature,omitempty"`

	// cpu使用率
	CPUUsage int64 `json:"cpuUsage,omitempty"`

	// localtime
	Localtime string `json:"localtime,omitempty"`

	// mem available
	MemAvailable string `json:"memAvailable,omitempty"`

	// 可用百分比
	// Example: 57
	MemAvailablePercentage int64 `json:"memAvailablePercentage,omitempty"`

	// mem total
	MemTotal string `json:"memTotal,omitempty"`

	// 运行时间
	Uptime int64 `json:"uptime,omitempty"`

	// uptime human
	UptimeHuman string `json:"uptimeHuman,omitempty"`
}

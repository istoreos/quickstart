package models

// swagger:model systemVersionResponse
type SystemVersionResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemVersionResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SystemVersionResponseResult
type SystemVersionResponseResult struct {

	// 固件版本
	FirmwareVersion string `json:"firmwareVersion,omitempty"`

	// 内核版本
	KernelVersion string `json:"kernelVersion,omitempty"`

	// 设备型号
	Model string `json:"model,omitempty"`

	// quickstart这个程序的版本号
	Quickstart string `json:"quickstart,omitempty"`
}

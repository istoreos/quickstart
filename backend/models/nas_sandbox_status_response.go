package models

// swagger:model nasSandboxStatusResponse
type NasSandboxStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasSandboxStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasSandboxStatusResponseResult
type NasSandboxStatusResponseResult struct {

	// 沙盒模式状态
	// Enum: [unsupport running stopped]
	Status string `json:"status,omitempty"`
}

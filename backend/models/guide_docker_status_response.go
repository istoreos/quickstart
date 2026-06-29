package models

// swagger:model guideDockerStatusResponse
type GuideDockerStatusResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideDockerStatusResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideDockerStatusResponseResult
type GuideDockerStatusResponseResult struct {

	// docker异常信息
	ErrorInfo string `json:"errorInfo,omitempty"`

	// docker根目录
	Path string `json:"path,omitempty"`

	// docker运行状态
	// Enum: [running stopped not installed]
	Status string `json:"status,omitempty"`
}

package models

// swagger:model guideDockerTransferResponse
type GuideDockerTransferResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideDockerTransferResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideDockerTransferResponseResult
type GuideDockerTransferResponseResult struct {

	// 目标路径不为空，无法直接覆盖
	EmptyPathWarning bool `json:"emptyPathWarning,omitempty"`

	// docker迁移路径
	Path string `json:"path,omitempty"`
}

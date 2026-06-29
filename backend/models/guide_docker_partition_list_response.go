package models

// swagger:model guideDockerPartitionListResponse
type GuideDockerPartitionListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideDockerPartitionListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideDockerPartitionListResponseResult
type GuideDockerPartitionListResponseResult struct {

	// docker推荐安装路径
	PartitionList []string `json:"partitionList"`
}

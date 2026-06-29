package models

// swagger:model guideDownloadPartitionListResponse
type GuideDownloadPartitionListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideDownloadPartitionListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideDownloadPartitionListResponseResult
type GuideDownloadPartitionListResponseResult struct {

	// 下载目录推荐安装路径
	PartitionList []string `json:"partitionList"`
}

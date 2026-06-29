package models

// swagger:model guideSoftSourceListResponse
type GuideSoftSourceListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideSoftSourceListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideSoftSourceListResponseResult
type GuideSoftSourceListResponseResult struct {

	// 软件源列表
	SoftSourceList []*GuideSoftSourceInfo `json:"softSourceList"`
}

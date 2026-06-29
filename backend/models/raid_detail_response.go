package models

// swagger:model raidDetailResponse
type RaidDetailResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *RaidDetailResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model RaidDetailResponseResult
type RaidDetailResponseResult struct {

	// detail
	Detail string `json:"detail,omitempty"`
}

package models

// swagger:model raidCreateListResponse
type RaidCreateListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *RaidCreateListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model RaidCreateListResponseResult
type RaidCreateListResponseResult struct {

	// members
	Members []*RaidMemberInfo `json:"members"`
}

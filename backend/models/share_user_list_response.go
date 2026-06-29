package models

// swagger:model shareUserListResponse
type ShareUserListResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *ShareUserListResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model ShareUserListResponseResult
type ShareUserListResponseResult struct {

	// users
	Users []*ShareUserInfo `json:"users"`
}

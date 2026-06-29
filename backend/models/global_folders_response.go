package models

// swagger:model globalFoldersResponse
type GlobalFoldersResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GlobalFolders `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}

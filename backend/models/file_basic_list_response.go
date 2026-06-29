package models

// swagger:model fileBasicListResponse
type FileBasicListResponse struct {
	JSONResponse

	// result
	Result *FileBasicListResponseResult `json:"result,omitempty"`
}
// swagger:model FileBasicListResponseResult
type FileBasicListResponseResult struct {

	// services
	Services []*FileBasicEntry `json:"services"`
}

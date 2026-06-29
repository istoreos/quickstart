package models

// swagger:model fileBasicListRequest
type FileBasicListRequest struct {

	// 只显示文件夹
	OnlyDir bool `json:"onlyDir,omitempty"`

	// path
	Path FilePath `json:"path,omitempty"`

	// show the hidden files
	ShowHidden bool `json:"showHidden,omitempty"`
}

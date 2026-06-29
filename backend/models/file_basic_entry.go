package models

// swagger:model fileBasicEntry
type FileBasicEntry struct {

	// file type
	FileType FileType `json:"fileType,omitempty"`

	// name
	Name string `json:"name,omitempty"`
}

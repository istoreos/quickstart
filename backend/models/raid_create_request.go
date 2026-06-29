package models

// swagger:model raidCreateRequest
type RaidCreateRequest struct {

	// raid磁盘路径数组
	DevicePaths []string `json:"devicePaths"`

	// raid级别
	// Enum: [linear raid0 raid1 raid5 raid6 raid10]
	Level string `json:"level,omitempty"`
}

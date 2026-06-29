package models

// swagger:model raidDeleteRequest
type RaidDeleteRequest struct {

	// raid成员
	Members []string `json:"members"`

	// raid挂载路径
	MountPath string `json:"mountPath,omitempty"`

	// raid磁盘路径
	Path string `json:"path,omitempty"`
}

package models

// swagger:model raidAddRequest
type RaidAddRequest struct {

	// 成员路径
	MemberPath string `json:"memberPath,omitempty"`

	// raid磁盘路径
	Path string `json:"path,omitempty"`
}

package models

// swagger:model raidRecoverRequest
type RaidRecoverRequest struct {

	// 检查磁盘是否存在raid分区
	CheckRaidPartition bool `json:"checkRaidPartition,omitempty"`

	// 成员路径
	MemberPath string `json:"memberPath,omitempty"`

	// raid磁盘路径
	Path string `json:"path,omitempty"`
}

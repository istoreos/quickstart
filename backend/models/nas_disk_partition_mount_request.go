package models

// swagger:model nasDiskPartitionMountRequest
type NasDiskPartitionMountRequest struct {

	// 目标挂载点
	MountPoint string `json:"mountPoint,omitempty"`

	// 硬盘分区路径
	Path string `json:"path,omitempty"`

	// 分区ID
	UUID string `json:"uuid,omitempty"`
}

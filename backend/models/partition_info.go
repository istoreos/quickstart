package models

// swagger:model partitionInfo
type PartitionInfo struct {

	// 文件系统类型
	Filesystem string `json:"filesystem,omitempty"`

	// 是否为docker所在分区
	IsDockerRoot bool `json:"isDockerRoot,omitempty"`

	// 是否设置了raid Flag
	IsRaidOn bool `json:"isRaidOn,omitempty"`

	// 是否只读
	IsReadOnly bool `json:"isReadOnly,omitempty"`

	// 是否为系统所在分区
	IsSystemRoot bool `json:"isSystemRoot,omitempty"`

	// 挂载点
	// Example: /mnt/sda1
	MountPoint string `json:"mountPoint,omitempty"`

	// 分区名称
	// Example: sda1
	Name string `json:"name,omitempty"`

	// 分区号，从1开始，内部使用
	Number uint64 `json:"number,omitempty"`

	// 设备路径
	// Example: /dev/sda1
	Path string `json:"path,omitempty"`

	// 结束扇区，内部使用
	SecEnd uint64 `json:"secEnd,omitempty"`

	// 起始扇区，内部使用
	SecStart uint64 `json:"secStart,omitempty"`

	// 设备容量bytes
	SizeInt string `json:"sizeInt,omitempty"`

	// 设备容量，因为 uint64 结构 json 表达不好，转换成 string
	Total string `json:"total,omitempty"`

	// 使用百分比
	// Example: 57
	Usage uint32 `json:"usage,omitempty"`

	// 设备可用容量，因为 uint64 结构 json 表达不好，转换成 string
	Used string `json:"used,omitempty"`

	// uuid
	UUID string `json:"uuid,omitempty"`
}

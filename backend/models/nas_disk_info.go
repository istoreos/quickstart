package models

// swagger:model nasDiskInfo
type NasDiskInfo struct {

	// active
	Active string `json:"active,omitempty"`

	// 分区列表
	Childrens []*PartitionInfo `json:"childrens"`

	// docker所在分区是否在该磁盘
	IsDockerRoot bool `json:"isDockerRoot,omitempty"`

	// 是否为外挂磁盘,包括sata和usb
	IsExternalDisk bool `json:"isExternalDisk,omitempty"`

	// 系统所在分区是否在该磁盘
	IsSystemRoot bool `json:"isSystemRoot,omitempty"`

	// 级别
	Level string `json:"level,omitempty"`

	// members
	Members []string `json:"members"`

	// name
	// Example: sda
	Name string `json:"name,omitempty"`

	// 设备分区表类型
	PartLabelType string `json:"partLabelType,omitempty"`

	// 路径
	// Example: /dev/sda
	Path string `json:"path,omitempty"`

	// 状态
	RebuildStatus string `json:"rebuildStatus,omitempty"`

	// 设备容量
	Size string `json:"size,omitempty"`

	// 设备容量bytes
	SizeInt string `json:"sizeInt,omitempty"`

	// smart错误
	SmartWarning bool `json:"smartWarning,omitempty"`

	// 状态
	Status string `json:"status,omitempty"`

	// 设备已挂载分区容量，因为 uint64 结构 json 表达不好，转换成 string
	Total string `json:"total,omitempty"`

	// tran name
	TranName string `json:"tranName,omitempty"`

	// 使用百分比
	// Example: 57
	Usage uint32 `json:"usage,omitempty"`

	// 设备可用容量，因为 uint64 结构 json 表达不好，转换成 string
	Used string `json:"used,omitempty"`

	// used int
	UsedInt string `json:"usedInt,omitempty"`

	// 设备型号
	// Example: WDC WD40EJRX-89T1XY0
	VenderModel string `json:"venderModel,omitempty"`
}

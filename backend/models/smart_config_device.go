package models

// swagger:model SmartConfigDevice
type SmartConfigDevice struct {

	// 磁盘路径
	DevicePath string `json:"devicePath,omitempty"`

	// 温度差异，如果为0则表示使用全局配置
	TmpDiff int64 `json:"tmpDiff,omitempty"`

	// 最大温度，如果为0则表示全局配置
	TmpMax int64 `json:"tmpMax,omitempty"`
}

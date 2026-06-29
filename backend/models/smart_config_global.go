package models

// swagger:model SmartConfigGlobal
type SmartConfigGlobal struct {

	// 是否启用
	Enable bool `json:"enable,omitempty"`

	// 电源模式类型
	// Enum: [never sleep standby idle]
	Powermode string `json:"powermode,omitempty"`

	// 温度差异，如果为0则表示禁用
	TmpDiff int64 `json:"tmpDiff,omitempty"`

	// 最大温度，如果为0则表示禁用
	TmpMax int64 `json:"tmpMax,omitempty"`
}

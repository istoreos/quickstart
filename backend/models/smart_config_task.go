package models

// swagger:model SmartConfigTask
type SmartConfigTask struct {

	// 每月的第几天,01 to 31
	DayPerMonth string `json:"dayPerMonth,omitempty"`

	// 每个星期几,1 (Monday) to 7 (Sunday)
	DayPerWeek string `json:"dayPerWeek,omitempty"`

	// 磁盘路径
	DevicePath string `json:"devicePath,omitempty"`

	// 每天的第几个小时,00 (midnight to just before 1 am) to 23 (11pm to just before midnight)
	Hour string `json:"hour,omitempty"`

	// 月份,01 (January) to 12 (December)
	Month string `json:"month,omitempty"`

	// 检查类型
	// Enum: [offline short long conveyance]
	Type string `json:"type,omitempty"`
}

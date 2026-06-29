package models

// swagger:model lANCtrlSpeedLimitItem
type LANCtrlSpeedLimitItem struct {

	// modify|delete|add
	Action string `json:"action,omitempty"`

	// 注解
	Comment string `json:"comment,omitempty"`

	// download speed
	DownloadSpeed int64 `json:"downloadSpeed,omitempty"`

	// 是否启用
	Enabled bool `json:"enabled,omitempty"`

	// 基于IP进行限速
	IP string `json:"ip,omitempty"`

	// 基于MAC进行限速
	Mac string `json:"mac,omitempty"`

	// 是否能访问网络
	NetworkAccess bool `json:"networkAccess,omitempty"`

	// upload speed
	UploadSpeed int64 `json:"uploadSpeed,omitempty"`

	// 设备名称
	Hostname string `json:"hostname,omitempty"`
}

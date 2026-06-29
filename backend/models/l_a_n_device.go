package models

// swagger:model lANDevice
type LANDevice struct {

	// download speed
	DownloadSpeed int64 `json:"downloadSpeed,omitempty"`

	// 下载速度的人类可读格式
	DownloadSpeedStr string `json:"downloadSpeedStr,omitempty"`

	// 设备对应的图片
	HostImg string `json:"hostImg,omitempty"`

	// 设备名称
	Hostname string `json:"hostname,omitempty"`

	// intr
	Intr string `json:"intr,omitempty"`

	// 设备IP地址
	IP string `json:"ip,omitempty"`

	// 设备MAC地址
	Mac string `json:"mac,omitempty"`

	// 设备品牌
	Vendor string `json:"vendor,omitempty"`

	// speed limit
	SpeedLimit *LANCtrlSpeedLimitItem `json:"speedLimit,omitempty"`

	// static assigned
	StaticAssigned *LANStaticAssigned `json:"staticAssigned,omitempty"`

	// upload speed
	UploadSpeed int64 `json:"uploadSpeed,omitempty"`

	// 上传速度的人类可读格式
	UploadSpeedStr string `json:"uploadSpeedStr,omitempty"`
}

package models

// swagger:model deviceSpeedStat
type DeviceSpeedStat struct {

	// download speed
	DownloadSpeed int64 `json:"downloadSpeed,omitempty"`

	// 下载速度的人类可读格式
	DownloadSpeedStr string `json:"downloadSpeedStr,omitempty"`

	// 设备的IP地址
	IP string `json:"ip,omitempty"`

	// upload speed
	UploadSpeed int64 `json:"uploadSpeed,omitempty"`

	// 上传速度的人类可读格式
	UploadSpeedStr string `json:"uploadSpeedStr,omitempty"`
}

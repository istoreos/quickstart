package models

// swagger:model guideTransmissionInitRequest
type GuideTransmissionInitRequest struct {

	// 配置目录
	ConfigPath string `json:"configPath,omitempty"`

	// 下载路径
	DownloadPath string `json:"downloadPath,omitempty"`
}

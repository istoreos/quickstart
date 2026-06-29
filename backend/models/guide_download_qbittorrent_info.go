package models

// swagger:model guideDownloadQbittorrentInfo
type GuideDownloadQbittorrentInfo struct {

	// 配置目录
	ConfigPath string `json:"configPath,omitempty"`

	// 下载目录
	DownloadPath string `json:"downloadPath,omitempty"`

	// 运行状态
	// Enum: [running stopped not installed]
	Status string `json:"status,omitempty"`

	// webUI拼接，例如192.168.100.1+$webPath
	WebPath string `json:"webPath,omitempty"`
}

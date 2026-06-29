package models

// swagger:model guideQbittorrentInitRequest
type GuideQbittorrentInitRequest struct {

	// 配置目录
	ConfigPath string `json:"configPath,omitempty"`

	// 下载路径
	DownloadPath string `json:"downloadPath,omitempty"`
}

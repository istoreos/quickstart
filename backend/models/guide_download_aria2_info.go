package models

// swagger:model guideDownloadAria2Info
type GuideDownloadAria2Info struct {

	// 配置目录
	ConfigPath string `json:"configPath,omitempty"`

	// 下载目录
	DownloadPath string `json:"downloadPath,omitempty"`

	// rpc端口
	RPCPort uint32 `json:"rpcPort,omitempty"`

	// rpc令牌
	RPCToken string `json:"rpcToken,omitempty"`

	// aria2运行状态
	// Enum: [running stopped not installed]
	Status string `json:"status,omitempty"`

	// webUI拼接，例如192.168.100.1+$webPath
	WebPath string `json:"webPath,omitempty"`
}

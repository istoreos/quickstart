package models

// swagger:model guideAria2InitRequest
type GuideAria2InitRequest struct {

	// bt服务器，为空则自动请求服务器去配置，用,分隔可以添加多个
	// Example: http://1337.abcvg.info:80/announce,http://milanesitracker.tekcities.com:80/announce
	BtTracker string `json:"btTracker,omitempty"`

	// 配置目录
	ConfigPath string `json:"configPath,omitempty"`

	// aria2下载路径
	DownloadPath string `json:"downloadPath,omitempty"`

	// rpc密钥
	RPCToken string `json:"rpcToken,omitempty"`
}

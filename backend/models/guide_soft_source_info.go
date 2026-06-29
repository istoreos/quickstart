package models

// swagger:model guideSoftSourceInfo
type GuideSoftSourceInfo struct {

	// 软件源id
	// Enum: [OpenWrtHttp OpenWrtHttps Tsinghua USTC Alibaba Cloud Tencent Cloud]
	Identity string `json:"identity,omitempty"`

	// 软件源名称
	Name string `json:"name,omitempty"`

	// 软件源链接
	URL string `json:"url,omitempty"`
}

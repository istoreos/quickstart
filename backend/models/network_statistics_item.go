package models

// swagger:model networkStatisticsItem
type NetworkStatisticsItem struct {

	// 该时段下载平均网速，单位Byte
	DownloadSpeed int64 `json:"downloadSpeed,omitempty"`

	// 统计结束时间
	EndTime int64 `json:"endTime,omitempty"`

	// 统计开始时间
	StartTime int64 `json:"startTime,omitempty"`

	// 该时段上传平均网速，单位Byte
	UploadSpeed int64 `json:"uploadSpeed,omitempty"`
}

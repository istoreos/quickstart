package models

// swagger:model appCheckRequest
type AppCheckRequest struct {

	// 显示是否运行
	CheckRunning bool `json:"checkRunning,omitempty"`

	// 插件名称
	Name string `json:"name,omitempty"`
}

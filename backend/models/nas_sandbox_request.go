package models

// swagger:model nasSandboxRequest
type NasSandboxRequest struct {

	// 分区路径
	// Example: /dev/sda1
	Path string `json:"path,omitempty"`
}

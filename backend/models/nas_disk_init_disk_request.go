package models

// swagger:model nasDiskInitDiskRequest
type NasDiskInitDiskRequest struct {

	// name
	// Example: sda
	Name string `json:"name,omitempty"`

	// 硬盘路径
	// Example: /dev/sda
	Path string `json:"path,omitempty"`
}

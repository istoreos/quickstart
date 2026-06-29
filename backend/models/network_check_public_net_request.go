package models

// swagger:model networkCheckPublicNetRequest
type NetworkCheckPublicNetRequest struct {

	// ip版本类型
	// Enum: [ipv4 ipv6]
	IPVersion string `json:"ipVersion,omitempty"`
}

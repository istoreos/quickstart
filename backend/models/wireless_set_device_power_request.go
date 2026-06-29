package models

// swagger:model wirelessSetDevicePowerRequest
type WirelessSetDevicePowerRequest struct {

	// device
	Device string `json:"device,omitempty"`

	// 发射功率 [100, 70, 50, 30] => [Max, High, Medium, Low]
	Txpower int64 `json:"txpower,omitempty"`
}

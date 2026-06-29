package models

// swagger:model lANCtrlSpeedLimitModule
type LANCtrlSpeedLimitModule struct {

	// download speed
	DownloadSpeed int64 `json:"downloadSpeed,omitempty"`

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// installed
	Installed bool `json:"installed,omitempty"`

	// upload speed
	UploadSpeed int64 `json:"uploadSpeed,omitempty"`
}

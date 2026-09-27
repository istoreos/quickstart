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

	// provider is diagnostic metadata for the advanced settings area.
	Provider string `json:"provider,omitempty"`

	Providers []*RateLimitProviderOption `json:"providers,omitempty"`
}

type RateLimitProviderOption struct {
	ID        string `json:"id"`
	Available bool   `json:"available"`
	Installed bool   `json:"installed"`
	Reason    string `json:"reason,omitempty"`
}

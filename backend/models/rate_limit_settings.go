package models

type RateLimitSettings struct {
	Enabled       bool   `json:"enabled"`
	UploadSpeed   int64  `json:"uploadSpeed"`
	DownloadSpeed int64  `json:"downloadSpeed"`
	Provider      string `json:"provider"`
}

type RateLimitSettingsRequest struct {
	Settings        *RateLimitSettings `json:"settings"`
	ExpectedVersion string             `json:"expectedVersion,omitempty"`
	IdempotencyKey  string             `json:"idempotencyKey,omitempty"`
}

type RateLimitSettingsResult struct {
	Current        *RateLimitSettings  `json:"current,omitempty"`
	Desired        *RateLimitSettings  `json:"desired,omitempty"`
	Changes        []*PolicyPlanChange `json:"changes"`
	ReloadServices []string            `json:"reloadServices"`
	Version        string              `json:"version"`
	RollbackPoint  string              `json:"rollbackPoint"`
	CanApply       bool                `json:"canApply"`
	Changed        bool                `json:"changed,omitempty"`
	Error          *DevicePolicyError  `json:"error,omitempty"`
	Transaction    *TaskTransaction    `json:"transaction,omitempty"`
}

type RateLimitSettingsResponse struct {
	JSONResponse
	Result *RateLimitSettingsResult `json:"result,omitempty"`
}

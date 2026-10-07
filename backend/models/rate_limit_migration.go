package models

type RateLimitMigrationItem struct {
	ID                     string             `json:"id"`
	MAC                    string             `json:"mac"`
	Disposition            string             `json:"disposition"`
	Reason                 string             `json:"reason,omitempty"`
	UploadBytesPerSecond   int64              `json:"uploadBytesPerSecond"`
	DownloadBytesPerSecond int64              `json:"downloadBytesPerSecond"`
	UploadBitsPerSecond    int64              `json:"uploadBitsPerSecond,omitempty"`
	DownloadBitsPerSecond  int64              `json:"downloadBitsPerSecond,omitempty"`
	Desired                *DeviceSpeedPolicy `json:"desired,omitempty"`
}

type RateLimitMigrationPlan struct {
	Version          string                    `json:"version"`
	Source           string                    `json:"source"`
	Target           string                    `json:"target"`
	CanApply         bool                      `json:"canApply"`
	Items            []*RateLimitMigrationItem `json:"items"`
	ConvertibleCount int                       `json:"convertibleCount"`
	UnsupportedCount int                       `json:"unsupportedCount"`
	ConflictCount    int                       `json:"conflictCount"`
	RequiredActions  []string                  `json:"requiredActions"`
	Error            *DevicePolicyError        `json:"error,omitempty"`
}

type RateLimitMigrationRequest struct {
	ExpectedVersion string                  `json:"expectedVersion"`
	Plan            *RateLimitMigrationPlan `json:"plan"`
}

type RateLimitMigrationRollbackRequest struct {
	MigrationID string `json:"migrationId"`
}

type RateLimitMigrationResult struct {
	Plan        *RateLimitMigrationPlan `json:"plan,omitempty"`
	MigrationID string                  `json:"migrationId,omitempty"`
	Changed     bool                    `json:"changed"`
	RolledBack  bool                    `json:"rolledBack,omitempty"`
	Error       *DevicePolicyError      `json:"error,omitempty"`
}

type RateLimitMigrationResponse struct {
	JSONResponse
	Result *RateLimitMigrationResult `json:"result,omitempty"`
}

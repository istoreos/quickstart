package models

type LanDeviceMigrationItem struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Source      string `json:"source"`
	Target      string `json:"target"`
	Disposition string `json:"disposition"`
	Summary     string `json:"summary"`
	Reason      string `json:"reason,omitempty"`
}

type LanDeviceMigrationPlan struct {
	ModelVersion    int                       `json:"modelVersion"`
	Mode            string                    `json:"mode"`
	Version         string                    `json:"version"`
	CanApply        bool                      `json:"canApply"`
	AlreadyApplied  bool                      `json:"alreadyApplied"`
	Items           []*LanDeviceMigrationItem `json:"items"`
	ConflictCount   int                       `json:"conflictCount"`
	UnresolvedCount int                       `json:"unresolvedCount"`
	SnapshotPaths   []string                  `json:"snapshotPaths"`
	Error           *DevicePolicyError        `json:"error,omitempty"`
}

type LanDeviceMigrationRequest struct {
	ExpectedVersion string `json:"expectedVersion"`
}

type LanDeviceMigrationResult struct {
	Plan    *LanDeviceMigrationPlan `json:"plan,omitempty"`
	Changed bool                    `json:"changed"`
	Error   *DevicePolicyError      `json:"error,omitempty"`
}

type LanDeviceMigrationResponse struct {
	JSONResponse
	Result *LanDeviceMigrationResult `json:"result,omitempty"`
}

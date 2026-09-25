package models

type GroupSpeedPolicy struct {
	Enabled       bool  `json:"enabled"`
	UploadSpeed   int64 `json:"uploadSpeed"`
	DownloadSpeed int64 `json:"downloadSpeed"`
}

type GroupQuotaPolicy struct {
	Enabled    bool   `json:"enabled"`
	Period     string `json:"period"`
	LimitBytes int64  `json:"limitBytes"`
	Action     string `json:"action"`
}

type GroupSchedule struct {
	ID          string            `json:"id"`
	Enabled     bool              `json:"enabled"`
	Days        []int             `json:"days"`
	StartMinute int               `json:"startMinute"`
	EndMinute   int               `json:"endMinute"`
	Action      string            `json:"action"`
	Speed       *GroupSpeedPolicy `json:"speed,omitempty"`
}

type GroupPolicy struct {
	Access    *bool             `json:"access,omitempty"`
	Speed     *GroupSpeedPolicy `json:"speed,omitempty"`
	TargetID  string            `json:"targetId,omitempty"`
	Quota     *GroupQuotaPolicy `json:"quota,omitempty"`
	Schedules []*GroupSchedule  `json:"schedules"`
}

type DeviceGroup struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Priority int          `json:"priority"`
	Members  []string     `json:"members"`
	Policy   *GroupPolicy `json:"policy"`
}

type EffectiveGroupPolicy struct {
	DeviceID      string            `json:"deviceId"`
	NetworkAccess bool              `json:"networkAccess"`
	Speed         *GroupSpeedPolicy `json:"speed"`
	TargetID      string            `json:"targetId"`
	Quota         *GroupQuotaPolicy `json:"quota,omitempty"`
	Managed       []string          `json:"managed"`
	Sources       []string          `json:"sources"`
	Reasons       []string          `json:"reasons"`
	EvaluatedAt   string            `json:"evaluatedAt"`
}

type DeviceGroupEvent struct {
	At       string `json:"at"`
	DeviceID string `json:"deviceId,omitempty"`
	State    string `json:"state"`
	Reason   string `json:"reason,omitempty"`
}

type DeviceGroupBatchItem struct {
	DeviceID string   `json:"deviceId"`
	Status   string   `json:"status"`
	Reason   string   `json:"reason,omitempty"`
	Sources  []string `json:"sources"`
}

type DeviceGroupBatchResult struct {
	Status      string                  `json:"status"`
	Planned     int                     `json:"planned"`
	Applied     int                     `json:"applied"`
	Skipped     int                     `json:"skipped"`
	Failed      int                     `json:"failed"`
	Reloads     map[string]int          `json:"reloads"`
	Items       []*DeviceGroupBatchItem `json:"items"`
	EvaluatedAt string                  `json:"evaluatedAt"`
	DurationMS  int64                   `json:"durationMs"`
}

type DeviceGroupsResult struct {
	GlobalPolicy     *GroupPolicy            `json:"globalPolicy"`
	Groups           []*DeviceGroup          `json:"groups"`
	DevicePolicies   map[string]*GroupPolicy `json:"devicePolicies"`
	Effective        []*EffectiveGroupPolicy `json:"effective"`
	Events           []*DeviceGroupEvent     `json:"events"`
	Timezone         string                  `json:"timezone"`
	NextEvaluationAt string                  `json:"nextEvaluationAt"`
	Version          string                  `json:"version"`
	Error            *DevicePolicyError      `json:"error,omitempty"`
	LastBatch        *DeviceGroupBatchResult `json:"lastBatch,omitempty"`
}

type DeviceGroupsResponse struct {
	JSONResponse
	Result *DeviceGroupsResult `json:"result,omitempty"`
}

type DeviceGroupMutationRequest struct {
	Action          string       `json:"action"`
	Group           *DeviceGroup `json:"group,omitempty"`
	GroupID         string       `json:"groupId,omitempty"`
	DeviceID        string       `json:"deviceId,omitempty"`
	GlobalPolicy    *GroupPolicy `json:"globalPolicy,omitempty"`
	DevicePolicy    *GroupPolicy `json:"devicePolicy,omitempty"`
	ExpectedVersion string       `json:"expectedVersion,omitempty"`
}

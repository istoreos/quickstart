package models

type TrafficInsightBucket struct {
	Start         string `json:"start"`
	UploadBytes   int64  `json:"uploadBytes"`
	DownloadBytes int64  `json:"downloadBytes"`
}

type TrafficQuota struct {
	DeviceID   string `json:"deviceId"`
	Enabled    bool   `json:"enabled"`
	Period     string `json:"period"`
	LimitBytes int64  `json:"limitBytes"`
	Action     string `json:"action"`
}

type TrafficQuotaStatus struct {
	Quota       *TrafficQuota `json:"quota,omitempty"`
	UsedBytes   int64         `json:"usedBytes"`
	Exceeded    bool          `json:"exceeded"`
	PeriodStart string        `json:"periodStart,omitempty"`
	NextResetAt string        `json:"nextResetAt,omitempty"`
}

type TrafficInsightsCapability struct {
	State       CapabilityState `json:"state"`
	Adapter     string          `json:"adapter"`
	Reason      string          `json:"reason,omitempty"`
	Limitations []string        `json:"limitations"`
}

type TrafficInsightsResult struct {
	DeviceID       string                     `json:"deviceId"`
	Range          string                     `json:"range"`
	Buckets        []*TrafficInsightBucket    `json:"buckets"`
	UploadBytes    int64                      `json:"uploadBytes"`
	DownloadBytes  int64                      `json:"downloadBytes"`
	Quota          *TrafficQuotaStatus        `json:"quota"`
	Local          *TrafficInsightsCapability `json:"local"`
	Bandix         *TrafficInsightsCapability `json:"bandix"`
	Timezone       string                     `json:"timezone"`
	StorageBytes   int64                      `json:"storageBytes"`
	StorageBudget  int64                      `json:"storageBudget"`
	WriteIntervalS int64                      `json:"writeIntervalSeconds"`
	Error          *DevicePolicyError         `json:"error,omitempty"`
}

type TrafficInsightsResponse struct {
	JSONResponse
	Result *TrafficInsightsResult `json:"result,omitempty"`
}

type TrafficQuotaRequest struct {
	DeviceID   string `json:"deviceId"`
	Enabled    bool   `json:"enabled"`
	Period     string `json:"period"`
	LimitBytes int64  `json:"limitBytes"`
	Action     string `json:"action"`
}

package models

// DeviceRuntimeDiagnostics intentionally contains aggregate runtime counters
// only. Device identifiers, addresses, names and traffic contents must never be
// exposed by this endpoint.
type DeviceRuntimeDiagnostics struct {
	Inventory *DeviceInventoryRuntimeDiagnostics `json:"inventory"`
	Traffic   *DeviceTrafficRuntimeDiagnostics   `json:"traffic"`
	Sampler   *DeviceSamplerRuntimeDiagnostics   `json:"sampler"`
	OUI       *DeviceOUIRuntimeDiagnostics       `json:"oui"`
}

type DeviceInventoryRuntimeDiagnostics struct {
	CacheReady       bool  `json:"cacheReady"`
	CacheAgeMS       int64 `json:"cacheAgeMs"`
	CacheTTLMS       int64 `json:"cacheTtlMs"`
	RefreshInFlight  bool  `json:"refreshInFlight"`
	HistoryEntries   int64 `json:"historyEntries"`
	HistoryLimit     int64 `json:"historyLimit"`
	LastPersistAgeMS int64 `json:"lastPersistAgeMs"`
}

type DeviceTrafficRuntimeDiagnostics struct {
	AddressStates int64 `json:"addressStates"`
	TotalStates   int64 `json:"totalStates"`
	AddressLimit  int64 `json:"addressLimit"`
	TotalLimit    int64 `json:"totalLimit"`
}

type DeviceSamplerRuntimeDiagnostics struct {
	Sampling      bool  `json:"sampling"`
	ProcFallback  bool  `json:"procFallback"`
	HostCount     int64 `json:"hostCount"`
	FlowCount     int64 `json:"flowCount"`
	LastSampleMS  int64 `json:"lastSampleMs"`
	SampleCount   int64 `json:"sampleCount"`
	FailureCount  int64 `json:"failureCount"`
	EvictionCount int64 `json:"evictionCount"`
}

type DeviceOUIRuntimeDiagnostics struct {
	Entries        int64 `json:"entries"`
	SourceBytes    int64 `json:"sourceBytes"`
	LoadDurationMS int64 `json:"loadDurationMs"`
	HeapAllocBytes int64 `json:"heapAllocBytes"`
}

type DeviceRuntimeDiagnosticsResponse struct {
	JSONResponse
	Result *DeviceRuntimeDiagnostics `json:"result,omitempty"`
}

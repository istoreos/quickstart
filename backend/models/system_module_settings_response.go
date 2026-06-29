package models

// swagger:model systemModuleSettingsResponse
type SystemModuleSettingsResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemModuleSettingsResponseResult `json:"result"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}

// swagger:model systemModuleSettingsResponseResult
type SystemModuleSettingsResponseResult struct {
	// diableDisplay
	DiableDisplay []string `json:"diableDisplay"`
}

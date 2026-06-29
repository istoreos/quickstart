package models

// swagger:model SmartConfigResponse
type SmartConfigResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SmartConfigResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SmartConfigResponseResult
type SmartConfigResponseResult struct {

	// devices
	Devices []*SmartConfigDevice `json:"devices"`

	// global
	Global *SmartConfigGlobal `json:"global,omitempty"`

	// tasks
	Tasks []*SmartConfigTask `json:"tasks"`
}

package models

// swagger:model SmartConfigRequest
type SmartConfigRequest struct {

	// devices
	Devices []*SmartConfigDevice `json:"devices"`

	// global
	Global *SmartConfigGlobal `json:"global,omitempty"`

	// tasks
	Tasks []*SmartConfigTask `json:"tasks"`
}

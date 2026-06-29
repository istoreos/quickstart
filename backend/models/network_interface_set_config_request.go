package models

// swagger:model NetworkInterfaceSetConfigRequest
type NetworkInterfaceSetConfigRequest struct {

	// configs
	Configs []*NetworkInterfaceConfig `json:"configs"`
}

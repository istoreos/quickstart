package models

// swagger:model SmartInfo
type SmartInfo struct {

	// health
	Health string `json:"health,omitempty"`

	// model
	Model string `json:"model,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// nvme ver
	NvmeVer string `json:"nvmeVer,omitempty"`

	// path
	Path string `json:"path,omitempty"`

	// rota rate
	RotaRate string `json:"rotaRate,omitempty"`

	// sata ver
	SataVer string `json:"sataVer,omitempty"`

	// serial
	Serial string `json:"serial,omitempty"`

	// size str
	SizeStr string `json:"sizeStr,omitempty"`

	// status
	Status string `json:"status,omitempty"`

	// temp
	Temp string `json:"temp,omitempty"`
}

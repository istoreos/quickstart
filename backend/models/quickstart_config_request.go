package models

// swagger:model QuickstartConfigRequest
type QuickstartConfigRequest struct {

	// key
	Key string `json:"key,omitempty"`

	// type
	// Enum: [list option]
	Type string `json:"type,omitempty"`

	// values
	Values []string `json:"values"`
}

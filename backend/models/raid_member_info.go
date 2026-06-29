package models

// swagger:model raidMemberInfo
type RaidMemberInfo struct {

	// model
	Model string `json:"model,omitempty"`

	// name
	Name string `json:"name,omitempty"`

	// path
	Path string `json:"path,omitempty"`

	// size str
	SizeStr string `json:"sizeStr,omitempty"`
}

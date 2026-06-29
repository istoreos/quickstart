package models

type DHNSChangeRequest struct {
	Action string   `json:"action"`
	Params []string `json:"params"`
}

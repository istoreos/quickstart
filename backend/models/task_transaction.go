package models

// TaskTransaction exposes stable product semantics instead of internal error strings.
type TaskTransaction struct {
	Task           string `json:"task"`
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
	Stage          string `json:"stage"`
	Status         string `json:"status"`
	Replayed       bool   `json:"replayed,omitempty"`
	RecoveryAction string `json:"recoveryAction,omitempty"`
}

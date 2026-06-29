package models

// swagger:model lANCtrlStaticAssignedResponse
type LANCtrlStaticAssignedResponse struct {
	JSONResponse

	// result
	Result []*LANStaticAssigned `json:"result"`
}

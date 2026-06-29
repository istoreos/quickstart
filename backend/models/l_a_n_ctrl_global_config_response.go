package models

// swagger:model lANCtrlGlobalConfigResponse
type LANCtrlGlobalConfigResponse struct {
	JSONResponse

	// result
	Result *LANCtrlGlobalConfig `json:"result,omitempty"`
}

package models

// swagger:model systemCsrfTokenResponse
type SystemCsrfTokenResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *SystemCsrfTokenResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model SystemCsrfTokenResponseResult
type SystemCsrfTokenResponseResult struct {

	// token
	Token string `json:"token,omitempty"`
}

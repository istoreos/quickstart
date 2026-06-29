package models

// swagger:model sDKNormalResponse
type SDKNormalResponse struct {

	// detail
	Detail string `json:"detail,omitempty"`

	// error
	Error ResponseError `json:"error,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	// Required: true
	Success *ResponseSuccess `json:"success"`
}

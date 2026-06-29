package models

// swagger:model guideSoftSourceResponse
type GuideSoftSourceResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideSoftSourceResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideSoftSourceResponseResult
type GuideSoftSourceResponseResult struct {

	// soft source
	SoftSource *GuideSoftSourceInfo `json:"softSource,omitempty"`
}

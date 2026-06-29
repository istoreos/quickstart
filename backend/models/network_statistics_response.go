package models

// swagger:model networkStatisticsResponse
type NetworkStatisticsResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NetworkStatisticsResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NetworkStatisticsResponseResult
type NetworkStatisticsResponseResult struct {

	// items
	Items []*NetworkStatisticsItem `json:"items"`

	// slots
	Slots int64 `json:"slots,omitempty"`
}

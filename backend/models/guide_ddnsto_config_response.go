package models

// swagger:model guideDdnstoConfigResponse
type GuideDdnstoConfigResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideDdnstoConfigResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideDdnstoConfigResponseResult
type GuideDdnstoConfigResponseResult struct {

	// ddnsto_deviceId
	DeviceID string `json:"deviceId,omitempty"`

	// net_addr
	NetAddr string `json:"netAddr,omitempty"`
}

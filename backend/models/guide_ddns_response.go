package models

// swagger:model guideDdnsResponse
type GuideDdnsResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideDdnsResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideDdnsResponseResult
type GuideDdnsResponseResult struct {

	// ddnsto配置的域名
	DdnstoDomain string `json:"ddnstoDomain,omitempty"`

	// ipv4配置的域名
	IPV4Domain string `json:"ipv4Domain,omitempty"`

	// ipv6配置的域名
	IPV6Domain string `json:"ipv6Domain,omitempty"`
}

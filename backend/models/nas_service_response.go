package models

// swagger:model nasServiceResponse
type NasServiceResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *NasServiceResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model NasServiceResponseResult
type NasServiceResponseResult struct {

	// linkease
	Linkease *NasServiceLinkeaseInfo `json:"linkease,omitempty"`

	// sambas
	Sambas []*NasServiceSambaInfo `json:"sambas"`

	// webdav
	Webdav *NasServiceWebdavInfo `json:"webdav,omitempty"`
}

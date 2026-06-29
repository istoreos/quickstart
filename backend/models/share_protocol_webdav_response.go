package models

// swagger:model shareProtocolWebdavResponse
type ShareProtocolWebdavResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *ShareProtocolWebdavConfig `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}

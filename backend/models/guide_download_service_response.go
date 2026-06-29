package models

// swagger:model guideDownloadServiceResponse
type GuideDownloadServiceResponse struct {

	// error
	Error ResponseError `json:"error,omitempty"`

	// result
	Result *GuideDownloadServiceResponseResult `json:"result,omitempty"`

	// scope
	Scope ResponseScope `json:"scope,omitempty"`

	// success
	Success ResponseSuccess `json:"success,omitempty"`
}
// swagger:model GuideDownloadServiceResponseResult
type GuideDownloadServiceResponseResult struct {

	// aria2
	Aria2 *GuideDownloadAria2Info `json:"aria2,omitempty"`

	// qbittorrent
	Qbittorrent *GuideDownloadQbittorrentInfo `json:"qbittorrent,omitempty"`

	// transmission
	Transmission *GuideDownloadTransmissionInfo `json:"transmission,omitempty"`
}

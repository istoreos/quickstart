package downloadservices

import "github.com/linkease/quick-start/istore-backend/models"

type Aria2Snapshot struct {
	ConfigPath   string
	DownloadPath string
	RPCPort      uint32
	RPCToken     string
	Status       string
	WebPath      string
}

type QbittorrentSnapshot struct {
	ConfigPath   string
	DownloadPath string
	Status       string
	WebPath      string
}

type TransmissionSnapshot struct {
	ConfigPath   string
	DownloadPath string
	Status       string
	WebPath      string
}

func MapStatus(installed bool, running bool) string {
	if !installed {
		return "not installed"
	}
	if running {
		return "running"
	}
	return "stopped"
}

func BuildStatusResponse(
	aria2 *Aria2Snapshot,
	qbit *QbittorrentSnapshot,
	transmission *TransmissionSnapshot,
) *models.GuideDownloadServiceResponse {
	resp := models.GuideDownloadServiceResponse{}
	result := models.GuideDownloadServiceResponseResult{}
	resp.Result = &result
	if aria2 != nil {
		result.Aria2 = &models.GuideDownloadAria2Info{
			ConfigPath:   aria2.ConfigPath,
			DownloadPath: aria2.DownloadPath,
			RPCPort:      aria2.RPCPort,
			RPCToken:     aria2.RPCToken,
			Status:       aria2.Status,
			WebPath:      aria2.WebPath,
		}
	}
	if qbit != nil {
		result.Qbittorrent = &models.GuideDownloadQbittorrentInfo{
			ConfigPath:   qbit.ConfigPath,
			DownloadPath: qbit.DownloadPath,
			Status:       qbit.Status,
			WebPath:      qbit.WebPath,
		}
	}
	if transmission != nil {
		result.Transmission = &models.GuideDownloadTransmissionInfo{
			ConfigPath:   transmission.ConfigPath,
			DownloadPath: transmission.DownloadPath,
			Status:       transmission.Status,
			WebPath:      transmission.WebPath,
		}
	}
	return &resp
}

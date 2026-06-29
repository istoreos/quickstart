package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/utils"
)

type SimpleDiskInfo struct {
	Name         string `json:"name"`
	PhyCandidate string `json:"phyCandidate,omitempty"`
	Used         string `json:"used"`
	Total        string `json:"total"`
	UsedInt      string `json:"usedInt"`
	TotalInt     string `json:"totalInt"`
	IsRoot       bool   `json:"isRoot"`
	UsedPercent  int    `json:"usedPercent"`
}

type LcdSimpleResponse struct {
	DockerOk     bool             `json:"dockerOk,omitempty"`
	LinkeaseOk   bool             `json:"linkeaseOk,omitempty"`
	DomesticLink bool             `json:"domesticLink,omitempty"`
	ForeignLink  bool             `json:"foreignLink,omitempty"`
	Cpu          int              `json:"cpu"`
	Memory       int              `json:"memory"`
	Temperature  int              `json:"temp1"`
	Devices      int              `json:"devices"`
	NetErr       string           `json:"netErr,omitempty"`
	Upload       string           `json:"upload,omitempty"`
	Download     string           `json:"download,omitempty"`
	IPv4         string           `json:"ipv4,omitempty"%%`
	IPv6         string           `json:"ipv6,omitempty"`
	PublicIPv4   string           `json:"publicIpv4,omitempty"`
	DnsList      []string         `json:"dnsList,omitempty"`
	Domain       string           `json:"domain,omitempty"`
	Uptime       int64            `json:"uptime,omitempty"`
	UptimeHuman  string           `json:"uptimeHuman,omitempty"`
	DiskInfos    []SimpleDiskInfo `json:"disks,omitempty"`
}

func LcdSimple(ctx context.Context, r *http.Request, serviceBackend *ServiceBackend, ns *WanStats) (*LcdSimpleResponse, error) {
	d := LcdSimpleResponse{}
	s, err := serviceBackend.GetSystemStatus(ctx)
	if err != nil {
		return nil, err
	}
	d.Cpu = int(s.Result.CPUUsage)
	d.Temperature = int(s.Result.CPUTemperature)
	//d.FahrenheitTemperature = int(s.Result.CPUTemperature*9/5) + 32
	d.Memory = 100 - int(s.Result.MemAvailablePercentage)
	d.Uptime = s.Result.Uptime
	d.UptimeHuman = s.Result.UptimeHuman
	d.DockerOk = CheckAppIsRunning("dockerd")
	d.LinkeaseOk = CheckAppIsRunning("linkease")

	//network
	netw, err := NetworkStatus(ctx, serviceBackend.netChecker, false)
	if err != nil {
		return nil, err
	}
	d.IPv4 = netw.Result.Ipv4addr
	d.IPv6 = netw.Result.Ipv6addr
	d.DnsList = netw.Result.DNSList

	if v := r.URL.Query().Get("disk"); v == "" || v == "1" {
		//disk
		infos, err := getDiskForLCD(ctx, false)
		if err != nil {
			return nil, err
		}
		d.DiskInfos = infos
	}

	d.Upload = "0"
	d.Download = "0"
	//static
	stic, err := NetworkStatistic(ctx, ns)
	if err == nil {
		if len(stic.Result.Items) > 0 {
			lastItem := stic.Result.Items[len(stic.Result.Items)-1]
			d.Upload = utils.ByteCountDecimal(uint64(lastItem.UploadSpeed))
			d.Download = utils.ByteCountDecimal(uint64(lastItem.DownloadSpeed))
		}
	}
	// Enum: [netDetecting netSuccess dnsFailed netFailed softSourceFailed]
	d.NetErr = netw.Result.NetworkInfo
	if len(d.IPv4) == 0 {
		d.NetErr = "dhcpError"
	}
	foreignStatus, publicIP, countDevice := serviceBackend.foreignChecker.GetStatus(serviceBackend.httpClient)
	d.PublicIPv4 = publicIP
	d.Devices = countDevice
	if d.NetErr == "netSuccess" || d.PublicIPv4 != "" {
		d.DomesticLink = true
	}
	if foreignStatus == NetworkOnlineOK {
		d.ForeignLink = true
	}
	return &d, nil
}

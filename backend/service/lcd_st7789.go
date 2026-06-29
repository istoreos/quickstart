package service

import (
	"context"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/linkease/quick-start/istore-backend/utils"
)

type LcdSt7789Response struct {
	DockerOk              bool             `json:"dockerOk"`
	LinkeaseOk            bool             `json:"linkeaseOk"`
	Time                  string           `json:"time"`
	Date                  string           `json:"date"`
	Upload                string           `json:"upload"`
	Download              string           `json:"download"`
	NetErr                string           `json:"netErr,omitempty"`
	Cpu                   int              `json:"cpu"`
	Memory                int              `json:"memory"`
	Temperature           int              `json:"temp1"`
	FahrenheitTemperature int              `json:"temp2"`
	Ip                    string           `json:"ip"`
	DiskInfos             []SimpleDiskInfo `json:"disks,omitempty"`
}

func LcdSt7789(ctx context.Context, r *http.Request, serviceBackend *ServiceBackend, ns *WanStats) (*LcdSt7789Response, error) {
	d := LcdSt7789Response{}
	s, err := serviceBackend.GetSystemStatus(ctx)
	if err != nil {
		return nil, err
	}
	d.Cpu = int(s.Result.CPUUsage)
	d.Temperature = int(s.Result.CPUTemperature)
	d.FahrenheitTemperature = int(s.Result.CPUTemperature*9/5) + 32
	d.Memory = 100 - int(s.Result.MemAvailablePercentage)

	//d.DockerOk = CheckAppIsRunning("dockerd")

	//datetimeStr := strings.Split(s.Result.Localtime, " ")
	//if len(datetimeStr) != 2 {
	//	return nil, errors.New("time error")
	//}
	//d.Date = datetimeStr[0]
	//d.Time = datetimeStr[1]

	//nas, err := NasServiceStatus(ctx)
	//if err != nil {
	//	return nil, err
	//}
	//d.LinkeaseOk = nas.Result.Linkease.Enabel

	//network
	netw, err := NetworkStatus(ctx, serviceBackend.netChecker, false)
	if err != nil {
		return nil, err
	}
	d.Ip = netw.Result.Ipv4addr
	if len(d.Ip) == 0 {
		d.Ip = "DHCP error"
	}

	if v := r.URL.Query().Get("disk"); v == "" || v == "1" {
		//disk
		infos, err := getDiskForLCD(ctx, true)
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
	// DhcpError
	switch netw.Result.NetworkInfo {
	case "dnsFailed":
		d.NetErr = "DNSFail"
	case "netFailed":
		d.NetErr = "NetFail"
	case "softSourceFailed":
		d.NetErr = "SoftFail"
	default:
		d.NetErr = ""
	}
	return &d, nil
}

func getDiskForLCD(ctx context.Context, usePhy bool) ([]SimpleDiskInfo, error) {
	infos := make([]SimpleDiskInfo, 0)
	disks, err := NasDiskStatus(ctx)
	if err != nil {
		return nil, err
	}
	for _, disk := range disks.Result.Disks {
		i := SimpleDiskInfo{}
		i.IsRoot = disk.IsSystemRoot
		i.Name = disk.Name
		if usePhy {
			i.PhyCandidate = getPhyCandidate(disk.Path)
		}
		i64, _ := strconv.ParseInt(disk.SizeInt, 10, 64)
		i.TotalInt = strconv.FormatInt(i64, 10)
		i.Total = utils.ByteCountDecimal(uint64(i64))
		i64, _ = strconv.ParseInt(disk.UsedInt, 10, 64)
		i.UsedInt = strconv.FormatInt(i64, 10)
		i.Used = utils.ByteCountDecimal(uint64(i64))
		i.UsedPercent = int(disk.Usage)
		infos = append(infos, i)
	}

	// TODO for be3600, why disk is empty ?
	// Read root only
	if len(infos) == 0 {
		var stat syscall.Statfs_t
		err := syscall.Statfs("/", &stat)
		if err == nil {
			total := stat.Blocks*uint64(stat.Bsize) + 1
			available := stat.Bavail * uint64(stat.Bsize)
			used := total - available
			i := SimpleDiskInfo{}
			i.IsRoot = true
			i.Name = "Root"
			i.TotalInt = strconv.FormatInt(int64(total), 10)
			i.Total = utils.ByteCountDecimal(total)
			i.UsedInt = strconv.FormatInt(int64(used), 10)
			i.Used = utils.ByteCountDecimal(used)
			i.UsedPercent = int(used * 100 / total)
			infos = append(infos, i)
		}
	}

	sort.Slice(infos, func(i, j int) bool {
		if infos[i].IsRoot && infos[j].IsRoot || (!infos[i].IsRoot && !infos[j].IsRoot) {
			return infos[i].Name < infos[j].Name
		}
		if infos[j].IsRoot {
			return false
		}
		return true
	})
	return infos, nil
}

func getPhyCandidate(devPath string) string {
	cmd := exec.Command("/usr/libexec/blockphy.sh", devPath)
	b, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.Trim(string(b), "\n")
}

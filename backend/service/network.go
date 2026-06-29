package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	networkwanstats "github.com/linkease/quick-start/istore-backend/modules/network/wanstats"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type wanStatsSampler struct {
	stats *WanStats
}

func (sampler wanStatsSampler) Samples(ctx context.Context) ([]networkwanstats.Sample, error) {
	items := sampler.stats.GetItems()
	samples := make([]networkwanstats.Sample, 0, len(items))
	for _, item := range items {
		samples = append(samples, networkwanstats.Sample{
			StartTime:     item.startTime,
			EndTime:       item.endTime,
			UploadSpeed:   item.txAvg,
			DownloadSpeed: item.rxAvg,
		})
	}
	return samples, nil
}

func NetworkStatistic(ctx context.Context, ws *WanStats) (*models.NetworkStatisticsResponse, error) {
	return networkwanstats.NewService(wanStatsSampler{stats: ws}, int64(slots)).GetNetworkStatistic(ctx)
}

type ubusNetworkInterfaceStatus struct {
	Ipv4   []*ubusNetworkInterfaceAddress `json:"ipv4-address"`
	Ipv6   []*ubusNetworkInterfaceAddress `json:"ipv6-address"`
	Proto  string                         `json:"proto"`
	Route  []*ubusNetworkInterfaceRoute   `json:"route"`
	UpTime int64                          `json:"uptime"`
}

// NetworkStatus
func NetworkStatus(ctx context.Context, netChecker *NetworkOnlineChecker, setupFinish bool) (*models.NetworkStatusResponse, error) {
	return newNetworkStatusService(netChecker).GetNetworkStatus(ctx, setupFinish)
}

func markSetupFinish(ctx context.Context) {
	uci.LoadConfig("quickstart", true)
	if val, ok := uci.GetLast("quickstart", "main", "setup"); !ok || val != "1" {
		utils.BatchRun(ctx, []string{"uci -q set quickstart.main.setup=1", "uci commit quickstart"}, 0)
	}
}

func NetworkDeviceList(ctx context.Context) (*models.DeviceListResponse, error) {
	devices, err := newNetworkDeviceListService().List(ctx)
	if err != nil {
		return nil, err
	}
	model := models.DeviceListResponseResult{
		Devices: devices,
	}
	resp := models.DeviceListResponse{}
	resp.Result = &model
	return &resp, nil
}

// 如 255.255.255.0 对应的网络位长度为 24
func SubNetMaskToLen(netmask string) (int, error) {
	ipSplitArr := strings.Split(netmask, ".")
	if len(ipSplitArr) != 4 {
		return 0, fmt.Errorf("netmask:%v is not valid, pattern should like: 255.255.255.0", netmask)
	}
	ipv4MaskArr := make([]byte, 4)
	for i, value := range ipSplitArr {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("ipMaskToInt call strconv.Atoi error:[%v] string value is: [%s]", err, value)
		}
		if intValue > 255 {
			return 0, fmt.Errorf("netmask cannot greater than 255, current value is: [%s]", value)
		}
		ipv4MaskArr[i] = byte(intValue)
	}

	ones, _ := net.IPv4Mask(ipv4MaskArr[0], ipv4MaskArr[1], ipv4MaskArr[2], ipv4MaskArr[3]).Size()
	return ones, nil
}

// 如 24 对应的子网掩码地址为 255.255.255.0
func LenToSubNetMask(subnet int) string {
	var buff bytes.Buffer
	for i := 0; i < subnet; i++ {
		buff.WriteString("1")
	}
	for i := subnet; i < 32; i++ {
		buff.WriteString("0")
	}
	masker := buff.String()
	a, _ := strconv.ParseUint(masker[:8], 2, 64)
	b, _ := strconv.ParseUint(masker[8:16], 2, 64)
	c, _ := strconv.ParseUint(masker[16:24], 2, 64)
	d, _ := strconv.ParseUint(masker[24:32], 2, 64)
	resultMask := fmt.Sprintf("%v.%v.%v.%v", a, b, c, d)
	return resultMask
}

func NetworkHomeBoxEnable(ctx context.Context) (*models.NetworkHomeBoxEnableResponse, error) {
	return newHomeBoxEnableService().Enable(ctx)
}

func NetworkInterfaceStatus(ctx context.Context) (*models.NetworkInterfaceStatusResponse, error) {
	interfaces, err := newNetworkInterfaceInventoryService().ListInventory(ctx)
	if err != nil {
		return nil, err
	}
	return buildNetworkInterfaceStatusResult(interfaces), nil
}

func NetworkInterfaceSetConfig(ctx context.Context, input NetworkInterfaceWriteInput) (*models.SDKNormalResponse, error) {
	return newNetworkInterfaceConfigService().ApplyConfigSet(ctx, input)
}

func NetworkInterfacePostConfig(ctx context.Context, r *http.Request) (*models.SDKNormalResponse, error) {
	req := models.NetworkInterfaceSetConfigRequest{}
	err := getBody(&req, r)

	//config的device可以为空，屏蔽掉解析错误
	if err != nil {
		return nil, err
	}
	return NetworkInterfaceSetConfig(ctx, NetworkInterfaceWriteInput{
		Configs: req.Configs,
	})
}

func NetworkInterfaceGetConfig(ctx context.Context) (*models.NetworkInterfaceGetConfigResponse, error) {
	ports, err := readNetworkPortStatus(ctx)
	if err != nil {
		return nil, err
	}

	interfaces, err := newNetworkInterfaceInventoryService().ListInventory(ctx)
	if err != nil {
		return nil, err
	}
	return buildNetworkInterfaceGetConfigResult(ports, filterNetworkInterfaceGetConfig(interfaces)), nil
}

func NetworkPortList(ctx context.Context) (*models.NetworkPortListResponse, error) {
	return newNetworkPortListService().GetPortList(ctx)
}

func getSlavePorts(allPorts map[string]*models.NetworkPortInfo, deviceName string) []*models.NetworkPortInfo {
	slaves := make([]*models.NetworkPortInfo, 0)
	if len(deviceName) < 1 {
		return slaves
	}
	for _, v := range allPorts {
		if v.Master == deviceName {
			slaves = append(slaves, v)
		}
	}
	if len(slaves) == 0 {
		v, ok := allPorts[deviceName]
		if ok {
			slaves = append(slaves, v)
		}
	}
	return slaves
}

func getPortStatus(ctx context.Context) ([]*models.NetworkPortInfo, error) {
	interfaces, err := GetIpInterface()
	if err != nil {
		return nil, err
	}
	l := make([]*models.NetworkPortInfo, 0)
	for _, intr := range interfaces {
		if strings.HasPrefix(intr.IfName, "wlan") || (intr.LinkInfo != nil && intr.LinkInfo.Kind != "" && intr.LinkInfo.Kind != "dsa") {
			continue
		}
		if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%v/device/uevent", intr.IfName)); err != nil {
			continue
		}

		model := models.NetworkPortInfo{
			Name:       intr.IfName,
			LinkState:  intr.Operstate,
			Master:     intr.Master,
			MacAddress: intr.MacAddress,
		}
		deviceName := model.Name
		// if len(intr.Master) > 0 {
		// 	deviceName = intr.Master
		// }

		data, err := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%v/speed", deviceName))
		if err == nil {
			speedStr := strings.Trim(string(data), "\n")
			if speedStr != "-1" {
				model.LinkSpeed = fmt.Sprintf("%v Mbit/s", speedStr)
			}
		}
		data, err = ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%v/statistics/tx_bytes", deviceName))
		if err == nil {
			tx_bytesStr := strings.Trim(string(data), "\n")
			tx_bytes, _ := strconv.ParseUint(tx_bytesStr, 10, 64)
			tx_packets, _ := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%v/statistics/tx_packets", deviceName))
			tx_packetsStr := strings.Trim(string(tx_packets), "\n")
			model.TxPackets = fmt.Sprintf("%v (%vpkts.)", utils.ByteCountBinary(tx_bytes), tx_packetsStr)
		}

		data, err = ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%v/statistics/rx_bytes", deviceName))
		if err == nil {
			rx_bytesStr := strings.Trim(string(data), "\n")
			rx_bytes, _ := strconv.ParseUint(rx_bytesStr, 10, 64)
			rx_packets, _ := ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%v/statistics/rx_packets", deviceName))
			rx_packetsStr := strings.Trim(string(rx_packets), "\n")
			model.RxPackets = fmt.Sprintf("%v (%vpkts.)", utils.ByteCountBinary(rx_bytes), rx_packetsStr)
		}

		data, _ = ioutil.ReadFile(fmt.Sprintf("/sys/class/net/%v/duplex", deviceName))
		model.Duplex = strings.Trim(string(data), "\n")
		l = append(l, &model)
	}
	sort.Slice(l, func(i int, j int) bool {
		return l[i].Name < l[j].Name
	})
	return l, nil
}

type IpInterface struct {
	IfIndex    int       `json:"ifindex"`
	IfName     string    `json:"ifname"`
	Flags      []string  `json:"flags"`
	Operstate  string    `json:"operstate"`
	LinkType   string    `json:"link_type"`
	Master     string    `json:"master"`
	Qdisc      string    `json:"qdisc"`
	MacAddress string    `json:"address"`
	LinkInfo   *LinkInfo `json:"linkinfo"`
}

type LinkInfo struct {
	Kind string `json:"info_kind"`
}

func GetIpInterface() ([]*IpInterface, error) {
	cmd := exec.Command("ip", "-pretty", "-detail", "-json", "link", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var interfaces []*IpInterface
	err = json.Unmarshal(output, &interfaces)
	if err != nil {
		return nil, err
	}
	return interfaces, nil
}

func NetworkCheckPublicNet(ctx context.Context, r *http.Request) (*models.NetworkCheckPublicNetResponse, error) {
	req := models.NetworkCheckPublicNetRequest{}
	err := getBody(&req, r)
	if err != nil {
		return nil, errors.New("请求解析失败")
	}
	return newNetworkPublicAddressService().CheckPublicAddress(req.IPVersion)
}

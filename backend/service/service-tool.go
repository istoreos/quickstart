package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/bitly/go-simplejson"
)

var pool sync.Pool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 4096))
	},
}

type DefaultInterface struct {
	interfaceName string
	deviceName    string
	ip            string
	dns           []string
	mask          int
	gateway       string
	proto         string
	l3Device      string
	upTime        int64
}

type DefaultInterfaces struct {
	ipv4 *DefaultInterface
	ipv6 *DefaultInterface
}

func jsonArrayLen(j *simplejson.Json) int {
	a, err := j.Array()
	if err == nil {
		return len(a)
	}
	return 0
}

type ubusNetworkInterfaceRoute struct {
	Nexthop string `json:"nexthop"`
	Target  string `json:"target"`
	Mask    int    `json:"mask"`
	Source  string `json:"source"`
}

type ubusNetworkInterfaceAddress struct {
	Address string `json:"address"`
	Mask    int    `json:"mask"`
}

type ubusIPV6PrefixAssignment struct {
	Address      string                       `json:"address"`
	Mask         int                          `json:"mask"`
	LocalAddress *ubusNetworkInterfaceAddress `json:"local-address"`
}

type ubusNetworkInterface struct {
	Name      string                         `json:"interface"`
	Route     []*ubusNetworkInterfaceRoute   `json:"route"`
	Device    string                         `json:"device"`
	Proto     string                         `json:"proto"`
	L3Device  string                         `json:"l3_device"`
	Ipv4      []*ubusNetworkInterfaceAddress `json:"ipv4-address"`
	DnsServer []string                       `json:"dns-server"`
	Ipv6      []*ubusNetworkInterfaceAddress `json:"ipv6-address"`
	Ipv6PA    []*ubusIPV6PrefixAssignment    `json:"ipv6-prefix-assignment"`
	UpTime    int64                          `json:"uptime"`
	Ip4Table  *int                           `json:"ip4table"`
	Ip6Table  *int                           `json:"ip6table"`
}

type ubusNetworkInterfaceDump struct {
	Interfaces []*ubusNetworkInterface `json:"interface"`
}

func outboundInterfaces() (*DefaultInterfaces, error) {
	ctx := context.Background()

	var blk ubusNetworkInterfaceDump
	err := UbusCallWithObject(ctx, "network.interface dump", &blk)
	var ipv4 DefaultInterface
	var ipv6 DefaultInterface
	if err == nil {
		for _, iface := range blk.Interfaces {
			for _, route := range iface.Route {
				if ipv4.ip == "" &&
					(iface.Ip4Table == nil || 254 == *iface.Ip4Table) &&
					route.Target == "0.0.0.0" &&
					route.Mask == 0 &&
					len(iface.Ipv4) > 0 {
					result := &DefaultInterface{interfaceName: iface.Name,
						deviceName: iface.Device,
						l3Device:   iface.L3Device,
						proto:      iface.Proto,
						upTime:     iface.UpTime,
						ip:         iface.Ipv4[0].Address,
						mask:       iface.Ipv4[0].Mask,
						dns:        iface.DnsServer,
						gateway:    route.Nexthop}
					ipv4 = *result
				}
				//ipv6
				if ipv6.ip == "" &&
					(iface.Ip6Table == nil || 254 == *iface.Ip6Table) &&
					route.Target == "::" &&
					route.Mask == 0 &&
					(len(iface.Ipv6) > 0 ||
						(len(iface.Ipv6PA) > 0 && iface.Ipv6PA[0].LocalAddress != nil)) {
					var ipv6addr *ubusNetworkInterfaceAddress
					if len(iface.Ipv6) > 0 {
						ipv6addr = iface.Ipv6[0]
					} else {
						ipv6addr = iface.Ipv6PA[0].LocalAddress
					}
					result := &DefaultInterface{interfaceName: iface.Name,
						deviceName: iface.Device,
						l3Device:   iface.L3Device,
						proto:      iface.Proto,
						upTime:     iface.UpTime,
						ip:         ipv6addr.Address,
						mask:       ipv6addr.Mask,
						dns:        iface.DnsServer,
						gateway:    route.Nexthop}
					ipv6 = *result
				}
			}
			if ipv4.ip != "" && ipv6.ip != "" {
				break
			}
		}
	} else {
		return nil, errors.New("获取当前联网接口失败")
	}

	result := &DefaultInterfaces{ipv4: &ipv4, ipv6: &ipv6}
	return result, nil
}

func outboundInterface() (*DefaultInterface, error) {
	ctx := context.Background()

	var blk ubusNetworkInterfaceDump
	err := UbusCallWithObject(ctx, "network.interface dump", &blk)
	if err == nil {
		for _, iface := range blk.Interfaces {
			for _, route := range iface.Route {
				if route.Target == "0.0.0.0" &&
					route.Mask == 0 &&
					len(iface.Ipv4) > 0 {
					result := &DefaultInterface{interfaceName: iface.Name,
						deviceName: iface.Device,
						l3Device:   iface.L3Device,
						proto:      iface.Proto,
						upTime:     iface.UpTime,
						ip:         iface.Ipv4[0].Address,
						mask:       iface.Ipv4[0].Mask,
						dns:        iface.DnsServer,
						gateway:    route.Nexthop}
					return result, nil
				}
				//ipv6
				if route.Target == "::" &&
					route.Mask == 0 &&
					len(iface.Ipv6) > 0 {
					result := &DefaultInterface{interfaceName: iface.Name,
						deviceName: iface.Device,
						l3Device:   iface.L3Device,
						proto:      iface.Proto,
						upTime:     iface.UpTime,
						ip:         iface.Ipv6[0].Address,
						mask:       iface.Ipv6[0].Mask,
						dns:        iface.DnsServer,
						gateway:    route.Nexthop}
					return result, nil
				}
			}
		}
	} else {
		return nil, errors.New("获取当前联网接口失败")
	}

	result := &DefaultInterface{interfaceName: "wan", deviceName: "eth0"}
	return result, nil
}

type ubusLanStatus struct {
	network          ubusNetworkInterface
	isDefaultGateway bool
	lanAddr          string
	nexthop          string
}

func ubusGetLanStatus(ctx context.Context) (*ubusLanStatus, error) {
	var lanStatus ubusLanStatus
	err := UbusCallWithObject(ctx, "network.interface.lan status", &lanStatus.network)
	if err != nil {
		return nil, err
	}
	network := &lanStatus.network
	// 主路由模式
	if len(network.Ipv4) > 0 {
		intr := network.Ipv4[0]
		lanStatus.lanAddr = intr.Address
	}
	// 旁路由模式
	if len(network.Route) > 0 {
		router := network.Route[0]
		if router.Target == "0.0.0.0" &&
			router.Mask == 0 {
			lanStatus.isDefaultGateway = true
			lanStatus.nexthop = router.Nexthop
		}
	}
	return &lanStatus, nil
}

func Exists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

func getBody(v interface{}, r *http.Request) error {
	buffer := pool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer func() {
		if buffer != nil {
			pool.Put(buffer)
			buffer = nil
		}
	}()

	_, err := io.Copy(buffer, r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buffer.Bytes(), v)
	if err != nil {
		return err
	}
	return nil
}

func IsPublicIPV4(ipString string) bool {
	IP := net.ParseIP(ipString)
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := IP.To4(); ip4 != nil {
		switch true {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}

func IsPublicIPV6(ipString string) bool {
	IP := net.ParseIP(ipString)
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if IP != nil && strings.HasPrefix(ipString, "2") {
		return true
	}
	return false
}

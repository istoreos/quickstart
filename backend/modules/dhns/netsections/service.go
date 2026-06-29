package netsections

import (
	"net"
	"strings"
)

type Reader interface {
	Sections(config string, sectionType string) ([]string, bool)
	Last(config string, section string, option string) (string, bool)
}

type IPv4Resolver func(device string) (*net.IPNet, error)

type Section struct {
	Name    string
	Device  string
	Proto   string
	Gateway string
	IPNet   net.IPNet
	Up      bool
}

func Collect(reader Reader, resolveIPv4 IPv4Resolver) map[string]*Section {
	netSecs := make(map[string]*Section)
	secs, ok := reader.Sections("network", "interface")
	if !ok {
		return netSecs
	}
	for _, sec := range secs {
		if sec == "loopback" {
			continue
		}
		secInfo := &Section{Name: sec}
		if v, ok := reader.Last("network", sec, "device"); ok {
			secInfo.Device = v
		} else {
			continue
		}
		if v, ok := reader.Last("network", sec, "gateway"); ok {
			secInfo.Gateway = v
		}
		if v, ok := reader.Last("network", sec, "proto"); ok {
			secInfo.Proto = v
			if v == "dhcp" {
				if resolveIPv4 == nil {
					continue
				}
				ipnet, err := resolveIPv4(secInfo.Device)
				if err != nil {
					continue
				}
				secInfo.IPNet = *ipnet
			} else if v == "static" {
				if v, ok := reader.Last("network", sec, "ipaddr"); ok {
					if strings.Contains(v, "/") {
						if ip, cidr, err := net.ParseCIDR(v); err == nil {
							secInfo.IPNet.IP = ip
							secInfo.IPNet.Mask = cidr.Mask
						}
					} else {
						secInfo.IPNet.IP = net.ParseIP(v)
					}
					if secInfo.IPNet.IP == nil {
						continue
					}
				} else {
					continue
				}
				if v, ok := reader.Last("network", sec, "netmask"); ok {
					secInfo.IPNet.Mask = net.IPMask(net.ParseIP(v))
				} else if secInfo.IPNet.Mask == nil {
					continue
				}
			} else {
				continue
			}
		} else {
			continue
		}
		netSecs[sec] = secInfo
	}
	return netSecs
}

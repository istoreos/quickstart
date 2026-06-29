package utils

import (
	"errors"
	"net"
)

func Ipv4ToLong(ip net.IP) uint {
	i := ip.To4()
	v := uint(i[0])<<24 + uint(i[1])<<16 + uint(i[2])<<8 + uint(i[3])
	return v
}

func LongToIpv4(v uint) net.IP {
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	return net.IPv4(v0, v1, v2, v3)
}

func CalcStartAndLimit(startIpStr, endIpStr, netMaskStr string) (int, int, error) {
	startIP := net.ParseIP(startIpStr)
	endIP := net.ParseIP(endIpStr)
	maskIP := net.ParseIP(netMaskStr)
	if startIP == nil ||
		startIP.To4() == nil ||
		endIP == nil ||
		endIP.To4() == nil ||
		maskIP == nil ||
		maskIP.To4() == nil {
		return 0, 0, errors.New("invalid input")
	}
	if !IsValidIpv4Mask(maskIP) {
		return 0, 0, errors.New("invalid mask")
	}
	mask := net.IPMask(maskIP)
	startMask := Ipv4ToLong(startIP.Mask(mask))
	startLong := Ipv4ToLong(startIP)
	endLong := Ipv4ToLong(endIP)
	if endLong <= startLong {
		return 0, 0, errors.New("invalid start/end")
	}
	start := int(startLong - startMask)
	end := int(endLong - startLong)
	return start, end, nil
}

func IsValidIpv4Mask(ip net.IP) bool {
	mask := Ipv4ToLong(ip)
	var tmp uint
	for j := 31; j >= 0; j-- {
		if 0 == (mask & (1 << j)) {
			break
		}
		tmp = tmp + (1 << j)
	}
	if ((^tmp) & mask) != 0 {
		return false
	}
	return true
}

func GetInterfaceIpv4(interfaceName string) (addr *net.IPNet, err error) {
	var (
		ief      *net.Interface
		addrs    []net.Addr
		ipv4Addr net.IPNet
	)
	if ief, err = net.InterfaceByName(interfaceName); err != nil { // get interface
		return
	}
	if addrs, err = ief.Addrs(); err != nil { // get addresses
		return
	}
	var found bool
	for _, addr := range addrs { // get ipv4 address
		if ipv4 := addr.(*net.IPNet).IP.To4(); ipv4 != nil {
			ipv4Addr = *(addr.(*net.IPNet))
			found = true
			break
		}
	}
	if !found {
		return nil, errors.New("not found")
	}
	return &ipv4Addr, nil
}

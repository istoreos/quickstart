package utils

import (
	"fmt"
	"net"
	"testing"
)

func testIpStartLimit(startIpStr, endIpStr, netMaskStr string, s1, s2 int, t *testing.T) {
	v1, v2, err := CalcStartAndLimit(startIpStr, endIpStr, netMaskStr)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("calc", startIpStr, endIpStr, netMaskStr, "result", v1, v2)
	if v1 != s1 || v2 != s2 {
		t.Fatal("not equal")
	}
}

func TestCalcStartAndLimit(t *testing.T) {
	testIpStartLimit("192.168.100.100", "192.168.100.250", "255.255.255.0", 100, 150, t)
	// LAN: 192.168.0.1/16, DHCP: 1000, 253
	testIpStartLimit("192.168.3.232", "192.168.4.229", "255.255.0.0", 1000, 253, t)
}

func testMask(mask string, should bool, t *testing.T) {
	ip := net.ParseIP(mask)
	b := IsValidIpv4Mask(ip)
	fmt.Println("ip=", ip, "mask=", b)
	if b != should {
		t.Fatal("not equal")
	}
}

func TestIsValidMask(t *testing.T) {
	testMask("255.255.255.0", true, t)
	testMask("255.255.0.0", true, t)
	testMask("255.0.0.0", true, t)
	testMask("255.240.0.0", true, t)
	testMask("255.241.0.0", false, t)
	testMask("255.255.1.0", false, t)
	testMask("255.0.255.0", false, t)
}

func TestIpContains(t *testing.T) {
	ip1 := net.ParseIP("192.168.100.102")
	ip2 := net.ParseIP("192.168.101.103")
	ip3 := net.ParseIP("192.168.100.101")
	var ipLan, ipLan2 net.IPNet
	ipLan.IP = ip1
	ipLan.Mask = net.IPMask(net.ParseIP("255.255.255.0"))
	if ipLan.Contains(ip2) {
		t.Fatal("error")
	}
	if !ipLan.Contains(ip3) {
		t.Fatal("error 2")
	}
	ipLan2.IP = ip1
	ipLan2.Mask = net.IPMask(net.ParseIP("255.255.0.0"))
	if !ipLan2.Contains(ip3) {
		t.Fatal("error3")
	}
}

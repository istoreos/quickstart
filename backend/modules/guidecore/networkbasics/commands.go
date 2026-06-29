package networkbasics

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

type InterfaceInput struct {
	InterfaceName string
	Proto         string
	Netmask       string
	IP            string
	Gateway       string
}

func BuildPPPoECommands(account string, password string) []string {
	return []string{
		fmt.Sprintf("uci set network.wan.proto='%v'", "pppoe"),
		fmt.Sprintf("uci set network.wan.username='%v'", account),
		fmt.Sprintf("uci set network.wan.password='%v'", password),
		"uci del network.wan.dns",
		"uci del network.wan.peerdns",
		"uci del network.wan.defaultroute",
		"uci set network.lan.defaultroute=0",
	}
}

func BuildInterfaceCommandBatches(input InterfaceInput) ([][]string, error) {
	batches := [][]string{{
		"uci set network.wan.defaultroute=0",
		"uci set network.lan.defaultroute=0",
		fmt.Sprintf("uci del network.%v.defaultroute", input.InterfaceName),
		fmt.Sprintf("uci set network.%v.proto=%v", input.InterfaceName, input.Proto),
	}}

	if input.Proto == "dhcp" {
		batches = append(batches, []string{
			fmt.Sprintf("uci del network.%v.ipaddr", input.InterfaceName),
			fmt.Sprintf("uci del network.%v.netmask", input.InterfaceName),
		})
		return batches, nil
	}

	if net.ParseIP(input.IP) == nil {
		return batches, fmt.Errorf("not a valid IP")
	}
	mask, err := subnetMaskToLen(input.Netmask)
	if err != nil {
		return batches, err
	}
	if mask == 0 {
		return batches, fmt.Errorf("not a valid NetMask")
	}

	batches = append(batches, []string{
		fmt.Sprintf("uci del network.%v.ipaddr", input.InterfaceName),
		fmt.Sprintf("uci del network.%v.netmask", input.InterfaceName),
		fmt.Sprintf("uci set network.%v.ipaddr='%v'", input.InterfaceName, input.IP),
		fmt.Sprintf("uci set network.%v.netmask='%v'", input.InterfaceName, input.Netmask),
	})
	if len(input.Gateway) > 0 {
		batches = append(batches, []string{
			fmt.Sprintf("uci set network.%v.gateway=%v", input.InterfaceName, input.Gateway),
		})
	}
	return batches, nil
}

func BuildDNSCommandBatches(interfaceName string, dnsProto string, dnsIPs []string) [][]string {
	batches := [][]string{{
		"uci del network.wan.dns",
		"uci del network.wan.peerdns",
		"uci del network.lan.dns",
		"uci del network.lan.peerdns",
	}}

	if dnsProto != "manual" || len(dnsIPs) == 0 {
		return batches
	}

	cmdList := []string{
		fmt.Sprintf("uci del network.%v.dns", interfaceName),
		fmt.Sprintf("uci set network.%v.peerdns=0", interfaceName),
	}
	for _, dnsIP := range dnsIPs {
		if len(dnsIP) > 0 {
			cmdList = append(cmdList, fmt.Sprintf("uci add_list network.%v.dns='%v'", interfaceName, dnsIP))
		}
	}
	batches = append(batches, cmdList)
	return batches
}

func subnetMaskToLen(netmask string) (int, error) {
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

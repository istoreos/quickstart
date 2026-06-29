package networkbasics

import (
	"strconv"
	"strings"
)

func BuildLANRange(lanIP string, startStr string, limitStr string) (string, string) {
	ipSegs := strings.Split(lanIP, ".")
	if len(ipSegs) != 4 {
		return "", ""
	}
	startInt, _ := strconv.ParseUint(startStr, 10, 16)
	limitInt, _ := strconv.ParseUint(limitStr, 10, 16)
	ipSegs[len(ipSegs)-1] = startStr
	dhcpStart := strings.Join(ipSegs, ".")
	ipSegs[len(ipSegs)-1] = strconv.FormatUint(startInt+limitInt-1, 10)
	dhcpEnd := strings.Join(ipSegs, ".")
	return dhcpStart, dhcpEnd
}

package guideddns

import (
	"errors"
	"fmt"
)

type GuideDDNSApplyCommandInput struct {
	ConfigName  string
	UseIPv6     string
	ServiceName string
	Domain      string
	UserName    string
	Password    string
	Interface   string
	HasPublic   bool
	IPURL       string
}

type GuideDDNSRuntimeInterfaceSnapshot struct {
	InterfaceName string
	IP            string
	Public        bool
}

type GuideDDNSRuntimeSnapshot struct {
	IPv4 *GuideDDNSRuntimeInterfaceSnapshot
	IPv6 *GuideDDNSRuntimeInterfaceSnapshot
}

type GuideDDNSRuntimeResolution struct {
	ConfigName string
	UseIPv6    string
	Interface  string
	HasPublic  bool
	IPURL      string
}

func BuildGuideDDNSServiceName(serviceName string) (string, error) {
	switch serviceName {
	case "ali":
		return "aliyun.com", nil
	case "oray":
		return "oray.com", nil
	case "dnspod":
		return "dnspod.cn", nil
	default:
		return "", errors.New("serviceName参数错误" + serviceName)
	}
}

func ResolveGuideDDNSRuntime(ipVersion string, snapshot GuideDDNSRuntimeSnapshot) (GuideDDNSRuntimeResolution, error) {
	switch ipVersion {
	case "ipv4":
		if snapshot.IPv4 == nil {
			return GuideDDNSRuntimeResolution{}, errors.New("IPVersion参数错误" + ipVersion)
		}
		resolution := GuideDDNSRuntimeResolution{
			ConfigName: "myddns_ipv4",
			UseIPv6:    "0",
			Interface:  snapshot.IPv4.InterfaceName,
			HasPublic:  snapshot.IPv4.Public,
		}
		if !snapshot.IPv4.Public {
			resolution.IPURL = "4.ipw.cn"
		}
		return resolution, nil
	case "ipv6":
		if snapshot.IPv6 == nil {
			return GuideDDNSRuntimeResolution{}, errors.New("IPVersion参数错误" + ipVersion)
		}
		resolution := GuideDDNSRuntimeResolution{
			ConfigName: "myddns_ipv6",
			UseIPv6:    "1",
			Interface:  snapshot.IPv6.InterfaceName,
			HasPublic:  snapshot.IPv6.Public,
		}
		if !snapshot.IPv6.Public {
			resolution.IPURL = "6.ipw.cn"
		}
		return resolution, nil
	default:
		return GuideDDNSRuntimeResolution{}, errors.New("IPVersion参数错误" + ipVersion)
	}
}

func BuildGuideDDNSApplyCommands(input GuideDDNSApplyCommandInput) []string {
	cmds := []string{
		fmt.Sprintf("uci set ddns.%v=service", input.ConfigName),
		fmt.Sprintf("uci set ddns.%v.enabled='1'", input.ConfigName),
		fmt.Sprintf("uci set ddns.%v.use_ipv6=%v", input.ConfigName, input.UseIPv6),
		fmt.Sprintf("uci set ddns.%v.service_name=%v", input.ConfigName, input.ServiceName),
		fmt.Sprintf("uci set ddns.%v.lookup_host=%v", input.ConfigName, input.Domain),
		fmt.Sprintf("uci set ddns.%v.domain=%v", input.ConfigName, input.Domain),
		fmt.Sprintf("uci set ddns.%v.username=%v", input.ConfigName, input.UserName),
		fmt.Sprintf("uci set ddns.%v.password=%v", input.ConfigName, input.Password),
		fmt.Sprintf("uci set ddns.%v.interface=%v", input.ConfigName, input.Interface),
		fmt.Sprintf("uci set ddns.%v.use_syslog=%v", input.ConfigName, "2"),
		fmt.Sprintf("uci set ddns.%v.check_unit=%v", input.ConfigName, "minutes"),
		fmt.Sprintf("uci set ddns.%v.force_unit=%v", input.ConfigName, "minutes"),
		fmt.Sprintf("uci set ddns.%v.retry_unit=%v", input.ConfigName, "seconds"),
	}
	if input.HasPublic {
		cmds = append(cmds,
			fmt.Sprintf("uci set ddns.%v.ip_source=%v", input.ConfigName, "network"),
			fmt.Sprintf("uci set ddns.%v.ip_network=%v", input.ConfigName, input.Interface),
		)
	} else {
		cmds = append(cmds,
			fmt.Sprintf("uci set ddns.%v.ip_source=%v", input.ConfigName, "web"),
			fmt.Sprintf("uci set ddns.%v.ip_url=%v", input.ConfigName, input.IPURL),
			fmt.Sprintf("uci del ddns.%v.ip_network", input.ConfigName),
		)
	}
	cmds = append(cmds, "uci commit ddns")
	return cmds
}

package status

import (
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type IPv4Snapshot struct {
	Address       string
	Mask          int
	Proto         string
	Gateway       string
	UptimeSeconds int64
}

type Snapshot struct {
	IPv4           *IPv4Snapshot
	IPv6Addr       string
	ResolvedIfName string
}

type DNSConfig struct {
	Proto   string
	DNSList []string
}

type OnlineStatus string

const (
	OnlineDetecting        OnlineStatus = "netDetecting"
	OnlineFailedDNS        OnlineStatus = "dnsFailed"
	OnlineFailedOffline    OnlineStatus = "netFailed"
	OnlineFailedSoftSource OnlineStatus = "softSourceFailed"
	OnlineOK               OnlineStatus = "netSuccess"
	OnlineUnknown          OnlineStatus = "unknown"
)

func (status OnlineStatus) String() string {
	return string(status)
}

func ResolveDNSConfig(proto string, peerDNSSet bool, peerDNSValue string, outboundDNS []string, manualDNS []string) DNSConfig {
	if proto == "static" || (peerDNSSet && peerDNSValue == "0") {
		return DNSConfig{
			Proto:   "manual",
			DNSList: append([]string(nil), manualDNS...),
		}
	}

	return DNSConfig{
		Proto:   "auto",
		DNSList: append([]string(nil), outboundDNS...),
	}
}

func buildResult(snapshot Snapshot, dnsConfig DNSConfig) *models.NetworkStatusResponseResult {
	result := &models.NetworkStatusResponseResult{
		DefaultInterface: snapshot.ResolvedIfName,
		DNSList:          append([]string(nil), dnsConfig.DNSList...),
		DNSProto:         dnsConfig.Proto,
		Ipv6addr:         snapshot.IPv6Addr,
	}

	if snapshot.IPv4 != nil {
		result.Ipv4addr = snapshot.IPv4.Address
		result.Ipv4mask = int32(snapshot.IPv4.Mask)
		result.Proto = snapshot.IPv4.Proto
		result.Gateway = snapshot.IPv4.Gateway
		if snapshot.IPv4.UptimeSeconds > 0 {
			result.UptimeStamp = snapshot.IPv4.UptimeSeconds
			result.Uptime = utils.SecondsToHuman(snapshot.IPv4.UptimeSeconds)
		}
	}

	return result
}

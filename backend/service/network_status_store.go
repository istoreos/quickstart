package service

import (
	"context"

	"github.com/digineo/go-uci"
	networkstatus "github.com/linkease/quick-start/istore-backend/modules/network/status"
)

var networkStatusOutboundInterfaces = outboundInterfaces
var networkStatusLoadConfig = uci.LoadConfig
var networkStatusGetLast = uci.GetLast
var networkStatusGet = uci.Get
var networkStatusMarkSetupFinish = markSetupFinish

type NetworkStatusReader = networkstatus.Reader
type NetworkOnlineStatusChecker = networkstatus.Checker
type NetworkSetupMarker = networkstatus.SetupMarker

type defaultNetworkStatusReader struct{}

func newDefaultNetworkStatusReader() NetworkStatusReader {
	return &defaultNetworkStatusReader{}
}

func (reader *defaultNetworkStatusReader) Read(ctx context.Context) (NetworkStatusSnapshot, NetworkStatusDNSConfig, error) {
	defaultIfs, err := networkStatusOutboundInterfaces()
	if err != nil {
		return NetworkStatusSnapshot{}, NetworkStatusDNSConfig{}, err
	}

	ipv4 := fallbackNetworkStatusIPv4(defaultIfs.ipv4)
	manualDNS := []string(nil)
	networkStatusLoadConfig("network", true)
	peerDNSValue, peerDNSSet := networkStatusGetLast("network", ipv4.interfaceName, "peerdns")
	if ipv4.proto == "static" || (peerDNSSet && peerDNSValue == "0") {
		if values, ok := networkStatusGet("network", ipv4.interfaceName, "dns"); ok {
			manualDNS = values
		}
	}

	dnsConfig := networkstatus.ResolveDNSConfig(ipv4.proto, peerDNSSet, peerDNSValue, ipv4.dns, manualDNS)
	snapshot := NetworkStatusSnapshot{
		IPv4:           newNetworkStatusIPv4Snapshot(ipv4),
		ResolvedIfName: ipv4.interfaceName,
	}
	if defaultIfs.ipv6 != nil {
		snapshot.IPv6Addr = defaultIfs.ipv6.ip
	}

	_ = ctx
	return snapshot, dnsConfig, nil
}

type defaultNetworkOnlineStatusChecker struct {
	checker *NetworkOnlineChecker
}

func newDefaultNetworkOnlineStatusChecker(checker *NetworkOnlineChecker) NetworkOnlineStatusChecker {
	return &defaultNetworkOnlineStatusChecker{checker: checker}
}

func (checker *defaultNetworkOnlineStatusChecker) GetStatus(ip string, gateway string, dns []string) (networkstatus.OnlineStatus, error) {
	if checker == nil || checker.checker == nil {
		return networkstatus.OnlineDetecting, nil
	}
	return mapNetworkOnlineStatus(checker.checker.GetStatus(ip, gateway, dns)), nil
}

func mapNetworkOnlineStatus(status NetworkOnlineStatus) networkstatus.OnlineStatus {
	switch status {
	case NetworkOnlineDetech:
		return networkstatus.OnlineDetecting
	case NetworkOnlineFailedDns:
		return networkstatus.OnlineFailedDNS
	case NetworkOnlineFailedOffline:
		return networkstatus.OnlineFailedOffline
	case NetworkOnlineFailedSoftSource:
		return networkstatus.OnlineFailedSoftSource
	case NetworkOnlineOK:
		return networkstatus.OnlineOK
	default:
		return networkstatus.OnlineUnknown
	}
}

type defaultNetworkSetupMarker struct{}

func newDefaultNetworkSetupMarker() NetworkSetupMarker {
	return &defaultNetworkSetupMarker{}
}

func (marker *defaultNetworkSetupMarker) MarkSetupFinish(ctx context.Context) {
	networkStatusMarkSetupFinish(ctx)
}

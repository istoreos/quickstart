package service

import networkstatus "github.com/linkease/quick-start/istore-backend/modules/network/status"

type NetworkStatusSnapshot = networkstatus.Snapshot
type NetworkStatusDNSConfig = networkstatus.DNSConfig

func fallbackNetworkStatusIPv4(intr *DefaultInterface) *DefaultInterface {
	if intr == nil {
		return &DefaultInterface{interfaceName: "wan", deviceName: "eth0"}
	}
	if intr.interfaceName == "" {
		copy := *intr
		copy.interfaceName = "wan"
		if copy.deviceName == "" {
			copy.deviceName = "eth0"
		}
		return &copy
	}
	return intr
}

func newNetworkStatusIPv4Snapshot(intr *DefaultInterface) *networkstatus.IPv4Snapshot {
	if intr == nil {
		return nil
	}
	return &networkstatus.IPv4Snapshot{
		Address:       intr.ip,
		Mask:          intr.mask,
		Proto:         intr.proto,
		Gateway:       intr.gateway,
		UptimeSeconds: intr.upTime,
	}
}

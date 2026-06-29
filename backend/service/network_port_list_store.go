package service

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/portlist"
)

type NetworkPortStatusReader = portlist.StatusReader
type NetworkPortMembershipReader = portlist.MembershipReader

var readNetworkPortStatus = func(ctx context.Context) ([]*models.NetworkPortInfo, error) {
	return getPortStatus(ctx)
}

var readNetworkPortMembershipUbusCall = UbusCall

var readNetworkPortMembershipDump = func(ctx context.Context) ([]NetworkPortMembershipSnapshot, error) {
	interfaceDump, err := readNetworkPortMembershipUbusCall(ctx, "network.interface dump")
	if err != nil {
		return nil, err
	}
	if interfaceDump == nil {
		return nil, errors.New("network interface dump is empty")
	}

	interfaces := interfaceDump.Get("interface")
	memberships := make([]NetworkPortMembershipSnapshot, 0, jsonArrayLen(interfaces))
	for i, ifaceCount := 0, jsonArrayLen(interfaces); i < ifaceCount; i++ {
		iface := interfaces.GetIndex(i)
		memberships = append(memberships, NetworkPortMembershipSnapshot{
			InterfaceName: iface.Get("interface").MustString(),
			Device:        iface.Get("device").MustString(),
		})
	}
	return memberships, nil
}

type defaultNetworkPortStatusReader struct{}

func newDefaultNetworkPortStatusReader() NetworkPortStatusReader {
	return &defaultNetworkPortStatusReader{}
}

func (reader *defaultNetworkPortStatusReader) Read(ctx context.Context) ([]*models.NetworkPortInfo, error) {
	return readNetworkPortStatus(ctx)
}

type defaultNetworkPortMembershipReader struct{}

func newDefaultNetworkPortMembershipReader() NetworkPortMembershipReader {
	return &defaultNetworkPortMembershipReader{}
}

func (reader *defaultNetworkPortMembershipReader) Read(ctx context.Context) ([]NetworkPortMembershipSnapshot, error) {
	memberships, err := readNetworkPortMembershipDump(ctx)
	if err != nil {
		return nil, errors.New("获取网络接口失败")
	}

	filtered := make([]NetworkPortMembershipSnapshot, 0, len(memberships))
	for _, membership := range memberships {
		if membership.InterfaceName == "docker" || membership.InterfaceName == "loopback" {
			continue
		}
		filtered = append(filtered, membership)
	}
	return filtered, nil
}

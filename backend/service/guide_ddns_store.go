package service

import (
	"context"
	"fmt"

	simplejson "github.com/bitly/go-simplejson"
)

type GuideDDNSReader interface {
	ReadDDNSPendingChanges(ctx context.Context, sessionID string) (bool, error)
	ReadOutboundInterfaces(ctx context.Context) (*GuideDDNSOutboundSnapshot, error)
	IsPublicIPv4(ip string) bool
	IsPublicIPv6(ip string) bool
	ReadDdnstoConfig(ctx context.Context) (*GuideDdnstoConfigSnapshot, error)
}

var readGuideDDNSUbusCall = func(ctx context.Context, arg string) (*simplejson.Json, error) {
	return UbusCall(ctx, arg)
}

var readGuideDDNSOutboundInterfaces = func() (*DefaultInterfaces, error) {
	return outboundInterfaces()
}

var readGuideDDNSIsPublicIPv4 = IsPublicIPV4

var readGuideDDNSIsPublicIPv6 = IsPublicIPV6

var readGuideDDNSUciGet = func(ctx context.Context, location string) (string, error) {
	return uciGet(ctx, location)
}

type defaultGuideDDNSReader struct{}

func newDefaultGuideDDNSReader() *defaultGuideDDNSReader {
	return &defaultGuideDDNSReader{}
}

func (reader *defaultGuideDDNSReader) ReadDDNSPendingChanges(ctx context.Context, sessionID string) (bool, error) {
	json, err := readGuideDDNSUbusCall(ctx, fmt.Sprintf(`uci changes {"config":"ddns","ubus_rpc_session":"%s"}`, sessionID))
	if err != nil {
		return false, err
	}
	changes, err := json.Get("changes").Array()
	if err != nil {
		return false, err
	}
	return len(changes) > 0, nil
}

func (reader *defaultGuideDDNSReader) ReadOutboundInterfaces(ctx context.Context) (*GuideDDNSOutboundSnapshot, error) {
	interfaces, err := readGuideDDNSOutboundInterfaces()
	if err != nil {
		return nil, err
	}
	snapshot := &GuideDDNSOutboundSnapshot{}
	if interfaces.ipv4 != nil {
		snapshot.IPv4 = &GuideDDNSInterfaceSnapshot{
			InterfaceName: interfaces.ipv4.interfaceName,
			DeviceName:    interfaces.ipv4.deviceName,
			IP:            interfaces.ipv4.ip,
			Gateway:       interfaces.ipv4.gateway,
			Proto:         interfaces.ipv4.proto,
			DNS:           append([]string(nil), interfaces.ipv4.dns...),
		}
	}
	if interfaces.ipv6 != nil {
		snapshot.IPv6 = &GuideDDNSInterfaceSnapshot{
			InterfaceName: interfaces.ipv6.interfaceName,
			DeviceName:    interfaces.ipv6.deviceName,
			IP:            interfaces.ipv6.ip,
			Gateway:       interfaces.ipv6.gateway,
			Proto:         interfaces.ipv6.proto,
			DNS:           append([]string(nil), interfaces.ipv6.dns...),
		}
	}
	return snapshot, nil
}

func (reader *defaultGuideDDNSReader) IsPublicIPv4(ip string) bool {
	return readGuideDDNSIsPublicIPv4(ip)
}

func (reader *defaultGuideDDNSReader) IsPublicIPv6(ip string) bool {
	return readGuideDDNSIsPublicIPv6(ip)
}

func (reader *defaultGuideDDNSReader) ReadDdnstoConfig(ctx context.Context) (*GuideDdnstoConfigSnapshot, error) {
	enabled, err := readGuideDDNSUciGet(ctx, "ddnsto.@ddnsto[0].enabled")
	if err != nil {
		return nil, err
	}
	token, err := readGuideDDNSUciGet(ctx, "ddnsto.@ddnsto[0].token")
	if err != nil {
		return nil, err
	}
	address, err := readGuideDDNSUciGet(ctx, "ddnsto.@ddnsto[0].address")
	if err != nil {
		return nil, err
	}
	return &GuideDdnstoConfigSnapshot{
		Enabled: enabled == "1",
		Token:   token,
		Address: address,
	}, nil
}

package service

import "github.com/linkease/quick-start/istore-backend/modules/network/publicaddress"

var networkPublicAddressOutboundInterfaces = outboundInterfaces
var networkPublicAddressIsPublicIPv4 = IsPublicIPV4
var networkPublicAddressIsPublicIPv6 = IsPublicIPV6

type defaultNetworkPublicAddressReader struct{}

func newDefaultNetworkPublicAddressReader() publicaddress.Reader {
	return &defaultNetworkPublicAddressReader{}
}

func (reader *defaultNetworkPublicAddressReader) Read() (publicaddress.Snapshot, error) {
	interfaces, err := networkPublicAddressOutboundInterfaces()
	if err != nil {
		return publicaddress.Snapshot{}, err
	}

	snapshot := publicaddress.Snapshot{}
	if interfaces.ipv4 != nil {
		snapshot.IPv4 = interfaces.ipv4.ip
	}
	if interfaces.ipv6 != nil {
		snapshot.IPv6 = interfaces.ipv6.ip
	}
	return snapshot, nil
}

type defaultNetworkPublicAddressClassifier struct{}

func newDefaultNetworkPublicAddressClassifier() publicaddress.Classifier {
	return &defaultNetworkPublicAddressClassifier{}
}

func (classifier *defaultNetworkPublicAddressClassifier) IsPublic(ipVersion string, address string) bool {
	if ipVersion == "ipv6" {
		return networkPublicAddressIsPublicIPv6(address)
	}
	return networkPublicAddressIsPublicIPv4(address)
}

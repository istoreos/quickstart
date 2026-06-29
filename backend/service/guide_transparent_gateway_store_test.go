package service

import (
	"context"
	"sync"
	"testing"
)

var guideTransparentGatewayReaderTestMu sync.Mutex

func TestDefaultGuideTransparentGatewayReaderReadsSnapshot(t *testing.T) {
	guideTransparentGatewayReaderTestMu.Lock()
	defer guideTransparentGatewayReaderTestMu.Unlock()

	originalIPAndMask := readGuideTransparentGatewayIPAndMask
	originalConfigLast := readGuideTransparentGatewayConfigLast
	originalDhcpEnabled := readGuideTransparentGatewayLanDHCPEnabled
	defer func() {
		readGuideTransparentGatewayIPAndMask = originalIPAndMask
		readGuideTransparentGatewayConfigLast = originalConfigLast
		readGuideTransparentGatewayLanDHCPEnabled = originalDhcpEnabled
	}()

	readGuideTransparentGatewayIPAndMask = func(net string) (string, string) {
		if net != "lan" {
			t.Fatalf("unexpected network name: %s", net)
		}
		return "192.168.50.1", "255.255.255.0"
	}
	readGuideTransparentGatewayConfigLast = func(config, section, option string) (string, bool) {
		switch config + "." + section + "." + option {
		case "network.lan.gateway":
			return "192.168.50.254", true
		case "network.lan.dns":
			return "223.5.5.5", true
		default:
			return "", false
		}
	}
	readGuideTransparentGatewayLanDHCPEnabled = func() bool { return true }

	reader := newDefaultGuideTransparentGatewayReader()
	snapshot := reader.ReadTransparentGateway(context.Background())
	if snapshot.StaticLanIP != "192.168.50.1" || snapshot.SubnetMask != "255.255.255.0" || snapshot.Gateway != "192.168.50.254" || snapshot.StaticDNSIP != "223.5.5.5" || !snapshot.EnableDhcp {
		t.Fatalf("unexpected transparent gateway snapshot: %#v", snapshot)
	}
}

func TestDefaultGuideTransparentGatewayReaderHandlesMissingGatewayFields(t *testing.T) {
	guideTransparentGatewayReaderTestMu.Lock()
	defer guideTransparentGatewayReaderTestMu.Unlock()

	originalIPAndMask := readGuideTransparentGatewayIPAndMask
	originalConfigLast := readGuideTransparentGatewayConfigLast
	originalDhcpEnabled := readGuideTransparentGatewayLanDHCPEnabled
	defer func() {
		readGuideTransparentGatewayIPAndMask = originalIPAndMask
		readGuideTransparentGatewayConfigLast = originalConfigLast
		readGuideTransparentGatewayLanDHCPEnabled = originalDhcpEnabled
	}()

	readGuideTransparentGatewayIPAndMask = func(net string) (string, string) {
		return "10.0.0.1", "255.255.255.0"
	}
	readGuideTransparentGatewayConfigLast = func(config, section, option string) (string, bool) {
		return "", false
	}
	readGuideTransparentGatewayLanDHCPEnabled = func() bool { return false }

	reader := newDefaultGuideTransparentGatewayReader()
	snapshot := reader.ReadTransparentGateway(context.Background())
	if snapshot.StaticLanIP != "10.0.0.1" || snapshot.SubnetMask != "255.255.255.0" || snapshot.Gateway != "" || snapshot.StaticDNSIP != "" || snapshot.EnableDhcp {
		t.Fatalf("unexpected transparent gateway fallback snapshot: %#v", snapshot)
	}
}

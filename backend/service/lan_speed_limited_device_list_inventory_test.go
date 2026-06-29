package service

import (
	"context"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestBuildLanSpeedLimitedDeviceInventoryIndexesOnlineDevices(t *testing.T) {
	snapshot := buildLanSpeedLimitedDeviceInventory([]*models.DeviceInfo{
		{Mac: "AA:BB:CC:DD:EE:FF", Ipv4addr: "192.168.100.2"},
		{Mac: "11:22:33:44:55:66", Ipv4addr: "192.168.100.3"},
	})
	if got := len(snapshot.DeviceByMAC); got != 2 {
		t.Fatalf("len(DeviceByMAC) = %d, want 2", got)
	}
	if got := len(snapshot.DeviceByIP); got != 2 {
		t.Fatalf("len(DeviceByIP) = %d, want 2", got)
	}
	if snapshot.DeviceByMAC["AA:BB:CC:DD:EE:FF"].Ipv4addr != "192.168.100.2" {
		t.Fatal("expected mac index to preserve device info")
	}
}

func TestBuildLanSpeedLimitedDeviceInventorySkipsMissingKeys(t *testing.T) {
	snapshot := buildLanSpeedLimitedDeviceInventory([]*models.DeviceInfo{
		{Mac: "", Ipv4addr: "192.168.100.2"},
		{Mac: "AA:BB:CC:DD:EE:FF", Ipv4addr: ""},
		nil,
	})
	if len(snapshot.DeviceByMAC) != 0 || len(snapshot.DeviceByIP) != 0 {
		t.Fatalf("expected empty indexes, got %+v", snapshot)
	}
}

func TestBuildSpeedLimitHostnameMapUsesBestEffortValues(t *testing.T) {
	hostnames := buildSpeedLimitHostnameMap([]speedLimitHostRecord{
		{MAC: "AA:BB:CC:DD:EE:FF", Name: "printer"},
		{MAC: "11:22:33:44:55:66", Name: ""},
	})
	if got := hostnames["AA:BB:CC:DD:EE:FF"]; got != "printer" {
		t.Fatalf("hostname = %q, want %q", got, "printer")
	}
	if got := hostnames["11:22:33:44:55:66"]; got != "" {
		t.Fatalf("hostname = %q, want empty string", got)
	}
}

func TestDefaultSpeedLimitHostnameReaderLoadsDhcpHosts(t *testing.T) {
	originalLoadConfig := lanSpeedLimitedDeviceListLoadConfig
	originalGetSections := lanSpeedLimitedDeviceListGetSections
	originalGetLast := lanSpeedLimitedDeviceListGetLast
	t.Cleanup(func() {
		lanSpeedLimitedDeviceListLoadConfig = originalLoadConfig
		lanSpeedLimitedDeviceListGetSections = originalGetSections
		lanSpeedLimitedDeviceListGetLast = originalGetLast
	})

	loadCalls := []string{}
	lanSpeedLimitedDeviceListLoadConfig = func(name string, forceReload bool) error {
		loadCalls = append(loadCalls, name)
		return nil
	}
	lanSpeedLimitedDeviceListGetSections = func(config, sectionType string) ([]string, bool) {
		if config != "dhcp" || sectionType != "host" {
			t.Fatalf("unexpected get sections call: %s %s", config, sectionType)
		}
		return []string{"host1", "host2"}, true
	}
	lanSpeedLimitedDeviceListGetLast = func(config, section, option string) (string, bool) {
		values := map[string]map[string]map[string]string{
			"dhcp": {
				"host1": {"mac": "aa:bb:cc:dd:ee:ff", "name": "printer"},
				"host2": {"mac": "11:22:33:44:55:66", "name": ""},
			},
		}
		if got, ok := values[config][section][option]; ok {
			return got, true
		}
		return "", false
	}

	reader := NewDefaultSpeedLimitHostnameReader()
	hostnames, err := reader.ReadHostnames(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(loadCalls) != 1 || loadCalls[0] != "dhcp" {
		t.Fatalf("unexpected load calls: %+v", loadCalls)
	}
	if got := hostnames["AA:BB:CC:DD:EE:FF"]; got != "printer" {
		t.Fatalf("hostname = %q, want %q", got, "printer")
	}
	if got := hostnames["11:22:33:44:55:66"]; got != "" {
		t.Fatalf("hostname = %q, want empty string", got)
	}
}

func TestDefaultSpeedLimitHostnameReaderReturnsEmptyMapWithoutHostSections(t *testing.T) {
	originalLoadConfig := lanSpeedLimitedDeviceListLoadConfig
	originalGetSections := lanSpeedLimitedDeviceListGetSections
	t.Cleanup(func() {
		lanSpeedLimitedDeviceListLoadConfig = originalLoadConfig
		lanSpeedLimitedDeviceListGetSections = originalGetSections
	})

	lanSpeedLimitedDeviceListLoadConfig = func(name string, forceReload bool) error {
		return nil
	}
	lanSpeedLimitedDeviceListGetSections = func(config, sectionType string) ([]string, bool) {
		return nil, false
	}

	reader := NewDefaultSpeedLimitHostnameReader()
	hostnames, err := reader.ReadHostnames(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(hostnames) != 0 {
		t.Fatalf("expected empty hostname map, got %+v", hostnames)
	}
}

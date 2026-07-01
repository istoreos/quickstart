package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/istoreos/quickstart/backend/models"
)

func TestDefaultNasServiceStatusReaderReadsUniShareServices(t *testing.T) {
	originalLoadConfig := loadNasServiceConfig
	originalGetLast := getNasServiceLast
	originalGetSections := getNasServiceSections
	originalGetValues := getNasServiceValues
	defer func() {
		loadNasServiceConfig = originalLoadConfig
		getNasServiceLast = originalGetLast
		getNasServiceSections = originalGetSections
		getNasServiceValues = originalGetValues
	}()

	var loadCalls []string
	loadNasServiceConfig = func(config string) {
		loadCalls = append(loadCalls, config)
		if config != "unishare" {
			t.Fatalf("unexpected config load: %q", config)
		}
	}
	getNasServiceSections = func(config string, sectionType string) ([]string, bool) {
		if config != "unishare" || sectionType != "share" {
			t.Fatalf("unexpected getSections call: %s %s", config, sectionType)
		}
		return []string{"@share[0]", "@share[1]", "@share[2]", "@share[3]"}, true
	}
	getNasServiceLast = func(config string, section string, option string) (string, bool) {
		if config != "unishare" {
			t.Fatalf("unexpected getLast config: %s", config)
		}
		values := map[string]map[string]string{
			"@share[0]": {
				"name": "samba-only",
				"path": "/mnt/samba",
			},
			"@share[1]": {
				"name": "webdav-only",
				"path": "/mnt/webdav",
			},
			"@share[2]": {
				"name": "both",
				"path": "/mnt/both",
			},
			"@share[3]": {
				"name": "no-proto",
				"path": "/mnt/no-proto",
			},
		}
		if option == "webdav_port" && section == "@global[0]" {
			return "", false
		}
		if sectionValues, ok := values[section]; ok {
			value, ok := sectionValues[option]
			return value, ok
		}
		t.Fatalf("unexpected getLast call: %s %s %s", config, section, option)
		return "", false
	}
	getNasServiceValues = func(config string, section string, option string) ([]string, bool) {
		if config != "unishare" || option != "proto" {
			t.Fatalf("unexpected getValues call: %s %s %s", config, section, option)
		}
		values := map[string][]string{
			"@share[0]": {"samba"},
			"@share[1]": {"webdav"},
			"@share[2]": {"samba", "webdav"},
		}
		value, ok := values[section]
		return value, ok
	}

	reader := newDefaultNasServiceStatusReader()

	shares := reader.ReadSambaShares()
	expectedShares := []*models.NasServiceSambaInfo{
		{ShareName: "samba-only", Path: "/mnt/samba"},
		{ShareName: "both", Path: "/mnt/both"},
	}
	if !reflect.DeepEqual(shares, expectedShares) {
		t.Fatalf("unexpected samba shares: %#v", shares)
	}
	port, ok := reader.ReadWebdavPort()
	if !ok || port != "8080" {
		t.Fatalf("unexpected webdav port: %q ok=%v", port, ok)
	}
	info := reader.ReadWebdavInfo()
	if info.Path != "/mnt/webdav" || info.Port != "8080" || info.Username != "" || info.Password != "" {
		t.Fatalf("unexpected WebDAV info: %#v", info)
	}
	expectedLoadCalls := []string{"unishare", "unishare", "unishare"}
	if !reflect.DeepEqual(loadCalls, expectedLoadCalls) {
		t.Fatalf("unexpected config loads: %#v", loadCalls)
	}
}

func TestDefaultNasServiceRuntimeReaderDelegatesToNetworkStatus(t *testing.T) {
	originalReadNetworkStatus := readNasServiceNetworkStatus
	defer func() {
		readNasServiceNetworkStatus = originalReadNetworkStatus
	}()

	reader := newDefaultNasServiceRuntimeReader()

	readNasServiceNetworkStatus = func(ctx context.Context) (*models.NetworkStatusResponse, error) {
		return &models.NetworkStatusResponse{
			Result: &models.NetworkStatusResponseResult{Ipv4addr: "192.168.100.1"},
		}, nil
	}
	ipv4, err := reader.ReadLANIPv4(context.Background())
	if err != nil {
		t.Fatalf("unexpected runtime reader error: %v", err)
	}
	if ipv4 != "192.168.100.1" {
		t.Fatalf("unexpected ipv4: %q", ipv4)
	}

	runtimeErr := errors.New("network status failed")
	readNasServiceNetworkStatus = func(ctx context.Context) (*models.NetworkStatusResponse, error) {
		return nil, runtimeErr
	}
	if _, err := reader.ReadLANIPv4(context.Background()); !errors.Is(err, runtimeErr) {
		t.Fatalf("expected runtime reader error, got %v", err)
	}
}

func TestDefaultNasServiceStatusReaderReadsConfiguredUniShareWebdavPort(t *testing.T) {
	originalLoadConfig := loadNasServiceConfig
	originalGetLast := getNasServiceLast
	defer func() {
		loadNasServiceConfig = originalLoadConfig
		getNasServiceLast = originalGetLast
	}()

	loadCalls := 0
	loadNasServiceConfig = func(config string) {
		loadCalls++
		if config != "unishare" {
			t.Fatalf("unexpected config load: %q", config)
		}
	}
	getNasServiceLast = func(config string, section string, option string) (string, bool) {
		if config == "unishare" && section == "@global[0]" && option == "webdav_port" {
			return "6086", true
		}
		t.Fatalf("unexpected getLast target: %s %s %s", config, section, option)
		return "", false
	}

	reader := newDefaultNasServiceStatusReader()
	port, ok := reader.ReadWebdavPort()
	if !ok || port != "6086" {
		t.Fatalf("unexpected WebDAV port: %q ok=%v", port, ok)
	}
	if loadCalls != 1 {
		t.Fatalf("expected one config load, got %d", loadCalls)
	}
}

func TestDefaultNasServiceStatusReaderReadsLinkeaseConfigFields(t *testing.T) {
	originalReadLinkeaseConfig := readNasServiceLinkeaseConfig
	defer func() {
		readNasServiceLinkeaseConfig = originalReadLinkeaseConfig
	}()

	readNasServiceLinkeaseConfig = func(ctx context.Context, key string) ([]byte, error) {
		switch key {
		case "preconfig":
			return []byte("01234567890\n"), nil
		case "port":
			return []byte("8897\n"), nil
		default:
			t.Fatalf("unexpected LinkEase key: %q", key)
			return nil, nil
		}
	}

	reader := newDefaultNasServiceStatusReader()
	enabledByConfig, port, err := reader.ReadLinkeaseInfo(context.Background())
	if err != nil {
		t.Fatalf("unexpected LinkEase read error: %v", err)
	}
	if !enabledByConfig || port != "8897" {
		t.Fatalf("unexpected LinkEase status: enabled=%v port=%q", enabledByConfig, port)
	}
}

func TestDefaultNasServiceStatusReaderTreatsMissingLinkeasePreconfigAsDisabled(t *testing.T) {
	originalReadLinkeaseConfig := readNasServiceLinkeaseConfig
	defer func() {
		readNasServiceLinkeaseConfig = originalReadLinkeaseConfig
	}()

	readNasServiceLinkeaseConfig = func(ctx context.Context, key string) ([]byte, error) {
		if key != "preconfig" {
			t.Fatalf("unexpected LinkEase key after missing preconfig: %q", key)
		}
		return nil, errors.New("exit status 1")
	}

	reader := newDefaultNasServiceStatusReader()
	enabledByConfig, port, err := reader.ReadLinkeaseInfo(context.Background())
	if err != nil {
		t.Fatalf("expected missing LinkEase preconfig to be non-fatal, got %v", err)
	}
	if enabledByConfig || port != "" {
		t.Fatalf("expected disabled LinkEase without preconfig, got enabled=%v port=%q", enabledByConfig, port)
	}
}

func TestDefaultNasServiceRuntimeReaderChecksLinkeaseBinary(t *testing.T) {
	originalHasBinary := hasNasServiceBinary
	defer func() {
		hasNasServiceBinary = originalHasBinary
	}()

	hasNasServiceBinary = func(path string) bool {
		if path != "/usr/sbin/linkease" {
			t.Fatalf("unexpected binary path: %q", path)
		}
		return true
	}

	reader := newDefaultNasServiceRuntimeReader()
	if !reader.HasLinkeaseBinary() {
		t.Fatalf("expected LinkEase binary to be reported present")
	}
}

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
)

var guideDownloadServicesReaderTestMu sync.Mutex

func TestDefaultGuideDownloadServicesReaderReadsAria2StatusSnapshot(t *testing.T) {
	guideDownloadServicesReaderTestMu.Lock()
	defer guideDownloadServicesReaderTestMu.Unlock()

	restore := stubGuideDownloadServiceReaderSeams()
	defer restore()

	readGuideDownloadServiceInstalled = func(ctx context.Context, initName string) (bool, error) {
		if initName != "aria2" {
			t.Fatalf("unexpected init service: %s", initName)
		}
		return true, nil
	}
	readGuideDownloadServiceRunning = func(processName string) bool {
		if processName != "aria2c" {
			t.Fatalf("unexpected process name: %s", processName)
		}
		return true
	}
	readGuideDownloadServiceConfig = func(ctx context.Context, location string) (string, error) {
		switch location {
		case "aria2.main.config_dir":
			return "/etc/aria2", nil
		case "aria2.main.dir":
			return "/mnt/data/download", nil
		case "aria2.main.rpc_secret":
			return "token", nil
		case "aria2.main.rpc_listen_port":
			return "", nil
		default:
			t.Fatalf("unexpected config location: %s", location)
			return "", nil
		}
	}

	reader := newDefaultGuideDownloadServicesReader()
	snapshot, err := reader.ReadAria2Status(context.Background())
	if err != nil {
		t.Fatalf("ReadAria2Status returned error: %v", err)
	}
	if snapshot.Status != "running" {
		t.Fatalf("expected running status, got %q", snapshot.Status)
	}
	if snapshot.ConfigPath != "/etc/aria2" || snapshot.DownloadPath != "/mnt/data/download" || snapshot.RPCToken != "token" {
		t.Fatalf("unexpected aria2 snapshot: %#v", snapshot)
	}
	if snapshot.RPCPort != 6800 {
		t.Fatalf("expected default rpc port 6800, got %d", snapshot.RPCPort)
	}
	if snapshot.WebPath != "/ariang" {
		t.Fatalf("expected /ariang web path, got %q", snapshot.WebPath)
	}
}

func TestDefaultGuideDownloadServicesReaderReadsQbittorrentStatusSnapshot(t *testing.T) {
	guideDownloadServicesReaderTestMu.Lock()
	defer guideDownloadServicesReaderTestMu.Unlock()

	restore := stubGuideDownloadServiceReaderSeams()
	defer restore()

	readGuideDownloadServiceInstalled = func(ctx context.Context, initName string) (bool, error) {
		if initName != "qbittorrent" {
			t.Fatalf("unexpected init service: %s", initName)
		}
		return true, nil
	}
	readGuideDownloadServiceRunning = func(processName string) bool {
		if processName != "qbittorrent" {
			t.Fatalf("unexpected process name: %s", processName)
		}
		return false
	}
	readGuideDownloadServiceConfig = func(ctx context.Context, location string) (string, error) {
		switch location {
		case "qbittorrent.main.profile":
			return "/etc/qbittorrent", nil
		case "qbittorrent.main.SavePath":
			return "/mnt/data/qbit", nil
		case "qbittorrent.main.Port":
			return "8080", nil
		default:
			t.Fatalf("unexpected config location: %s", location)
			return "", nil
		}
	}

	reader := newDefaultGuideDownloadServicesReader()
	snapshot, err := reader.ReadQbittorrentStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadQbittorrentStatus returned error: %v", err)
	}
	if snapshot.Status != "stopped" {
		t.Fatalf("expected stopped status, got %q", snapshot.Status)
	}
	if snapshot.ConfigPath != "/etc/qbittorrent" || snapshot.DownloadPath != "/mnt/data/qbit" || snapshot.WebPath != ":8080" {
		t.Fatalf("unexpected qbittorrent snapshot: %#v", snapshot)
	}
}

func TestDefaultGuideDownloadServicesReaderReadsTransmissionStatusSnapshot(t *testing.T) {
	guideDownloadServicesReaderTestMu.Lock()
	defer guideDownloadServicesReaderTestMu.Unlock()

	restore := stubGuideDownloadServiceReaderSeams()
	defer restore()

	readGuideDownloadServiceInstalled = func(ctx context.Context, initName string) (bool, error) {
		if initName != "transmission" {
			t.Fatalf("unexpected init service: %s", initName)
		}
		return false, nil
	}

	reader := newDefaultGuideDownloadServicesReader()
	snapshot, err := reader.ReadTransmissionStatus(context.Background())
	if err != nil {
		t.Fatalf("ReadTransmissionStatus returned error: %v", err)
	}
	if snapshot.Status != "not installed" {
		t.Fatalf("expected not installed status, got %q", snapshot.Status)
	}
	if snapshot.ConfigPath != "" || snapshot.DownloadPath != "" || snapshot.WebPath != "" {
		t.Fatalf("unexpected transmission snapshot: %#v", snapshot)
	}
}

func TestDefaultGuideDownloadServicesReaderPropagatesInstallErrors(t *testing.T) {
	guideDownloadServicesReaderTestMu.Lock()
	defer guideDownloadServicesReaderTestMu.Unlock()

	restore := stubGuideDownloadServiceReaderSeams()
	defer restore()

	readGuideDownloadServiceInstalled = func(ctx context.Context, initName string) (bool, error) {
		return false, errors.New("boom")
	}

	reader := newDefaultGuideDownloadServicesReader()
	if _, err := reader.ReadAria2Status(context.Background()); err == nil {
		t.Fatal("expected install error")
	}
}

func stubGuideDownloadServiceReaderSeams() func() {
	originalInstalled := readGuideDownloadServiceInstalled
	originalRunning := readGuideDownloadServiceRunning
	originalConfig := readGuideDownloadServiceConfig
	return func() {
		readGuideDownloadServiceInstalled = originalInstalled
		readGuideDownloadServiceRunning = originalRunning
		readGuideDownloadServiceConfig = originalConfig
	}
}

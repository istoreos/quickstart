package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	downloadservices "github.com/linkease/quick-start/istore-backend/modules/guidestorage/downloadservices"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideDownloadServicesReader interface {
	ReadAria2Status(ctx context.Context) (*GuideDownloadAria2Snapshot, error)
	ReadQbittorrentStatus(ctx context.Context) (*GuideDownloadQbittorrentSnapshot, error)
	ReadTransmissionStatus(ctx context.Context) (*GuideDownloadTransmissionSnapshot, error)
}

var readGuideDownloadServiceInstalled = func(ctx context.Context, initName string) (bool, error) {
	out, err := utils.BatchOutputCmd(ctx, fmt.Sprintf("[ -e /etc/init.d/%s ] && echo true || echo false", initName), 0)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

var readGuideDownloadServiceRunning = func(processName string) bool {
	return CheckAppIsRunning(processName)
}

var readGuideDownloadServiceConfig = func(ctx context.Context, location string) (string, error) {
	return uciGet(ctx, location)
}

type defaultGuideDownloadServicesReader struct{}

func newDefaultGuideDownloadServicesReader() *defaultGuideDownloadServicesReader {
	return &defaultGuideDownloadServicesReader{}
}

func (reader *defaultGuideDownloadServicesReader) ReadAria2Status(ctx context.Context) (*GuideDownloadAria2Snapshot, error) {
	installed, err := readGuideDownloadServiceInstalled(ctx, "aria2")
	if err != nil {
		return nil, err
	}

	snapshot := &GuideDownloadAria2Snapshot{
		Status:  mapGuideDownloadServiceStatus(installed, readGuideDownloadServiceRunning("aria2c")),
		WebPath: "/ariang",
	}
	if !installed {
		return snapshot, nil
	}

	snapshot.ConfigPath, _ = readGuideDownloadServiceConfig(ctx, "aria2.main.config_dir")
	snapshot.DownloadPath, _ = readGuideDownloadServiceConfig(ctx, "aria2.main.dir")
	snapshot.RPCToken, _ = readGuideDownloadServiceConfig(ctx, "aria2.main.rpc_secret")
	rpcPort, _ := readGuideDownloadServiceConfig(ctx, "aria2.main.rpc_listen_port")
	if rpcPort != "" {
		if rpcPortInt, parseErr := strconv.ParseUint(rpcPort, 10, 32); parseErr == nil {
			snapshot.RPCPort = uint32(rpcPortInt)
		}
	}
	if snapshot.RPCPort == 0 {
		snapshot.RPCPort = 6800
	}
	return snapshot, nil
}

func (reader *defaultGuideDownloadServicesReader) ReadQbittorrentStatus(ctx context.Context) (*GuideDownloadQbittorrentSnapshot, error) {
	installed, err := readGuideDownloadServiceInstalled(ctx, "qbittorrent")
	if err != nil {
		return nil, err
	}

	snapshot := &GuideDownloadQbittorrentSnapshot{
		Status: mapGuideDownloadServiceStatus(installed, readGuideDownloadServiceRunning("qbittorrent")),
	}
	if !installed {
		return snapshot, nil
	}

	snapshot.ConfigPath, _ = readGuideDownloadServiceConfig(ctx, "qbittorrent.main.profile")
	snapshot.DownloadPath, _ = readGuideDownloadServiceConfig(ctx, "qbittorrent.main.SavePath")
	if port, _ := readGuideDownloadServiceConfig(ctx, "qbittorrent.main.Port"); port != "" {
		snapshot.WebPath = ":" + port
	}
	return snapshot, nil
}

func (reader *defaultGuideDownloadServicesReader) ReadTransmissionStatus(ctx context.Context) (*GuideDownloadTransmissionSnapshot, error) {
	installed, err := readGuideDownloadServiceInstalled(ctx, "transmission")
	if err != nil {
		return nil, err
	}

	snapshot := &GuideDownloadTransmissionSnapshot{
		Status: mapGuideDownloadServiceStatus(installed, readGuideDownloadServiceRunning("transmission")),
	}
	if !installed {
		return snapshot, nil
	}

	snapshot.ConfigPath, _ = readGuideDownloadServiceConfig(ctx, "transmission.@transmission[0].config_dir")
	snapshot.DownloadPath, _ = readGuideDownloadServiceConfig(ctx, "transmission.@transmission[0].download_dir")
	if port, _ := readGuideDownloadServiceConfig(ctx, "transmission.@transmission[0].rpc_port"); port != "" {
		snapshot.WebPath = ":" + port
	}
	return snapshot, nil
}

func mapGuideDownloadServiceStatus(installed bool, running bool) string {
	return downloadservices.MapStatus(installed, running)
}

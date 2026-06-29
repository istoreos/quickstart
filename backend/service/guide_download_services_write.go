package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	downloadservices "github.com/linkease/quick-start/istore-backend/modules/guidestorage/downloadservices"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideDownloadServicesWriter interface {
	ValidateDownloadPath(path string) error
	EnsureDownloadDir(ctx context.Context, path string) error
	CanAccessPath(path string) bool
	WriteAria2Config(ctx context.Context, input GuideAria2InitInput) error
	WriteAria2Trackers(ctx context.Context, trackers []string) error
	RestartAria2(ctx context.Context) error
	WriteQbittorrentConfig(ctx context.Context, input GuideQbittorrentInitInput) error
	RestartQbittorrent(ctx context.Context) error
	WriteTransmissionConfig(ctx context.Context, input GuideTransmissionInitInput) error
	RestartTransmission(ctx context.Context) error
}

type GuideDownloadServicesRuntime interface {
	ResolveAria2Trackers(ctx context.Context, rawTrackers string) ([]string, error)
}

var validateGuideDownloadServicePath = func(path string) error {
	return downloadservices.ValidateDownloadPath(path)
}

var createGuideDownloadServiceDir = func(ctx context.Context, path string) error {
	return utils.BatchRun(ctx, downloadservices.BuildEnsureDownloadDirCommands(path), 0)
}

var accessGuideDownloadServicePath = func(path string) bool {
	return canAccessPath(path)
}

var applyGuideDownloadServiceCommands = func(ctx context.Context, cmds []string) error {
	return utils.BatchRun(ctx, cmds, 0)
}

var fetchGuideDownloadServiceAria2Trackers = func(ctx context.Context) (string, error) {
	c := http.Client{Timeout: 2 * time.Second}
	resp, err := c.Get("https://trackerslist.com/best_aria2.txt")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type defaultGuideDownloadServicesWriter struct{}

type defaultGuideDownloadServicesRuntime struct{}

func newDefaultGuideDownloadServicesWriter() *defaultGuideDownloadServicesWriter {
	return &defaultGuideDownloadServicesWriter{}
}

func newDefaultGuideDownloadServicesRuntime() *defaultGuideDownloadServicesRuntime {
	return &defaultGuideDownloadServicesRuntime{}
}

func (writer *defaultGuideDownloadServicesWriter) ValidateDownloadPath(path string) error {
	return validateGuideDownloadServicePath(path)
}

func (writer *defaultGuideDownloadServicesWriter) EnsureDownloadDir(ctx context.Context, path string) error {
	return createGuideDownloadServiceDir(ctx, path)
}

func (writer *defaultGuideDownloadServicesWriter) CanAccessPath(path string) bool {
	return accessGuideDownloadServicePath(path)
}

func (writer *defaultGuideDownloadServicesWriter) WriteAria2Config(ctx context.Context, input GuideAria2InitInput) error {
	return applyGuideDownloadServiceCommands(ctx, downloadservices.BuildAria2ConfigCommands(input))
}

func (writer *defaultGuideDownloadServicesWriter) WriteAria2Trackers(ctx context.Context, trackers []string) error {
	for _, cmds := range downloadservices.BuildAria2TrackerCommandBatches(trackers) {
		if err := applyGuideDownloadServiceCommands(ctx, cmds); err != nil {
			return err
		}
	}
	return nil
}

func (writer *defaultGuideDownloadServicesWriter) RestartAria2(ctx context.Context) error {
	return applyGuideDownloadServiceCommands(ctx, downloadservices.BuildAria2RestartCommands())
}

func (writer *defaultGuideDownloadServicesWriter) WriteQbittorrentConfig(ctx context.Context, input GuideQbittorrentInitInput) error {
	return applyGuideDownloadServiceCommands(ctx, downloadservices.BuildQbittorrentConfigCommands(input))
}

func (writer *defaultGuideDownloadServicesWriter) RestartQbittorrent(ctx context.Context) error {
	return applyGuideDownloadServiceCommands(ctx, downloadservices.BuildQbittorrentRestartCommands())
}

func (writer *defaultGuideDownloadServicesWriter) WriteTransmissionConfig(ctx context.Context, input GuideTransmissionInitInput) error {
	return applyGuideDownloadServiceCommands(ctx, downloadservices.BuildTransmissionConfigCommands(input))
}

func (writer *defaultGuideDownloadServicesWriter) RestartTransmission(ctx context.Context) error {
	return applyGuideDownloadServiceCommands(ctx, downloadservices.BuildTransmissionRestartCommands())
}

func (runtime *defaultGuideDownloadServicesRuntime) ResolveAria2Trackers(ctx context.Context, rawTrackers string) ([]string, error) {
	if rawTrackers != "" {
		return downloadservices.ParseTrackers(rawTrackers), nil
	}
	body, err := fetchGuideDownloadServiceAria2Trackers(ctx)
	if err != nil {
		return nil, errors.New("请求btTacker列表失败，请检查设备网络后，重试或手动配置")
	}
	return downloadservices.ParseTrackers(body), nil
}

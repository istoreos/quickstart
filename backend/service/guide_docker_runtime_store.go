package service

import (
	"context"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideDockerRuntimeReader interface {
	ReadDockerRuntime(ctx context.Context) (*GuideDockerRuntimeSnapshot, error)
}

var readGuideDockerInstalled = func(ctx context.Context) (bool, error) {
	out, err := utils.BatchOutputCmd(ctx, "which dockerd", 0)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) != "", nil
}

var readGuideDockerRunning = func() bool {
	return CheckAppIsRunning("dockerd")
}

var readGuideDockerRootPath = func(ctx context.Context) (string, error) {
	out, err := utils.BatchOutputCmd(ctx, "uci get dockerd.globals.data_root", 0)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

var readGuideDockerDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
	return getAllDisks(ctx)
}

type defaultGuideDockerRuntimeReader struct{}

func newDefaultGuideDockerRuntimeReader() *defaultGuideDockerRuntimeReader {
	return &defaultGuideDockerRuntimeReader{}
}

func (reader *defaultGuideDockerRuntimeReader) ReadDockerRuntime(ctx context.Context) (*GuideDockerRuntimeSnapshot, error) {
	installed, err := readGuideDockerInstalled(ctx)
	if err != nil {
		return nil, err
	}

	snapshot := &GuideDockerRuntimeSnapshot{
		Installed: installed,
	}
	if !installed {
		return snapshot, nil
	}

	snapshot.Running = readGuideDockerRunning()
	snapshot.Path, err = readGuideDockerRootPath(ctx)
	if err != nil {
		return nil, err
	}
	disks, err := readGuideDockerDisks(ctx)
	if err != nil {
		return nil, err
	}
	snapshot.ErrorInfo = buildGuideDockerRuntimeWarning(disks)
	return snapshot, nil
}

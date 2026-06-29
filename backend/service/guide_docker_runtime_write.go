package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideDockerRuntimeWriter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

var runGuideDockerStart = func(ctx context.Context) error {
	_, err := utils.BatchOutputCmd(ctx, "/etc/init.d/dockerd start", 0)
	return err
}

var runGuideDockerStop = func(ctx context.Context) error {
	_, err := utils.BatchOutputCmd(ctx, "/etc/init.d/dockerd stop", 0)
	return err
}

type defaultGuideDockerRuntimeWriter struct{}

func newDefaultGuideDockerRuntimeWriter() *defaultGuideDockerRuntimeWriter {
	return &defaultGuideDockerRuntimeWriter{}
}

func (writer *defaultGuideDockerRuntimeWriter) Start(ctx context.Context) error {
	return runGuideDockerStart(ctx)
}

func (writer *defaultGuideDockerRuntimeWriter) Stop(ctx context.Context) error {
	return runGuideDockerStop(ctx)
}

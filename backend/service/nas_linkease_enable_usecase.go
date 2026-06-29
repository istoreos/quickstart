package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/nas/linkease"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type NasLinkeaseConfigReader = linkease.ConfigReader
type NasLinkeaseConfigWriter = linkease.ConfigWriter
type NasLinkeaseEnableService = linkease.Service

var readNasLinkeaseConfig = func(ctx context.Context, key string) ([]byte, error) {
	return utils.BatchOutputCmd(ctx, fmt.Sprintf("uci get linkease.@linkease[0].%s", key), 0)
}

var runNasLinkeaseEnable = func(ctx context.Context, cmdList []string) error {
	return utils.BatchRun(ctx, cmdList, 0)
}

type defaultNasLinkeaseConfigReader struct{}

func (reader *defaultNasLinkeaseConfigReader) ReadEnabled(ctx context.Context) (string, error) {
	ret, err := readNasLinkeaseConfig(ctx, "enabled")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ret)), nil
}

func (reader *defaultNasLinkeaseConfigReader) ReadPort(ctx context.Context) (string, error) {
	ret, err := readNasLinkeaseConfig(ctx, "port")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ret)), nil
}

type defaultNasLinkeaseConfigWriter struct{}

func (writer *defaultNasLinkeaseConfigWriter) Enable(ctx context.Context) error {
	return runNasLinkeaseEnable(ctx, []string{
		"uci set linkease.@linkease[0].enabled=1",
		"uci commit linkease",
		"/etc/init.d/linkease restart",
	})
}

type nasLinkeaseEnableFacade interface {
	Enable(ctx context.Context) (*models.NasLinkeaseEnableResponseResult, error)
}

var newNasLinkeaseEnableServiceFacade = func() nasLinkeaseEnableFacade {
	return newNasLinkeaseEnableService()
}

func newNasLinkeaseEnableService() *linkease.Service {
	return linkease.NewService(&defaultNasLinkeaseConfigReader{}, &defaultNasLinkeaseConfigWriter{})
}

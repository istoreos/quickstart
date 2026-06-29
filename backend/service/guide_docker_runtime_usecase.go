package service

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type guideDockerRuntimeFacade interface {
	GetStatus(ctx context.Context) (*models.GuideDockerStatusResponse, error)
	Switch(ctx context.Context, enable bool) (*models.SDKNormalResponse, error)
}

var newGuideDockerRuntimeFacade = func() guideDockerRuntimeFacade {
	return newGuideDockerRuntimeService()
}

type GuideDockerRuntimeService struct {
	reader GuideDockerRuntimeReader
	writer GuideDockerRuntimeWriter
}

func newGuideDockerRuntimeService() *GuideDockerRuntimeService {
	return &GuideDockerRuntimeService{
		reader: newDefaultGuideDockerRuntimeReader(),
		writer: newDefaultGuideDockerRuntimeWriter(),
	}
}

func (service *GuideDockerRuntimeService) GetStatus(ctx context.Context) (*models.GuideDockerStatusResponse, error) {
	snapshot, err := service.reader.ReadDockerRuntime(ctx)
	if err != nil {
		return nil, err
	}

	status := "not installed"
	if snapshot.Installed {
		if snapshot.Running {
			status = "running"
		} else {
			status = "stopped"
		}
	}

	resp := models.GuideDockerStatusResponse{
		Result: &models.GuideDockerStatusResponseResult{
			Status:    status,
			Path:      snapshot.Path,
			ErrorInfo: snapshot.ErrorInfo,
		},
	}
	return &resp, nil
}

func (service *GuideDockerRuntimeService) Switch(ctx context.Context, enable bool) (*models.SDKNormalResponse, error) {
	var err error
	if enable {
		err = service.writer.Start(ctx)
		if err != nil {
			return nil, errors.New("docker启动失败")
		}
	} else {
		err = service.writer.Stop(ctx)
		if err != nil {
			return nil, errors.New("docker停止失败")
		}
	}

	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}, nil
}

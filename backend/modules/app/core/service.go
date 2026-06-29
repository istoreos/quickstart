package core

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	IsInstalled(ctx context.Context, name string) (bool, error)
	IsRunning(ctx context.Context, name string) bool
	Install(ctx context.Context, name string) (string, error)
	InstalledList(ctx context.Context) ([]*models.AppInstalled, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (service *Service) Check(ctx context.Context, req models.AppCheckRequest) (*models.AppCheckResponse, error) {
	model := models.AppCheckResponseResult{Name: req.Name}
	didInstall, err := service.store.IsInstalled(ctx, req.Name)
	if err != nil {
		return nil, errors.New("检测" + req.Name + "失败")
	}
	if didInstall {
		model.Status = "installed"
		if req.CheckRunning {
			if service.store.IsRunning(ctx, req.Name) {
				model.Status = "running"
			} else {
				model.Status = "stopped"
			}
		}
	} else {
		model.Status = "uninstalled"
	}
	return &models.AppCheckResponse{Result: &model}, nil
}

func (service *Service) Install(ctx context.Context, req models.AppInstallRequest) (*models.SDKNormalResponse, error) {
	if len(req.Name) == 0 {
		return nil, errors.New("missing param")
	}
	ret, err := service.store.Install(ctx, req.Name)
	if err != nil {
		resp := models.SDKNormalResponse{
			Error: models.ResponseError(err.Error()),
			Scope: models.ResponseScope("1003"),
		}
		return &resp, nil
	}
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success, Detail: ret}, nil
}

func (service *Service) InstalledList(ctx context.Context) (models.AppInstalledListResponse, error) {
	modelApps, err := service.store.InstalledList(ctx)
	if err != nil {
		return nil, err
	}
	return models.AppInstalledListResponse(modelApps), nil
}

package power

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	Run(ctx context.Context, commands []string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Reboot(ctx context.Context) (*models.SDKNormalResponse, error) {
	if err := svc.store.Run(ctx, []string{"echo 'trigger reboot'", "reboot"}); err != nil {
		return nil, errors.New("重启失败" + err.Error())
	}
	return successResponse(), nil
}

func (svc *Service) PowerOff(ctx context.Context) (*models.SDKNormalResponse, error) {
	if err := svc.store.Run(ctx, []string{"echo 'trigger poweroff'", "poweroff"}); err != nil {
		return nil, errors.New("关机失败" + err.Error())
	}
	return successResponse(), nil
}

func successResponse() *models.SDKNormalResponse {
	success := models.ResponseSuccess(int64(0))
	return &models.SDKNormalResponse{Success: &success}
}

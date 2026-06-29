package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	systempower "github.com/linkease/quick-start/istore-backend/modules/system/power"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type systemPowerFacade interface {
	Reboot(ctx context.Context) (*models.SDKNormalResponse, error)
	PowerOff(ctx context.Context) (*models.SDKNormalResponse, error)
}

var newSystemPowerService = func() systemPowerFacade {
	return systempower.NewService(defaultSystemPowerStore{})
}

type defaultSystemPowerStore struct{}

func (store defaultSystemPowerStore) Run(ctx context.Context, commands []string) error {
	_, _, err := utils.BatchOutErr(ctx, commands, 0)
	return err
}

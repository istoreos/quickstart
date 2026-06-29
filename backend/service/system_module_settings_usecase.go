package service

import (
	"context"

	platformuci "github.com/linkease/quick-start/istore-backend/internal/platform/uci"
	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/system/modulesettings"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type systemModuleSettingsFacade interface {
	Get(ctx context.Context) (*models.SystemModuleSettingsResponseResult, error)
	Set(ctx context.Context, req models.SystemModuleSettingsRequest) (*models.SDKNormalResponse, error)
}

var newSystemModuleSettingsService = func() systemModuleSettingsFacade {
	return modulesettings.NewService(defaultSystemModuleSettingsStore{})
}

type defaultSystemModuleSettingsStore struct{}

func (store defaultSystemModuleSettingsStore) ReadDisabledDisplayModules(ctx context.Context) ([]string, error) {
	return platformuci.ListOption("quickstart", "modules", "module"), nil
}

func (store defaultSystemModuleSettingsStore) HasDisabledDisplaySection(ctx context.Context) bool {
	return haveUciSection("quickstart", "disabledisplay", "modules")
}

func (store defaultSystemModuleSettingsStore) ApplyCommands(ctx context.Context, commands []string) error {
	return utils.UCIBatchRun(ctx, commands, "", 0)
}

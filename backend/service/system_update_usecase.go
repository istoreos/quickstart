package service

import (
	"context"
	"os/exec"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/system/updatecheck"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type systemUpdateFacade interface {
	Check(ctx context.Context) (*models.SystemCheckUpdateResponseResult, error)
	SetAutoCheck(ctx context.Context, req models.SystemAutoCheckUpdateRequest) (*models.SDKNormalResponse, error)
}

var newSystemUpdateService = func() systemUpdateFacade {
	return updatecheck.NewService(defaultSystemUpdateStore{})
}

type defaultSystemUpdateStore struct{}

func (store defaultSystemUpdateStore) RunUpdateCheck(ctx context.Context) (string, int, error) {
	ret, err := utils.BatchOutputCmd(ctx, "ota check", 0)
	if err == nil {
		return string(ret), 0, nil
	}

	exitCode := -1
	if exitError, ok := err.(*exec.ExitError); ok {
		exitCode = exitError.ExitCode()
	}
	return string(ret), exitCode, err
}

func (store defaultSystemUpdateStore) ApplyAutoCheckCommands(ctx context.Context, commands []string) error {
	return utils.BatchRun(ctx, commands, 0)
}

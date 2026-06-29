package modulesettings

import (
	"context"
	"errors"
	"fmt"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	ReadDisabledDisplayModules(ctx context.Context) ([]string, error)
	HasDisabledDisplaySection(ctx context.Context) bool
	ApplyCommands(ctx context.Context, commands []string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Get(ctx context.Context) (*models.SystemModuleSettingsResponseResult, error) {
	modules, err := svc.store.ReadDisabledDisplayModules(ctx)
	if err != nil {
		return nil, err
	}
	result := &models.SystemModuleSettingsResponseResult{
		DiableDisplay: make([]string, 0, len(modules)),
	}
	result.DiableDisplay = append(result.DiableDisplay, modules...)
	return result, nil
}

func (svc *Service) Set(ctx context.Context, req models.SystemModuleSettingsRequest) (*models.SDKNormalResponse, error) {
	if req.DiableDisplay == nil {
		return nil, errors.New("invalid params")
	}

	commands := buildSetCommands(req.DiableDisplay, svc.store.HasDisabledDisplaySection(ctx))
	if len(commands) > 0 {
		if err := svc.store.ApplyCommands(ctx, commands); err != nil {
			return nil, err
		}
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func buildSetCommands(disabledModules []string, hasSection bool) []string {
	var commands []string
	if hasSection {
		commands = append(commands, "delete quickstart.modules")
	}

	if len(disabledModules) > 0 {
		commands = append(commands, "set quickstart.modules=disabledisplay")
		for _, module := range disabledModules {
			commands = append(commands, fmt.Sprintf("add_list quickstart.modules.module='%s'", module))
		}
	}

	if len(commands) > 0 {
		commands = append(commands, "commit quickstart")
	}
	return commands
}

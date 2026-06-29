package updatecheck

import (
	"context"
	"errors"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	RunUpdateCheck(ctx context.Context) (string, int, error)
	ApplyAutoCheckCommands(ctx context.Context, commands []string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Check(ctx context.Context) (*models.SystemCheckUpdateResponseResult, error) {
	output, exitCode, err := svc.store.RunUpdateCheck(ctx)
	if err != nil {
		result := &models.SystemCheckUpdateResponseResult{
			NeedUpdate: false,
			Msg:        output,
		}
		if exitCode == 1 {
			result.Msg = "Already the latest firmware"
			return result, nil
		}
		return nil, err
	}

	return &models.SystemCheckUpdateResponseResult{
		NeedUpdate: true,
		Msg:        output,
	}, nil
}

func (svc *Service) SetAutoCheck(ctx context.Context, req models.SystemAutoCheckUpdateRequest) (*models.SDKNormalResponse, error) {
	if err := svc.store.ApplyAutoCheckCommands(ctx, buildAutoCheckCommands(req.Enable)); err != nil {
		return nil, errors.New("设置失败")
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func buildAutoCheckCommands(enable bool) []string {
	if enable {
		return []string{
			"uci delete quickstart.main.disable_update_check",
			"uci commit quickstart",
		}
	}
	return []string{
		"uci set quickstart.main.disable_update_check=1",
		"uci commit quickstart",
	}
}

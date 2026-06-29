package password

import (
	"context"
	"errors"
	"fmt"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	CallSetPassword(ctx context.Context, command string) (bool, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) SetRootPassword(ctx context.Context, req models.NasSystemSetPasswordRequest) (*models.SDKNormalResponse, error) {
	changed, err := svc.store.CallSetPassword(ctx, BuildSetRootPasswordCommand(req.Password))
	if err != nil {
		return nil, errors.New("设置密码错误")
	}
	if !changed {
		resp := models.SDKNormalResponse{
			Error: models.ResponseError("-100"),
			Scope: models.ResponseScope("system.setpassd"),
		}
		return &resp, nil
	}

	success := models.ResponseSuccess(int64(0))
	resp := models.SDKNormalResponse{Success: &success}
	return &resp, nil
}

func BuildSetRootPasswordCommand(password string) string {
	return fmt.Sprintf("luci setPassword {\"username\":\"root\",\"password\":\"%s\"}", password)
}

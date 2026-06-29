package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	systempassword "github.com/linkease/quick-start/istore-backend/modules/system/password"
)

type systemPasswordFacade interface {
	SetRootPassword(ctx context.Context, req models.NasSystemSetPasswordRequest) (*models.SDKNormalResponse, error)
}

var newSystemPasswordService = func() systemPasswordFacade {
	return systempassword.NewService(defaultSystemPasswordStore{})
}

type defaultSystemPasswordStore struct{}

func (store defaultSystemPasswordStore) CallSetPassword(ctx context.Context, command string) (bool, error) {
	simpleJ, err := UbusCall(ctx, command)
	if err != nil {
		return false, err
	}
	changed, err := simpleJ.Get("result").Bool()
	if err != nil {
		return false, nil
	}
	return changed, nil
}

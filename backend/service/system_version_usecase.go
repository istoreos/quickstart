package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	systemversion "github.com/linkease/quick-start/istore-backend/modules/system/version"
)

type systemVersionFacade interface {
	Get(ctx context.Context) (*models.SystemVersionResponseResult, error)
}

var newSystemVersionService = func() systemVersionFacade {
	return systemversion.NewService(defaultSystemVersionStore{}, VERSION)
}

type defaultSystemVersionStore struct{}

func (store defaultSystemVersionStore) ReadBoard(ctx context.Context) (systemversion.Board, error) {
	var board systemversion.Board
	err := UbusCallWithObject(ctx, "system board", &board)
	return board, err
}

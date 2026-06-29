package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/modules/network/homebox"
)

type homeBoxEnableFacade interface {
	Enable(ctx context.Context) (*models.NetworkHomeBoxEnableResponse, error)
}

var newHomeBoxEnableService = func() homeBoxEnableFacade {
	return homebox.NewDefaultHomeBoxEnableService()
}

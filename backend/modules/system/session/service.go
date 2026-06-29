package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Store interface {
	ReadToken(ctx context.Context, command string) (string, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Get(ctx context.Context, sessionID string) (*models.SystemCsrfTokenResponseResult, error) {
	if sessionID == "" {
		return nil, errors.New("need auth")
	}

	token, err := svc.store.ReadToken(ctx, BuildGetSessionCommand(sessionID))
	if err != nil {
		return nil, errors.New("获取session失败")
	}
	if token == "" {
		return nil, errors.New("fail to get token")
	}

	return &models.SystemCsrfTokenResponseResult{Token: token}, nil
}

func BuildGetSessionCommand(sessionID string) string {
	return fmt.Sprintf("session get {\"ubus_rpc_session\":\"%s\"}", sessionID)
}

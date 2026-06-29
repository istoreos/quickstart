package service

import (
	"context"
	"net/http"

	"github.com/linkease/quick-start/istore-backend/models"
	systemsession "github.com/linkease/quick-start/istore-backend/modules/system/session"
)

type systemSessionFacade interface {
	Get(ctx context.Context, sessionID string) (*models.SystemCsrfTokenResponseResult, error)
}

var newSystemSessionService = func() systemSessionFacade {
	return systemsession.NewService(defaultSystemSessionStore{})
}

type defaultSystemSessionStore struct{}

func (store defaultSystemSessionStore) ReadToken(ctx context.Context, command string) (string, error) {
	simpleJ, err := UbusCall(ctx, command)
	if err != nil {
		return "", err
	}
	token, err := simpleJ.Get("values").Get("token").String()
	if err != nil {
		return "", nil
	}
	return token, nil
}

func systemSessionIDFromRequest(r *http.Request) string {
	sessionID, _ := r.Cookie("sysauth")
	sessionHTTP, _ := r.Cookie("sysauth_http")
	sessionHTTPS, _ := r.Cookie("sysauth_https")
	if sessionID == nil {
		sessionID = sessionHTTP
	}
	if sessionID == nil {
		sessionID = sessionHTTPS
	}
	if sessionID == nil {
		return ""
	}
	return sessionID.Value
}

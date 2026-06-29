package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeSystemSessionFacade struct {
	result    *models.SystemCsrfTokenResponseResult
	err       error
	sessionID string
	called    bool
}

func (svc *fakeSystemSessionFacade) Get(ctx context.Context, sessionID string) (*models.SystemCsrfTokenResponseResult, error) {
	svc.called = true
	svc.sessionID = sessionID
	return svc.result, svc.err
}

func TestSystemGetSessionUsesSysauthCookieFirst(t *testing.T) {
	original := newSystemSessionService
	defer func() { newSystemSessionService = original }()

	facade := &fakeSystemSessionFacade{
		result: &models.SystemCsrfTokenResponseResult{Token: "token-1"},
	}
	newSystemSessionService = func() systemSessionFacade {
		return facade
	}

	req := httptest.NewRequest("GET", "/session", nil)
	req.AddCookie(&http.Cookie{Name: "sysauth_https", Value: "https-session"})
	req.AddCookie(&http.Cookie{Name: "sysauth_http", Value: "http-session"})
	req.AddCookie(&http.Cookie{Name: "sysauth", Value: "main-session"})

	resp, err := SystemGetSession(context.Background(), req)
	if err != nil {
		t.Fatalf("SystemGetSession returned error: %v", err)
	}
	if resp.Result == nil || resp.Result.Token != "token-1" {
		t.Fatalf("Result = %#v", resp.Result)
	}
	if facade.sessionID != "main-session" {
		t.Fatalf("sessionID = %q, want main-session", facade.sessionID)
	}
}

func TestSystemGetSessionFallsBackToHttpThenHttpsCookie(t *testing.T) {
	original := newSystemSessionService
	defer func() { newSystemSessionService = original }()

	facade := &fakeSystemSessionFacade{
		result: &models.SystemCsrfTokenResponseResult{Token: "token-1"},
	}
	newSystemSessionService = func() systemSessionFacade {
		return facade
	}

	req := httptest.NewRequest("GET", "/session", nil)
	req.AddCookie(&http.Cookie{Name: "sysauth_https", Value: "https-session"})
	req.AddCookie(&http.Cookie{Name: "sysauth_http", Value: "http-session"})

	if _, err := SystemGetSession(context.Background(), req); err != nil {
		t.Fatalf("SystemGetSession returned error: %v", err)
	}
	if facade.sessionID != "http-session" {
		t.Fatalf("sessionID = %q, want http-session", facade.sessionID)
	}

	req = httptest.NewRequest("GET", "/session", nil)
	req.AddCookie(&http.Cookie{Name: "sysauth_https", Value: "https-session"})
	if _, err := SystemGetSession(context.Background(), req); err != nil {
		t.Fatalf("SystemGetSession returned error: %v", err)
	}
	if facade.sessionID != "https-session" {
		t.Fatalf("sessionID = %q, want https-session", facade.sessionID)
	}
}

func TestSystemGetSessionDelegatesMissingCookieToFacade(t *testing.T) {
	original := newSystemSessionService
	defer func() { newSystemSessionService = original }()

	expectedErr := errors.New("need auth")
	facade := &fakeSystemSessionFacade{err: expectedErr}
	newSystemSessionService = func() systemSessionFacade {
		return facade
	}

	req := httptest.NewRequest("GET", "/session", nil)
	if _, err := SystemGetSession(context.Background(), req); !errors.Is(err, expectedErr) {
		t.Fatalf("SystemGetSession error = %v, want expectedErr", err)
	}
	if !facade.called {
		t.Fatal("Get was not called")
	}
	if facade.sessionID != "" {
		t.Fatalf("sessionID = %q, want empty", facade.sessionID)
	}
}

package session

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	command string
	token   string
	err     error
	called  bool
}

func (store *fakeStore) ReadToken(ctx context.Context, command string) (string, error) {
	store.called = true
	store.command = command
	return store.token, store.err
}

func TestGetReturnsNeedAuthWithoutSessionID(t *testing.T) {
	t.Parallel()

	store := &fakeStore{}
	svc := NewService(store)

	if _, err := svc.Get(context.Background(), ""); err == nil || err.Error() != "need auth" {
		t.Fatalf("Get error = %v, want need auth", err)
	}
	if store.called {
		t.Fatal("store was called for empty session id")
	}
}

func TestGetBuildsCommandAndReturnsToken(t *testing.T) {
	t.Parallel()

	store := &fakeStore{token: "token-1"}
	svc := NewService(store)

	result, err := svc.Get(context.Background(), "sess-1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if store.command != `session get {"ubus_rpc_session":"sess-1"}` {
		t.Fatalf("command = %q", store.command)
	}
	if result.Token != "token-1" {
		t.Fatalf("Token = %q", result.Token)
	}
}

func TestGetMapsStoreError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{err: errors.New("ubus failed")})

	if _, err := svc.Get(context.Background(), "sess-1"); err == nil || err.Error() != "获取session失败" {
		t.Fatalf("Get error = %v, want 获取session失败", err)
	}
}

func TestGetRejectsEmptyToken(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{token: ""})

	if _, err := svc.Get(context.Background(), "sess-1"); err == nil || err.Error() != "fail to get token" {
		t.Fatalf("Get error = %v, want fail to get token", err)
	}
}

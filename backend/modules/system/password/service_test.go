package password

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	command string
	changed bool
	err     error
}

func (store *fakeStore) CallSetPassword(ctx context.Context, command string) (bool, error) {
	store.command = command
	return store.changed, store.err
}

func TestSetRootPasswordBuildsCommandAndReturnsSuccess(t *testing.T) {
	t.Parallel()

	store := &fakeStore{changed: true}
	svc := NewService(store)

	resp, err := svc.SetRootPassword(context.Background(), models.NasSystemSetPasswordRequest{
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("SetRootPassword returned error: %v", err)
	}
	if store.command != `luci setPassword {"username":"root","password":"secret"}` {
		t.Fatalf("command = %q", store.command)
	}
	if resp.Success == nil || *resp.Success != models.ResponseSuccess(0) {
		t.Fatalf("Success = %#v, want 0", resp.Success)
	}
}

func TestSetRootPasswordMapsStoreError(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{err: errors.New("ubus failed")})

	if _, err := svc.SetRootPassword(context.Background(), models.NasSystemSetPasswordRequest{
		Password: "secret",
	}); err == nil || err.Error() != "设置密码错误" {
		t.Fatalf("SetRootPassword error = %v, want 设置密码错误", err)
	}
}

func TestSetRootPasswordReturnsBusinessErrorWhenPasswordUnchanged(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{changed: false})

	resp, err := svc.SetRootPassword(context.Background(), models.NasSystemSetPasswordRequest{
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("SetRootPassword returned error: %v", err)
	}
	if resp.Error != models.ResponseError("-100") {
		t.Fatalf("Error = %q, want -100", resp.Error)
	}
	if resp.Scope != models.ResponseScope("system.setpassd") {
		t.Fatalf("Scope = %q, want system.setpassd", resp.Scope)
	}
}

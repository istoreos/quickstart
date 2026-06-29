package user

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	users      []*models.ShareUserInfo
	readErr    error
	createCall *createCall
	createErr  error
	updateCall *updateCall
	updateErr  error
	deleteCall *deleteCall
	deleteErr  error
}

type createCall struct {
	index int
	input CreateInput
}

type updateCall struct {
	index    int
	password string
}

type deleteCall struct {
	index int
}

func (store *fakeStore) ReadUsers(ctx context.Context) ([]*models.ShareUserInfo, error) {
	if store.readErr != nil {
		return nil, store.readErr
	}
	return store.users, nil
}

func (store *fakeStore) CreateUser(ctx context.Context, index int, input CreateInput) error {
	store.createCall = &createCall{index: index, input: input}
	return store.createErr
}

func (store *fakeStore) UpdateUser(ctx context.Context, index int, password string) error {
	store.updateCall = &updateCall{index: index, password: password}
	return store.updateErr
}

func (store *fakeStore) DeleteUser(ctx context.Context, index int) error {
	store.deleteCall = &deleteCall{index: index}
	return store.deleteErr
}

func TestServiceListFiltersEmptyUsernames(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{users: []*models.ShareUserInfo{
		{UserName: "alice", Password: "a"},
		{Password: "empty"},
		{UserName: "bob", Password: "b"},
	}})

	users, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(users) != 2 || users[0].UserName != "alice" || users[1].UserName != "bob" {
		t.Fatalf("unexpected users: %#v", users)
	}
}

func TestServiceCreateValidatesInputAndRejectsDuplicates(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{})
	cases := []struct {
		name  string
		input CreateInput
		err   string
	}{
		{name: "missing", input: CreateInput{}, err: "param missing"},
		{name: "long password", input: CreateInput{UserName: "alice", Password: "1234567890123456"}, err: "the password must be less than 15 characters"},
		{name: "long username", input: CreateInput{UserName: "abcdefghijklmnop", Password: "pw"}, err: "the username must be less than 15 characters"},
		{name: "reserved users", input: CreateInput{UserName: "users", Password: "pw"}, err: "invalid username"},
		{name: "reserved everyone", input: CreateInput{UserName: "everyone", Password: "pw"}, err: "invalid username"},
		{name: "reserved root", input: CreateInput{UserName: "root", Password: "pw"}, err: "invalid username"},
		{name: "uppercase first", input: CreateInput{UserName: "Alice", Password: "pw"}, err: "invalid username, should begin with lowercase letter"},
		{name: "invalid chars", input: CreateInput{UserName: "alice$", Password: "pw"}, err: "invalid username"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := svc.Create(context.Background(), tc.input); err == nil || err.Error() != tc.err {
				t.Fatalf("expected %q, got %v", tc.err, err)
			}
		})
	}

	svc = NewService(&fakeStore{users: []*models.ShareUserInfo{{UserName: "alice"}}})
	if err := svc.Create(context.Background(), CreateInput{UserName: "alice", Password: "pw"}); err == nil || err.Error() != "user already exist" {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestServiceCreateUsesNextUserIndexAndWritesStore(t *testing.T) {
	t.Parallel()

	store := &fakeStore{users: []*models.ShareUserInfo{
		{UserName: "alice"},
		{UserName: "bob"},
	}}
	svc := NewService(store)

	err := svc.Create(context.Background(), CreateInput{UserName: "carol", Password: "pw"})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if store.createCall == nil || store.createCall.index != 2 || store.createCall.input.UserName != "carol" || store.createCall.input.Password != "pw" {
		t.Fatalf("unexpected create call: %#v", store.createCall)
	}
}

func TestServiceUpdatePreservesLegacyValidationAndUpdatesMatchingUser(t *testing.T) {
	t.Parallel()

	store := &fakeStore{users: []*models.ShareUserInfo{
		{UserName: "alice"},
		{UserName: "bob"},
	}}
	svc := NewService(store)

	err := svc.Update(context.Background(), UpdateInput{UserName: "bob", Password: "newpw"})
	if err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if store.updateCall == nil || store.updateCall.index != 1 || store.updateCall.password != "newpw" {
		t.Fatalf("unexpected update call: %#v", store.updateCall)
	}

	err = svc.Update(context.Background(), UpdateInput{UserName: "alice$", Password: "pw"})
	if err == nil || err.Error() != "user not found" {
		t.Fatalf("expected legacy update to allow invalid chars before lookup, got %v", err)
	}
}

func TestServiceUpdateValidatesAndReportsMissingUser(t *testing.T) {
	t.Parallel()

	svc := NewService(&fakeStore{})
	cases := []struct {
		name  string
		input UpdateInput
		err   string
	}{
		{name: "missing", input: UpdateInput{}, err: "param missing"},
		{name: "long password", input: UpdateInput{UserName: "alice", Password: "1234567890123456"}, err: "the password must be less than 15 characters"},
		{name: "long username", input: UpdateInput{UserName: "abcdefghijklmnop", Password: "pw"}, err: "the username must be less than 15 characters"},
		{name: "reserved users", input: UpdateInput{UserName: "users", Password: "pw"}, err: "invalid username"},
		{name: "reserved everyone", input: UpdateInput{UserName: "everyone", Password: "pw"}, err: "invalid username"},
		{name: "uppercase first", input: UpdateInput{UserName: "Alice", Password: "pw"}, err: "invalid username, should begin with lowercase letter"},
		{name: "not found", input: UpdateInput{UserName: "alice", Password: "pw"}, err: "user not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := svc.Update(context.Background(), tc.input); err == nil || err.Error() != tc.err {
				t.Fatalf("expected %q, got %v", tc.err, err)
			}
		})
	}
}

func TestServiceDeleteValidatesAndDeletesMatchingUser(t *testing.T) {
	t.Parallel()

	store := &fakeStore{users: []*models.ShareUserInfo{
		{UserName: "alice"},
		{UserName: "bob"},
	}}
	svc := NewService(store)

	if err := svc.Delete(context.Background(), DeleteInput{UserName: "bob"}); err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if store.deleteCall == nil || store.deleteCall.index != 1 {
		t.Fatalf("unexpected delete call: %#v", store.deleteCall)
	}

	cases := []struct {
		name  string
		input DeleteInput
		err   string
	}{
		{name: "missing", input: DeleteInput{}, err: "param missing"},
		{name: "reserved users", input: DeleteInput{UserName: "users"}, err: "invalid username"},
		{name: "reserved everyone", input: DeleteInput{UserName: "everyone"}, err: "invalid username"},
		{name: "not found", input: DeleteInput{UserName: "carol"}, err: "user not found"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := svc.Delete(context.Background(), tc.input); err == nil || err.Error() != tc.err {
				t.Fatalf("expected %q, got %v", tc.err, err)
			}
		})
	}
}

func TestServicePropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	readErr := errors.New("read failed")
	svc := NewService(&fakeStore{readErr: readErr})
	if _, err := svc.List(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("expected list read error, got %v", err)
	}
	if err := svc.Create(context.Background(), CreateInput{UserName: "alice", Password: "pw"}); !errors.Is(err, readErr) {
		t.Fatalf("expected create read error, got %v", err)
	}

	writeErr := errors.New("write failed")
	store := &fakeStore{createErr: writeErr, updateErr: writeErr, deleteErr: writeErr, users: []*models.ShareUserInfo{{UserName: "alice"}}}
	svc = NewService(store)
	if err := svc.Create(context.Background(), CreateInput{UserName: "bob", Password: "pw"}); !errors.Is(err, writeErr) {
		t.Fatalf("expected create write error, got %v", err)
	}
	if err := svc.Update(context.Background(), UpdateInput{UserName: "alice", Password: "pw"}); !errors.Is(err, writeErr) {
		t.Fatalf("expected update write error, got %v", err)
	}
	if err := svc.Delete(context.Background(), DeleteInput{UserName: "alice"}); !errors.Is(err, writeErr) {
		t.Fatalf("expected delete write error, got %v", err)
	}
}

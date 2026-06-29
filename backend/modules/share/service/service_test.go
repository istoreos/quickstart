package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	shares []*ShareRecord
	users  []*models.ShareUserInfo

	readErr   error
	createErr error
	updateErr error
	deleteErr error

	createdIndex int
	createdInput CreateInput
	updatedIndex int
	updatedInput UpdateInput
	deletedIndex int
}

func (store *fakeStore) ReadConfig(ctx context.Context) ([]*ShareRecord, []*models.ShareUserInfo, error) {
	if store.readErr != nil {
		return nil, nil, store.readErr
	}
	return store.shares, store.users, nil
}

func (store *fakeStore) CreateShare(ctx context.Context, index int, input CreateInput) error {
	store.createdIndex = index
	store.createdInput = input
	return store.createErr
}

func (store *fakeStore) UpdateShare(ctx context.Context, index int, input UpdateInput) error {
	store.updatedIndex = index
	store.updatedInput = input
	return store.updateErr
}

func (store *fakeStore) DeleteShare(ctx context.Context, index int) error {
	store.deletedIndex = index
	return store.deleteErr
}

func TestListBuildsShareServicesWithUsersPermissionsAndProtocols(t *testing.T) {
	svc := NewService(&fakeStore{
		users: []*models.ShareUserInfo{
			{UserName: "alice"},
			{UserName: "bob"},
			{},
		},
		shares: []*ShareRecord{
			{
				Name:  "media",
				Path:  "/mnt/media",
				RO:    []string{"bob", "missing"},
				RW:    []string{"alice", "bob"},
				Proto: []string{"samba", "webdav", "ftp"},
			},
			{
				Name:  "docs",
				Path:  "/mnt/docs",
				RO:    []string{"alice"},
				Proto: []string{"webdav"},
			},
		},
	})

	services, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	want := []*models.ShareServiceInfo{
		{
			Name:   "media",
			Path:   "/mnt/media",
			Samba:  true,
			Webdav: true,
			Users: []*models.ShareServiceUserPermission{
				{UserName: "alice", Rw: true},
				{UserName: "bob", Rw: true},
				{UserName: ""},
			},
		},
		{
			Name:   "docs",
			Path:   "/mnt/docs",
			Webdav: true,
			Users: []*models.ShareServiceUserPermission{
				{UserName: "alice", Ro: true},
				{UserName: "bob"},
				{UserName: ""},
			},
		},
	}
	if !reflect.DeepEqual(services, want) {
		t.Fatalf("List mismatch\nwant: %#v\n got: %#v", want, services)
	}
}

func TestCreateValidatesInputAndDuplicateName(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateInput
		wantErr string
	}{
		{name: "missing name", input: CreateInput{Path: "/mnt/data"}, wantErr: "param missing"},
		{name: "missing path", input: CreateInput{Name: "media"}, wantErr: "param missing"},
		{name: "long name", input: CreateInput{Name: strings.Repeat("a", 16), Path: "/mnt/data"}, wantErr: "name must be less than 15 characters"},
		{name: "invalid regex", input: CreateInput{Name: "bad.name", Path: "/mnt/data"}, wantErr: "invalid name "},
		{name: "uppercase first", input: CreateInput{Name: "Media", Path: "/mnt/data"}, wantErr: "invalid name "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(&fakeStore{})
			err := svc.Create(context.Background(), tt.input)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Create error = %v, want %q", err, tt.wantErr)
			}
		})
	}

	svc := NewService(&fakeStore{shares: []*ShareRecord{{Name: "media"}}})
	err := svc.Create(context.Background(), CreateInput{Name: "media", Path: "/mnt/media"})
	if err == nil || err.Error() != "already exist" {
		t.Fatalf("Create duplicate error = %v, want already exist", err)
	}
}

func TestCreateWritesNextShareIndex(t *testing.T) {
	store := &fakeStore{
		shares: []*ShareRecord{{Name: "media"}, {Name: "docs"}},
	}
	svc := NewService(store)
	input := CreateInput{
		Name:   "backup",
		Path:   "/mnt/backup",
		Samba:  true,
		Webdav: true,
		Users: []*models.ShareServiceUserPermission{
			{UserName: "alice", Rw: true},
			{UserName: "bob", Ro: true},
		},
	}

	if err := svc.Create(context.Background(), input); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if store.createdIndex != 2 {
		t.Fatalf("createdIndex = %d, want 2", store.createdIndex)
	}
	if !reflect.DeepEqual(store.createdInput, input) {
		t.Fatalf("created input mismatch\nwant: %#v\n got: %#v", input, store.createdInput)
	}
}

func TestUpdateFindsExistingShareAndWritesSameIndex(t *testing.T) {
	store := &fakeStore{
		shares: []*ShareRecord{{Name: "media"}, {Name: "docs"}},
	}
	svc := NewService(store)
	input := UpdateInput{Name: "docs", Path: "/mnt/docs", Webdav: true}

	if err := svc.Update(context.Background(), input); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if store.updatedIndex != 1 {
		t.Fatalf("updatedIndex = %d, want 1", store.updatedIndex)
	}
	if !reflect.DeepEqual(store.updatedInput, input) {
		t.Fatalf("updated input mismatch\nwant: %#v\n got: %#v", input, store.updatedInput)
	}
}

func TestUpdateKeepsLegacyValidationAndNotFound(t *testing.T) {
	svc := NewService(&fakeStore{})

	tests := []struct {
		name    string
		input   UpdateInput
		wantErr string
	}{
		{name: "missing name", input: UpdateInput{Path: "/mnt/data"}, wantErr: "param missing"},
		{name: "missing path", input: UpdateInput{Name: "media"}, wantErr: "param missing"},
		{name: "long name", input: UpdateInput{Name: strings.Repeat("a", 16), Path: "/mnt/data"}, wantErr: "name must be less than 15 characters"},
		{name: "invalid regex", input: UpdateInput{Name: "bad.name", Path: "/mnt/data"}, wantErr: "invalid name "},
		{name: "uppercase first", input: UpdateInput{Name: "Media", Path: "/mnt/data"}, wantErr: "invalid name "},
		{name: "not found", input: UpdateInput{Name: "media", Path: "/mnt/data"}, wantErr: "share not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Update(context.Background(), tt.input)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("Update error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteValidatesFindsShareAndWritesIndex(t *testing.T) {
	svc := NewService(&fakeStore{})
	err := svc.Delete(context.Background(), DeleteInput{})
	if err == nil || err.Error() != "param missing" {
		t.Fatalf("Delete missing error = %v, want param missing", err)
	}

	err = svc.Delete(context.Background(), DeleteInput{Name: "media"})
	if err == nil || err.Error() != "share not found" {
		t.Fatalf("Delete not found error = %v, want share not found", err)
	}

	store := &fakeStore{shares: []*ShareRecord{{Name: "media"}, {Name: "docs"}}}
	svc = NewService(store)
	if err := svc.Delete(context.Background(), DeleteInput{Name: "docs"}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if store.deletedIndex != 1 {
		t.Fatalf("deletedIndex = %d, want 1", store.deletedIndex)
	}
}

func TestStoreErrorsArePropagated(t *testing.T) {
	readErr := errors.New("read failed")
	writeErr := errors.New("write failed")

	svc := NewService(&fakeStore{readErr: readErr})
	if _, err := svc.List(context.Background()); !errors.Is(err, readErr) {
		t.Fatalf("List error = %v, want readErr", err)
	}
	if err := svc.Create(context.Background(), CreateInput{Name: "media", Path: "/mnt/media"}); !errors.Is(err, readErr) {
		t.Fatalf("Create read error = %v, want readErr", err)
	}

	svc = NewService(&fakeStore{shares: []*ShareRecord{{Name: "media"}}, createErr: writeErr, updateErr: writeErr, deleteErr: writeErr})
	if err := svc.Create(context.Background(), CreateInput{Name: "docs", Path: "/mnt/docs"}); !errors.Is(err, writeErr) {
		t.Fatalf("Create write error = %v, want writeErr", err)
	}
	if err := svc.Update(context.Background(), UpdateInput{Name: "media", Path: "/mnt/media"}); !errors.Is(err, writeErr) {
		t.Fatalf("Update write error = %v, want writeErr", err)
	}
	if err := svc.Delete(context.Background(), DeleteInput{Name: "media"}); !errors.Is(err, writeErr) {
		t.Fatalf("Delete write error = %v, want writeErr", err)
	}
}

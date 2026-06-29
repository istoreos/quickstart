package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/linkease/quick-start/istore-backend/models"
)

type ShareRecord struct {
	Name  string
	Path  string
	RO    []string
	RW    []string
	Proto []string
}

type CreateInput struct {
	Name   string
	Path   string
	Samba  bool
	Webdav bool
	Users  []*models.ShareServiceUserPermission
}

type UpdateInput = CreateInput

type DeleteInput struct {
	Name string
}

type Store interface {
	ReadConfig(ctx context.Context) ([]*ShareRecord, []*models.ShareUserInfo, error)
	CreateShare(ctx context.Context, index int, input CreateInput) error
	UpdateShare(ctx context.Context, index int, input UpdateInput) error
	DeleteShare(ctx context.Context, index int) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) List(ctx context.Context) ([]*models.ShareServiceInfo, error) {
	shares, users, err := svc.store.ReadConfig(ctx)
	if err != nil {
		return nil, err
	}

	services := make([]*models.ShareServiceInfo, 0, len(shares))
	for _, share := range shares {
		item := &models.ShareServiceInfo{
			Name:  share.Name,
			Path:  share.Path,
			Users: buildUserPermissions(users, share),
		}
		for _, proto := range share.Proto {
			switch proto {
			case "samba":
				item.Samba = true
			case "webdav":
				item.Webdav = true
			}
		}
		services = append(services, item)
	}
	return services, nil
}

func (svc *Service) Create(ctx context.Context, input CreateInput) error {
	if err := validateShareInput(input.Name, input.Path); err != nil {
		return err
	}

	shares, _, err := svc.store.ReadConfig(ctx)
	if err != nil {
		return err
	}
	for _, share := range shares {
		if share.Name == input.Name {
			return errors.New("already exist")
		}
	}
	return svc.store.CreateShare(ctx, len(shares), input)
}

func (svc *Service) Update(ctx context.Context, input UpdateInput) error {
	if err := validateShareInput(input.Name, input.Path); err != nil {
		return err
	}

	shares, _, err := svc.store.ReadConfig(ctx)
	if err != nil {
		return err
	}
	index := findShareIndex(shares, input.Name)
	if index < 0 {
		return errors.New("share not found")
	}
	return svc.store.UpdateShare(ctx, index, input)
}

func (svc *Service) Delete(ctx context.Context, input DeleteInput) error {
	if input.Name == "" {
		return errors.New("param missing")
	}

	shares, _, err := svc.store.ReadConfig(ctx)
	if err != nil {
		return err
	}
	index := findShareIndex(shares, input.Name)
	if index < 0 {
		return errors.New("share not found")
	}
	return svc.store.DeleteShare(ctx, index)
}

func buildUserPermissions(users []*models.ShareUserInfo, share *ShareRecord) []*models.ShareServiceUserPermission {
	perms := make([]*models.ShareServiceUserPermission, 0, len(users))
	for _, user := range users {
		perm := &models.ShareServiceUserPermission{UserName: user.UserName}
		if containsName(share.RW, user.UserName) {
			perm.Rw = true
		}
		if containsName(share.RO, user.UserName) && !perm.Rw {
			perm.Ro = true
		}
		perms = append(perms, perm)
	}
	return perms
}

func validateShareInput(name, path string) error {
	if name == "" || path == "" {
		return errors.New("param missing")
	}
	if len(name) > 15 {
		return errors.New("name must be less than 15 characters")
	}
	if !IsNameValid(name) {
		return errors.New("invalid name ")
	}
	return nil
}

func IsNameValid(name string) bool {
	match, _ := regexp.MatchString("^[a-z][a-z0-9_-]*$", name)
	return match
}

func findShareIndex(shares []*ShareRecord, name string) int {
	index := -1
	for idx, share := range shares {
		if share.Name == name {
			index = idx
		}
	}
	return index
}

func containsName(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

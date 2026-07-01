package user

import (
	"context"
	"errors"
	"regexp"
	"unicode"

	"github.com/istoreos/quickstart/backend/models"
)

type CreateInput struct {
	UserName string
	Password string
}

type UpdateInput struct {
	UserName string
	Password string
}

type DeleteInput struct {
	UserName string
}

type Store interface {
	ReadUsers(ctx context.Context) ([]*models.ShareUserInfo, error)
	CreateUser(ctx context.Context, index int, input CreateInput) error
	UpdateUser(ctx context.Context, index int, password string) error
	DeleteUser(ctx context.Context, index int) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) List(ctx context.Context) ([]*models.ShareUserInfo, error) {
	users, err := svc.store.ReadUsers(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]*models.ShareUserInfo, 0, len(users))
	for _, user := range users {
		if user != nil && len(user.UserName) > 0 {
			filtered = append(filtered, user)
		}
	}
	return filtered, nil
}

func (svc *Service) Create(ctx context.Context, input CreateInput) error {
	if err := validateCreateInput(input); err != nil {
		return err
	}

	users, err := svc.store.ReadUsers(ctx)
	if err != nil {
		return err
	}
	for _, user := range users {
		if user != nil && input.UserName == user.UserName {
			return errors.New("user already exist")
		}
	}
	return svc.store.CreateUser(ctx, len(users), input)
}

func (svc *Service) Ensure(ctx context.Context, input CreateInput) error {
	if err := validateCreateInput(input); err != nil {
		return err
	}

	users, err := svc.store.ReadUsers(ctx)
	if err != nil {
		return err
	}
	for idx, user := range users {
		if user != nil && input.UserName == user.UserName {
			return svc.store.UpdateUser(ctx, idx, input.Password)
		}
	}
	return svc.store.CreateUser(ctx, len(users), input)
}

func (svc *Service) Update(ctx context.Context, input UpdateInput) error {
	if err := validateUpdateInput(input); err != nil {
		return err
	}

	users, err := svc.store.ReadUsers(ctx)
	if err != nil {
		return err
	}
	for idx, user := range users {
		if user != nil && input.UserName == user.UserName {
			return svc.store.UpdateUser(ctx, idx, input.Password)
		}
	}
	return errors.New("user not found")
}

func (svc *Service) Delete(ctx context.Context, input DeleteInput) error {
	if len(input.UserName) == 0 {
		return errors.New("param missing")
	}
	if input.UserName == "users" || input.UserName == "everyone" {
		return errors.New("invalid username")
	}

	users, err := svc.store.ReadUsers(ctx)
	if err != nil {
		return err
	}
	for idx, user := range users {
		if user != nil && input.UserName == user.UserName {
			return svc.store.DeleteUser(ctx, idx)
		}
	}
	return errors.New("user not found")
}

func validateCreateInput(input CreateInput) error {
	if len(input.UserName) == 0 || len(input.Password) == 0 {
		return errors.New("param missing")
	}
	if len(input.Password) > 15 {
		return errors.New("the password must be less than 15 characters")
	}
	if len(input.UserName) > 15 {
		return errors.New("the username must be less than 15 characters")
	}
	if input.UserName == "users" || input.UserName == "everyone" || input.UserName == "root" {
		return errors.New("invalid username")
	}
	if !unicode.IsLower(rune(input.UserName[0])) {
		return errors.New("invalid username, should begin with lowercase letter")
	}
	if !IsUsernameValid(input.UserName) {
		return errors.New("invalid username")
	}
	return nil
}

func validateUpdateInput(input UpdateInput) error {
	if len(input.UserName) == 0 || len(input.Password) == 0 {
		return errors.New("param missing")
	}
	if len(input.Password) > 15 {
		return errors.New("the password must be less than 15 characters")
	}
	if len(input.UserName) > 15 {
		return errors.New("the username must be less than 15 characters")
	}
	if input.UserName == "users" || input.UserName == "everyone" {
		return errors.New("invalid username")
	}
	if !unicode.IsLower(rune(input.UserName[0])) {
		return errors.New("invalid username, should begin with lowercase letter")
	}
	return nil
}

func IsUsernameValid(name string) bool {
	match, _ := regexp.MatchString("^[a-z][a-z0-9_-]*$", name)
	return match
}

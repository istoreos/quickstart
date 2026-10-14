package service

import (
	"context"
	"errors"
	"strings"
)

type StaticAssignmentTagValidator interface {
	NormalizeTag(ctx context.Context, input StaticAssignmentWriteInput) (StaticAssignmentWriteInput, error)
}

type StaticAssignmentWriteStore interface {
	ApplyStaticAssignment(ctx context.Context, input StaticAssignmentWriteInput) error
}

type LanStaticAssignmentWriteService struct {
	store        StaticAssignmentWriteStore
	tagValidator StaticAssignmentTagValidator
}

func NewLanStaticAssignmentWriteService(store StaticAssignmentWriteStore, tagValidator StaticAssignmentTagValidator) *LanStaticAssignmentWriteService {
	return &LanStaticAssignmentWriteService{
		store:        store,
		tagValidator: tagValidator,
	}
}

func NewDefaultLanStaticAssignmentWriteService() *LanStaticAssignmentWriteService {
	lanStatus := NewDefaultLanStatusReader()
	dhcpStore := NewDefaultDhcpConfigStore()
	return NewLanStaticAssignmentWriteService(
		NewDefaultStaticAssignmentWriteStore(),
		NewDefaultStaticAssignmentTagValidator(lanStatus, dhcpStore),
	)
}

func (svc *LanStaticAssignmentWriteService) ApplyStaticAssignment(ctx context.Context, input StaticAssignmentWriteInput) error {
	if strings.TrimSpace(input.AssignedMAC) == "" {
		return errors.New("mac address is required")
	}
	originalAction := input.Action

	switch input.Action {
	case "add", "modify", "delete":
	default:
		return errors.New("invalid action")
	}
	normalizedInput, err := normalizeStaticAssignmentInput(input)
	if err != nil {
		return err
	}
	input = normalizedInput

	if input.Action == "modify" {
		input.Action = "add"
	}
	if originalAction == "add" && input.IsNoOpAdd() {
		return nil
	}

	normalized, err := svc.tagValidator.NormalizeTag(ctx, input)
	if err != nil {
		return err
	}
	normalized, err = normalizeStaticAssignmentInput(normalized)
	if err != nil {
		return err
	}
	return svc.store.ApplyStaticAssignment(ctx, normalized)
}

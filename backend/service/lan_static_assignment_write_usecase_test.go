package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeStaticAssignmentWriteStore struct {
	applyInput StaticAssignmentWriteInput
	applyErr   error
}

func (store *fakeStaticAssignmentWriteStore) ApplyStaticAssignment(ctx context.Context, input StaticAssignmentWriteInput) error {
	store.applyInput = input
	return store.applyErr
}

type fakeStaticAssignmentTagValidator struct {
	normalizedTagName string
	normalizedTitle   string
	materializeAuto   bool
	validateErr       error
}

func (v *fakeStaticAssignmentTagValidator) NormalizeTag(ctx context.Context, input StaticAssignmentWriteInput) (StaticAssignmentWriteInput, error) {
	if v.validateErr != nil {
		return StaticAssignmentWriteInput{}, v.validateErr
	}
	input.TagName = v.normalizedTagName
	input.TagTitle = v.normalizedTitle
	input.MaterializeAutoTag = v.materializeAuto
	return input, nil
}

type fakeStaticAssignmentLanStatusReader struct {
	readLanStatusFn func(ctx context.Context) (LanStatusSnapshot, error)
}

func (reader *fakeStaticAssignmentLanStatusReader) ReadLanStatus(ctx context.Context) (LanStatusSnapshot, error) {
	return reader.readLanStatusFn(ctx)
}

type fakeStaticAssignmentDhcpConfigStore struct {
	loadLanStateFn func(ctx context.Context) (*LanDhcpState, error)
}

func (store *fakeStaticAssignmentDhcpConfigStore) LoadLanState(ctx context.Context) (*LanDhcpState, error) {
	return store.loadLanStateFn(ctx)
}

func (store *fakeStaticAssignmentDhcpConfigStore) ApplyTagConfig(ctx context.Context, input DhcpTagConfigInput) error {
	_ = ctx
	_ = input
	return errors.New("not implemented")
}

func (store *fakeStaticAssignmentDhcpConfigStore) ApplyGatewayConfig(ctx context.Context, input DhcpGatewayInput, lanStatus LanStatusSnapshot) error {
	_ = ctx
	_ = input
	_ = lanStatus
	return errors.New("not implemented")
}

func TestNewDefaultLanStaticAssignmentWriteServiceBuildsDependencies(t *testing.T) {
	t.Parallel()

	svc := NewDefaultLanStaticAssignmentWriteService()
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
	if svc.store == nil {
		t.Fatal("expected non-nil store")
	}
	if svc.tagValidator == nil {
		t.Fatal("expected non-nil tag validator")
	}
}

func TestLanStaticAssignmentWriteServiceRejectsMissingMAC(t *testing.T) {
	t.Parallel()

	svc := NewLanStaticAssignmentWriteService(&fakeStaticAssignmentWriteStore{}, &fakeStaticAssignmentTagValidator{})
	err := svc.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{Action: "add"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLanStaticAssignmentWriteServiceDeleteStillRequiresMAC(t *testing.T) {
	t.Parallel()

	svc := NewLanStaticAssignmentWriteService(&fakeStaticAssignmentWriteStore{}, &fakeStaticAssignmentTagValidator{})
	err := svc.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{Action: "delete"})
	if err == nil || err.Error() != "mac address is required" {
		t.Fatalf("err = %v, want mac address is required", err)
	}
}

func TestLanStaticAssignmentWriteServiceNormalizesModifyToAdd(t *testing.T) {
	t.Parallel()

	store := &fakeStaticAssignmentWriteStore{}
	validator := &fakeStaticAssignmentTagValidator{}
	svc := NewLanStaticAssignmentWriteService(store, validator)

	err := svc.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "modify",
		AssignedMAC: "AA:BB:CC:DD:EE:FF",
	})
	if err != nil {
		t.Fatalf("ApplyStaticAssignment returned error: %v", err)
	}
	if store.applyInput.Action != "add" {
		t.Fatalf("Action = %q, want %q", store.applyInput.Action, "add")
	}
}

func TestLanStaticAssignmentWriteServiceTreatsEmptyAddAsNoOp(t *testing.T) {
	t.Parallel()

	store := &fakeStaticAssignmentWriteStore{}
	svc := NewLanStaticAssignmentWriteService(store, &fakeStaticAssignmentTagValidator{})

	err := svc.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "add",
		AssignedMAC: "AA:BB:CC:DD:EE:01",
	})
	if err != nil {
		t.Fatalf("ApplyStaticAssignment returned error: %v", err)
	}
	if store.applyInput.AssignedMAC != "" {
		t.Fatalf("store should not be called, got %+v", store.applyInput)
	}
}

func TestLanStaticAssignmentWriteServicePropagatesTagValidationError(t *testing.T) {
	t.Parallel()

	svc := NewLanStaticAssignmentWriteService(
		&fakeStaticAssignmentWriteStore{},
		&fakeStaticAssignmentTagValidator{validateErr: errors.New("dhcp tag not found")},
	)
	err := svc.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "add",
		AssignedMAC: "AA:BB:CC:DD:EE:02",
		TagName:     "missing",
	})
	if err == nil || err.Error() != "dhcp tag not found" {
		t.Fatalf("err = %v, want dhcp tag not found", err)
	}
}

func TestLanStaticAssignmentWriteServiceNormalizesDefaultTagToEmptyBinding(t *testing.T) {
	t.Parallel()

	store := &fakeStaticAssignmentWriteStore{}
	validator := NewDefaultStaticAssignmentTagValidator(
		&fakeStaticAssignmentLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr:          "192.168.100.1",
					Nexthop:          "192.168.100.254",
					IsDefaultGateway: true,
				}, nil
			},
		},
		&fakeStaticAssignmentDhcpConfigStore{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{}, nil
			},
		},
	)
	svc := NewLanStaticAssignmentWriteService(store, validator)

	err := svc.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "add",
		AssignedMAC: "AA:BB:CC:DD:EE:04",
		TagName:     "t_auto_c0a86401",
	})
	if err != nil {
		t.Fatalf("ApplyStaticAssignment returned error: %v", err)
	}
	if store.applyInput.TagName != "" || store.applyInput.TagTitle != "default" {
		t.Fatalf("normalized input = %+v", store.applyInput)
	}
	if store.applyInput.MaterializeAutoTag {
		t.Fatalf("default tag should not materialize auto tag, got %+v", store.applyInput)
	}
}

func TestLanStaticAssignmentWriteServiceRejectsUnknownTagFromSharedDhcpState(t *testing.T) {
	t.Parallel()

	validator := NewDefaultStaticAssignmentTagValidator(
		&fakeStaticAssignmentLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{LanAddr: "192.168.100.1"}, nil
			},
		},
		&fakeStaticAssignmentDhcpConfigStore{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{}, nil
			},
		},
	)
	store := &fakeStaticAssignmentWriteStore{}
	svc := NewLanStaticAssignmentWriteService(store, validator)

	err := svc.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "add",
		AssignedMAC: "AA:BB:CC:DD:EE:05",
		TagName:     "missing",
	})
	if err == nil || err.Error() != "dhcp tag not found" {
		t.Fatalf("err = %v, want dhcp tag not found", err)
	}
	if store.applyInput.AssignedMAC != "" {
		t.Fatalf("store should not be called, got %+v", store.applyInput)
	}
}

func TestLanStaticAssignmentWriteServiceUsesSharedDhcpAutoCreatedFlagForMaterialization(t *testing.T) {
	t.Parallel()

	store := &fakeStaticAssignmentWriteStore{}
	validator := NewDefaultStaticAssignmentTagValidator(
		&fakeStaticAssignmentLanStatusReader{
			readLanStatusFn: func(ctx context.Context) (LanStatusSnapshot, error) {
				_ = ctx
				return LanStatusSnapshot{
					LanAddr:          "192.168.100.1",
					Nexthop:          "192.168.100.254",
					IsDefaultGateway: false,
				}, nil
			},
		},
		&fakeStaticAssignmentDhcpConfigStore{
			loadLanStateFn: func(ctx context.Context) (*LanDhcpState, error) {
				_ = ctx
				return &LanDhcpState{
					Tags: []DhcpTagRecord{
						{TagName: "manual_parent", TagTitle: "parent", Gateway: "192.168.100.254", AutoCreated: true},
					},
				}, nil
			},
		},
	)
	svc := NewLanStaticAssignmentWriteService(store, validator)

	err := svc.ApplyStaticAssignment(context.Background(), StaticAssignmentWriteInput{
		Action:      "add",
		AssignedMAC: "AA:BB:CC:DD:EE:06",
		TagName:     "manual_parent",
	})
	if err != nil {
		t.Fatalf("auto tag normalize returned error: %v", err)
	}
	if store.applyInput.TagName != "manual_parent" || store.applyInput.TagTitle != "parent" {
		t.Fatalf("normalized input = %+v", store.applyInput)
	}
	if !store.applyInput.MaterializeAutoTag {
		t.Fatalf("expected MaterializeAutoTag to be true, got %+v", store.applyInput)
	}
}

type fakeLanStaticAssignmentWriteFacade struct {
	input StaticAssignmentWriteInput
	err   error
}

func (svc *fakeLanStaticAssignmentWriteFacade) ApplyStaticAssignment(ctx context.Context, input StaticAssignmentWriteInput) error {
	_ = ctx
	svc.input = input
	return svc.err
}

func TestServiceBackendLanStaticAssignmentWriteCompatibilityDelegatesToService(t *testing.T) {
	original := newLanStaticAssignmentWriteService
	defer func() {
		newLanStaticAssignmentWriteService = original
	}()

	fake := &fakeLanStaticAssignmentWriteFacade{}
	newLanStaticAssignmentWriteService = func() lanStaticAssignmentWriteFacade {
		return fake
	}

	req := httptest.NewRequest("POST", "/cgi-bin/luci/istore/lanctrl/staticDeviceConfig/", strings.NewReader(`{"action":"add","assignedMac":"aa:bb:cc:dd:ee:30","assignedIP":"192.168.100.30","bindIP":true,"hostname":"printer","tagName":"office","tagTitle":"Office"}`))
	resp, err := (&ServiceBackend{}).PostLanStaticDeviceConfig(context.Background(), req)
	if err != nil || resp == nil {
		t.Fatalf("PostLanStaticDeviceConfig returned resp=%#v err=%v", resp, err)
	}
	if fake.input.Action != "add" || fake.input.AssignedMAC != "aa:bb:cc:dd:ee:30" || fake.input.AssignedIP != "192.168.100.30" || !fake.input.BindIP || fake.input.Hostname != "printer" || fake.input.TagName != "office" || fake.input.TagTitle != "Office" {
		t.Fatalf("input = %+v", fake.input)
	}
}

func TestServiceBackendLanStaticAssignmentWriteCompatibilityPropagatesError(t *testing.T) {
	original := newLanStaticAssignmentWriteService
	defer func() {
		newLanStaticAssignmentWriteService = original
	}()

	newLanStaticAssignmentWriteService = func() lanStaticAssignmentWriteFacade {
		return &fakeLanStaticAssignmentWriteFacade{err: errors.New("apply failed")}
	}

	req := httptest.NewRequest("POST", "/cgi-bin/luci/istore/lanctrl/staticDeviceConfig/", strings.NewReader(`{"action":"add","assignedMac":"aa:bb:cc:dd:ee:31"}`))
	resp, err := (&ServiceBackend{}).PostLanStaticDeviceConfig(context.Background(), req)
	if err == nil || err.Error() != "apply failed" || resp != nil {
		t.Fatalf("PostLanStaticDeviceConfig returned resp=%#v err=%v, want apply failed", resp, err)
	}
}

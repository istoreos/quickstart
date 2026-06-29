package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeNasLinkeaseEnableFacade struct {
	result *models.NasLinkeaseEnableResponseResult
	err    error
	calls  int
}

func (facade *fakeNasLinkeaseEnableFacade) Enable(ctx context.Context) (*models.NasLinkeaseEnableResponseResult, error) {
	facade.calls++
	return facade.result, facade.err
}

func TestNasServiceLinkeaseEnableCompatibilityDelegatesToService(t *testing.T) {
	originalFactory := newNasLinkeaseEnableServiceFacade
	defer func() {
		newNasLinkeaseEnableServiceFacade = originalFactory
	}()

	facade := &fakeNasLinkeaseEnableFacade{
		result: &models.NasLinkeaseEnableResponseResult{Port: "8897"},
	}
	newNasLinkeaseEnableServiceFacade = func() nasLinkeaseEnableFacade {
		return facade
	}

	resp, err := NasServiceLinkeaseEnable(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if facade.calls != 1 {
		t.Fatalf("expected one service call, got %d", facade.calls)
	}
	if resp == nil || resp.Result == nil || resp.Result.Port != "8897" {
		t.Fatalf("unexpected wrapper response: %#v", resp)
	}
}

func TestNasServiceLinkeaseEnableCompatibilityPropagatesServiceError(t *testing.T) {
	originalFactory := newNasLinkeaseEnableServiceFacade
	defer func() {
		newNasLinkeaseEnableServiceFacade = originalFactory
	}()

	serviceErr := errors.New("linkease enable failed")
	newNasLinkeaseEnableServiceFacade = func() nasLinkeaseEnableFacade {
		return &fakeNasLinkeaseEnableFacade{err: serviceErr}
	}

	if _, err := NasServiceLinkeaseEnable(context.Background(), nil); !errors.Is(err, serviceErr) {
		t.Fatalf("expected wrapper to propagate service error, got %v", err)
	}
}

func TestDefaultNasLinkeaseConfigReaderReadsTrimmedValues(t *testing.T) {
	original := readNasLinkeaseConfig
	defer func() {
		readNasLinkeaseConfig = original
	}()

	gotKeys := []string{}
	readNasLinkeaseConfig = func(ctx context.Context, key string) ([]byte, error) {
		gotKeys = append(gotKeys, key)
		switch key {
		case "enabled":
			return []byte("1\n"), nil
		case "port":
			return []byte("8897\n"), nil
		default:
			t.Fatalf("unexpected linkease key: %s", key)
			return nil, nil
		}
	}

	reader := &defaultNasLinkeaseConfigReader{}
	enabled, err := reader.ReadEnabled(context.Background())
	if err != nil {
		t.Fatalf("unexpected enabled read error: %v", err)
	}
	port, err := reader.ReadPort(context.Background())
	if err != nil {
		t.Fatalf("unexpected port read error: %v", err)
	}
	if enabled != "1" || port != "8897" {
		t.Fatalf("unexpected trimmed values: enabled=%q port=%q", enabled, port)
	}
	if !reflect.DeepEqual(gotKeys, []string{"enabled", "port"}) {
		t.Fatalf("unexpected read keys: %#v", gotKeys)
	}
}

func TestDefaultNasLinkeaseConfigWriterRunsEnableCommands(t *testing.T) {
	original := runNasLinkeaseEnable
	defer func() {
		runNasLinkeaseEnable = original
	}()

	var got []string
	runNasLinkeaseEnable = func(ctx context.Context, cmdList []string) error {
		got = append([]string(nil), cmdList...)
		return nil
	}

	err := (&defaultNasLinkeaseConfigWriter{}).Enable(context.Background())
	if err != nil {
		t.Fatalf("unexpected writer error: %v", err)
	}
	want := []string{
		"uci set linkease.@linkease[0].enabled=1",
		"uci commit linkease",
		"/etc/init.d/linkease restart",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("enable commands = %#v, want %#v", got, want)
	}
}

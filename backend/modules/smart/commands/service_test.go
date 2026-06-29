package commands

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

type fakeStore struct {
	outErrStdout string
	outErrErr    error
	output       string
	outputErr    error

	outErrCommands [][]string
	outputCommands [][]string
}

func (store *fakeStore) OutputWithErr(ctx context.Context, commands []string) (string, string, error) {
	store.outErrCommands = append(store.outErrCommands, append([]string(nil), commands...))
	return store.outErrStdout, "", store.outErrErr
}

func (store *fakeStore) Output(ctx context.Context, commands []string) (string, error) {
	store.outputCommands = append(store.outputCommands, append([]string(nil), commands...))
	return store.output, store.outputErr
}

func TestServiceStartTestPreservesLegacyResultMessages(t *testing.T) {
	store := &fakeStore{}
	service := NewService(store)

	resp, err := service.StartTest(context.Background(), models.SmartTestRequest{
		Type:       "short",
		DevicePath: "/dev/sda",
	})
	if err != nil {
		t.Fatalf("start test success: %v", err)
	}
	requireCommands(t, store.outErrCommands[0], []string{"smartctl -t short /dev/sda"})
	if resp.Result == nil || resp.Result.Result != "磁盘检测运行成功" {
		t.Fatalf("unexpected success response: %#v", resp)
	}

	store.outErrStdout = "Self-test already in progress"
	store.outErrErr = errors.New("smartctl failed")
	resp, err = service.StartTest(context.Background(), models.SmartTestRequest{
		Type:       "long",
		DevicePath: "/dev/sdb",
	})
	if err != nil {
		t.Fatalf("start test running: %v", err)
	}
	requireCommands(t, store.outErrCommands[1], []string{"smartctl -t long /dev/sdb"})
	if resp.Result == nil || resp.Result.Result != "磁盘测试正在运行\nSelf-test already in progress" {
		t.Fatalf("unexpected running response: %#v", resp)
	}
}

func TestServiceReadsTestResultAttributeAndExtendOutputs(t *testing.T) {
	store := &fakeStore{output: "smart output"}
	service := NewService(store)

	testResp, err := service.TestResult(context.Background(), models.SmartTestResultRequest{
		Type:       "selftest",
		DevicePath: "/dev/sda",
	})
	if err != nil {
		t.Fatalf("test result: %v", err)
	}
	requireCommands(t, store.outputCommands[0], []string{"smartctl -l selftest /dev/sda"})
	if testResp.Result == nil || testResp.Result.Result != "smart output" {
		t.Fatalf("unexpected test result response: %#v", testResp)
	}

	attrResp, err := service.AttributeResult(context.Background(), models.SmartAttributeResultRequest{DevicePath: "/dev/sdb"})
	if err != nil {
		t.Fatalf("attribute result: %v", err)
	}
	requireCommands(t, store.outputCommands[1], []string{"smartctl -A /dev/sdb"})
	if attrResp.Result == nil || attrResp.Result.Result != "smart output" {
		t.Fatalf("unexpected attribute response: %#v", attrResp)
	}

	extendResp, err := service.ExtendResult(context.Background(), models.SmartExtendResultRequest{DevicePath: "/dev/sdc"})
	if err != nil {
		t.Fatalf("extend result: %v", err)
	}
	requireCommands(t, store.outputCommands[2], []string{"smartctl -a /dev/sdc"})
	if extendResp.Result == nil || extendResp.Result.Result != "smart output" {
		t.Fatalf("unexpected extend response: %#v", extendResp)
	}
}

func TestServiceResultReadsPreserveLegacyErrorMessage(t *testing.T) {
	store := &fakeStore{outputErr: errors.New("smartctl failed")}
	service := NewService(store)

	if _, err := service.TestResult(context.Background(), models.SmartTestResultRequest{}); err == nil || err.Error() != "smart获取测试结果失败" {
		t.Fatalf("expected test result error, got %v", err)
	}
	if _, err := service.AttributeResult(context.Background(), models.SmartAttributeResultRequest{}); err == nil || err.Error() != "smart获取测试结果失败" {
		t.Fatalf("expected attribute result error, got %v", err)
	}
	if _, err := service.ExtendResult(context.Background(), models.SmartExtendResultRequest{}); err == nil || err.Error() != "smart获取测试结果失败" {
		t.Fatalf("expected extend result error, got %v", err)
	}
}

func requireCommands(t *testing.T, got []string, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected commands\nwant: %#v\n got: %#v", want, got)
	}
}

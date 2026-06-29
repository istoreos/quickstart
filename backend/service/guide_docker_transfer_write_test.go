package service

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var guideDockerTransferWriterTestMu sync.Mutex

func TestDefaultGuideDockerTransferWriterDelegatesValidationAndTransfer(t *testing.T) {
	guideDockerTransferWriterTestMu.Lock()
	defer guideDockerTransferWriterTestMu.Unlock()

	originalValidate := writeGuideDockerTransferValidatePath
	originalTransfer := writeGuideDockerTransferExecuteTransfer
	defer func() {
		writeGuideDockerTransferValidatePath = originalValidate
		writeGuideDockerTransferExecuteTransfer = originalTransfer
	}()

	var gotValidate struct {
		targetPath string
		originPath string
	}
	writeGuideDockerTransferValidatePath = func(ctx context.Context, targetPath, originPath string) error {
		gotValidate.targetPath = targetPath
		gotValidate.originPath = originPath
		return nil
	}

	expectedResult := &models.GuideDockerTransferResponseResult{
		Path:             "/mnt/data/docker",
		EmptyPathWarning: true,
	}
	var gotTransfer struct {
		targetPath   string
		force        bool
		overwriteDir bool
		originPath   string
	}
	writeGuideDockerTransferExecuteTransfer = func(ctx context.Context, targetPath string, force bool, overwriteDir bool, originPath string) (*models.GuideDockerTransferResponseResult, error) {
		gotTransfer.targetPath = targetPath
		gotTransfer.force = force
		gotTransfer.overwriteDir = overwriteDir
		gotTransfer.originPath = originPath
		return expectedResult, nil
	}

	writer := newDefaultGuideDockerTransferWriter()

	if err := writer.ValidateTargetPath(context.Background(), "/mnt/data/docker", "/mnt/origin/docker"); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if gotValidate.targetPath != "/mnt/data/docker" || gotValidate.originPath != "/mnt/origin/docker" {
		t.Fatalf("unexpected validate delegation: %#v", gotValidate)
	}

	result, err := writer.TransferPath(context.Background(), "/mnt/data/docker", true, false, "/mnt/origin/docker")
	if err != nil {
		t.Fatalf("unexpected transfer error: %v", err)
	}
	if result != expectedResult {
		t.Fatalf("unexpected transfer result: %#v", result)
	}
	if gotTransfer.targetPath != "/mnt/data/docker" || !gotTransfer.force || gotTransfer.overwriteDir || gotTransfer.originPath != "/mnt/origin/docker" {
		t.Fatalf("unexpected transfer delegation: %#v", gotTransfer)
	}
}

func TestDefaultGuideDockerTransferWriterPropagatesErrors(t *testing.T) {
	guideDockerTransferWriterTestMu.Lock()
	defer guideDockerTransferWriterTestMu.Unlock()

	originalValidate := writeGuideDockerTransferValidatePath
	originalTransfer := writeGuideDockerTransferExecuteTransfer
	originalRun := writeGuideDockerTransferRunCommands
	defer func() {
		writeGuideDockerTransferValidatePath = originalValidate
		writeGuideDockerTransferExecuteTransfer = originalTransfer
		writeGuideDockerTransferRunCommands = originalRun
	}()

	validateErr := errors.New("validate failed")
	writeGuideDockerTransferValidatePath = func(ctx context.Context, targetPath, originPath string) error {
		return validateErr
	}

	writer := newDefaultGuideDockerTransferWriter()
	if err := writer.ValidateTargetPath(context.Background(), "/mnt/data/docker", "/mnt/origin/docker"); !errors.Is(err, validateErr) {
		t.Fatalf("expected validate error, got %v", err)
	}

	transferErr := errors.New("transfer failed")
	writeGuideDockerTransferExecuteTransfer = func(ctx context.Context, targetPath string, force bool, overwriteDir bool, originPath string) (*models.GuideDockerTransferResponseResult, error) {
		return nil, transferErr
	}
	if _, err := writer.TransferPath(context.Background(), "/mnt/data/docker", true, false, "/mnt/origin/docker"); !errors.Is(err, transferErr) {
		t.Fatalf("expected transfer error, got %v", err)
	}

	runErr := errors.New("update failed")
	writeGuideDockerTransferRunCommands = func(ctx context.Context, cmds []string) error {
		return runErr
	}
	if err := writer.UpdateDockerRootPath(context.Background(), "/mnt/data/docker"); !errors.Is(err, runErr) {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestDefaultGuideDockerTransferWriterUpdatesDockerRootPath(t *testing.T) {
	guideDockerTransferWriterTestMu.Lock()
	defer guideDockerTransferWriterTestMu.Unlock()

	originalRun := writeGuideDockerTransferRunCommands
	defer func() {
		writeGuideDockerTransferRunCommands = originalRun
	}()

	var got []string
	writeGuideDockerTransferRunCommands = func(ctx context.Context, cmds []string) error {
		got = append([]string(nil), cmds...)
		return nil
	}

	writer := newDefaultGuideDockerTransferWriter()
	if err := writer.UpdateDockerRootPath(context.Background(), "/mnt/data/docker"); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	expected := []string{
		"uci set dockerd.globals.data_root='/mnt/data/docker'",
		"uci commit dockerd",
		"/etc/init.d/dockerd restart",
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("unexpected update command delegation: %#v", got)
	}
}

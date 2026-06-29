package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
	dockertransfer "github.com/linkease/quick-start/istore-backend/modules/guidestorage/dockertransfer"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideDockerTransferWriter interface {
	ValidateTargetPath(ctx context.Context, targetPath string, originPath string) error
	TransferPath(ctx context.Context, targetPath string, force bool, overwriteDir bool, originPath string) (*models.GuideDockerTransferResponseResult, error)
	UpdateDockerRootPath(ctx context.Context, path string) error
}

var writeGuideDockerTransferValidatePath = func(ctx context.Context, targetPath string, originPath string) error {
	return checkDockerPath(ctx, targetPath, originPath)
}

var writeGuideDockerTransferExecuteTransfer = func(ctx context.Context, targetPath string, force bool, overwriteDir bool, originPath string) (*models.GuideDockerTransferResponseResult, error) {
	return transferDockerPath(ctx, targetPath, force, overwriteDir, originPath)
}

var writeGuideDockerTransferRunCommands = func(ctx context.Context, cmds []string) error {
	return utils.BatchRun(ctx, cmds, 0)
}

type defaultGuideDockerTransferWriter struct{}

func newDefaultGuideDockerTransferWriter() *defaultGuideDockerTransferWriter {
	return &defaultGuideDockerTransferWriter{}
}

func (writer *defaultGuideDockerTransferWriter) ValidateTargetPath(ctx context.Context, targetPath string, originPath string) error {
	return writeGuideDockerTransferValidatePath(ctx, targetPath, originPath)
}

func (writer *defaultGuideDockerTransferWriter) TransferPath(ctx context.Context, targetPath string, force bool, overwriteDir bool, originPath string) (*models.GuideDockerTransferResponseResult, error) {
	return writeGuideDockerTransferExecuteTransfer(ctx, targetPath, force, overwriteDir, originPath)
}

func (writer *defaultGuideDockerTransferWriter) UpdateDockerRootPath(ctx context.Context, path string) error {
	return writeGuideDockerTransferRunCommands(ctx, dockertransfer.BuildUpdateCommands(path))
}

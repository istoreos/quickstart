package service

import (
	"context"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
	dockertransfer "github.com/linkease/quick-start/istore-backend/modules/guidestorage/dockertransfer"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type GuideDockerTransferReader interface {
	ReadDockerRootPath(ctx context.Context) (*GuideDockerRootSnapshot, error)
	ReadPartitionCandidates(ctx context.Context) ([]*GuideDockerPartitionCandidate, error)
}

var readGuideDockerTransferRootPath = func(ctx context.Context) (string, error) {
	out, err := utils.BatchOutputCmd(ctx, "uci get dockerd.globals.data_root", 0)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

var readGuideDockerTransferDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
	return getAllDisks(ctx)
}

type defaultGuideDockerTransferReader struct{}

func newDefaultGuideDockerTransferReader() *defaultGuideDockerTransferReader {
	return &defaultGuideDockerTransferReader{}
}

func (reader *defaultGuideDockerTransferReader) ReadDockerRootPath(ctx context.Context) (*GuideDockerRootSnapshot, error) {
	path, err := readGuideDockerTransferRootPath(ctx)
	if err != nil {
		return nil, err
	}
	return &GuideDockerRootSnapshot{Path: path}, nil
}

func (reader *defaultGuideDockerTransferReader) ReadPartitionCandidates(ctx context.Context) ([]*GuideDockerPartitionCandidate, error) {
	disks, err := readGuideDockerTransferDisks(ctx)
	if err != nil {
		return nil, err
	}
	return dockertransfer.BuildPartitionCandidates(disks), nil
}

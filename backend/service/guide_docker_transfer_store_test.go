package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var guideDockerTransferReaderTestMu sync.Mutex

func TestDefaultGuideDockerTransferReaderReadsRootAndCandidates(t *testing.T) {
	guideDockerTransferReaderTestMu.Lock()
	defer guideDockerTransferReaderTestMu.Unlock()

	originalRootPath := readGuideDockerTransferRootPath
	originalDisks := readGuideDockerTransferDisks
	defer func() {
		readGuideDockerTransferRootPath = originalRootPath
		readGuideDockerTransferDisks = originalDisks
	}()

	readGuideDockerTransferRootPath = func(ctx context.Context) (string, error) {
		return "/mnt/docker-root", nil
	}
	readGuideDockerTransferDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
		return []*models.NasDiskInfo{
			{
				Childrens: []*models.PartitionInfo{
					{
						MountPoint: "/mnt/data",
						Filesystem: "ext4",
						SizeInt:    "17179869184",
					},
				},
			},
		}, nil
	}

	reader := newDefaultGuideDockerTransferReader()

	root, err := reader.ReadDockerRootPath(context.Background())
	if err != nil {
		t.Fatalf("unexpected root-path read error: %v", err)
	}
	if root.Path != "/mnt/docker-root" {
		t.Fatalf("unexpected docker root snapshot: %#v", root)
	}

	candidates, err := reader.ReadPartitionCandidates(context.Background())
	if err != nil {
		t.Fatalf("unexpected partition-candidate read error: %v", err)
	}
	if len(candidates) != 1 || candidates[0].Path != "/mnt/data/docker" {
		t.Fatalf("unexpected partition candidates: %#v", candidates)
	}
}

func TestDefaultGuideDockerTransferReaderPropagatesErrors(t *testing.T) {
	guideDockerTransferReaderTestMu.Lock()
	defer guideDockerTransferReaderTestMu.Unlock()

	originalRootPath := readGuideDockerTransferRootPath
	originalDisks := readGuideDockerTransferDisks
	defer func() {
		readGuideDockerTransferRootPath = originalRootPath
		readGuideDockerTransferDisks = originalDisks
	}()

	rootErr := errors.New("root failed")
	readGuideDockerTransferRootPath = func(ctx context.Context) (string, error) {
		return "", rootErr
	}

	reader := newDefaultGuideDockerTransferReader()
	if _, err := reader.ReadDockerRootPath(context.Background()); !errors.Is(err, rootErr) {
		t.Fatalf("expected root-path error, got %v", err)
	}

	readGuideDockerTransferRootPath = func(ctx context.Context) (string, error) {
		return "/mnt/docker-root", nil
	}
	diskErr := errors.New("disk failed")
	readGuideDockerTransferDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
		return nil, diskErr
	}
	if _, err := reader.ReadPartitionCandidates(context.Background()); !errors.Is(err, diskErr) {
		t.Fatalf("expected disk error, got %v", err)
	}
}

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestDefaultGuideDockerRuntimeReaderReadsRuntimeSnapshot(t *testing.T) {
	originalInstalled := readGuideDockerInstalled
	originalRunning := readGuideDockerRunning
	originalRootPath := readGuideDockerRootPath
	originalDisks := readGuideDockerDisks
	defer func() {
		readGuideDockerInstalled = originalInstalled
		readGuideDockerRunning = originalRunning
		readGuideDockerRootPath = originalRootPath
		readGuideDockerDisks = originalDisks
	}()

	readGuideDockerInstalled = func(ctx context.Context) (bool, error) { return true, nil }
	readGuideDockerRunning = func() bool { return true }
	readGuideDockerRootPath = func(ctx context.Context) (string, error) { return "/mnt/docker", nil }
	readGuideDockerDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
		return []*models.NasDiskInfo{
			{
				IsDockerRoot: true,
				Childrens: []*models.PartitionInfo{
					{IsDockerRoot: true, Filesystem: "ext4"},
				},
			},
		}, nil
	}

	reader := newDefaultGuideDockerRuntimeReader()
	snapshot, err := reader.ReadDockerRuntime(context.Background())
	if err != nil {
		t.Fatalf("unexpected docker runtime read error: %v", err)
	}
	if !snapshot.Installed || !snapshot.Running || snapshot.Path != "/mnt/docker" || snapshot.ErrorInfo != "" {
		t.Fatalf("unexpected docker runtime snapshot: %#v", snapshot)
	}
}

func TestDefaultGuideDockerRuntimeReaderNotInstalledSkipsFurtherReads(t *testing.T) {
	originalInstalled := readGuideDockerInstalled
	originalRootPath := readGuideDockerRootPath
	defer func() {
		readGuideDockerInstalled = originalInstalled
		readGuideDockerRootPath = originalRootPath
	}()

	readGuideDockerInstalled = func(ctx context.Context) (bool, error) { return false, nil }
	readGuideDockerRootPath = func(ctx context.Context) (string, error) {
		t.Fatal("root path should not be read when docker is not installed")
		return "", nil
	}

	reader := newDefaultGuideDockerRuntimeReader()
	snapshot, err := reader.ReadDockerRuntime(context.Background())
	if err != nil {
		t.Fatalf("unexpected not-installed read error: %v", err)
	}
	if snapshot.Installed || snapshot.Running || snapshot.Path != "" || snapshot.ErrorInfo != "" {
		t.Fatalf("unexpected not-installed snapshot: %#v", snapshot)
	}
}

func TestDefaultGuideDockerRuntimeReaderPropagatesRootPathAndDiskErrors(t *testing.T) {
	originalInstalled := readGuideDockerInstalled
	originalRootPath := readGuideDockerRootPath
	originalDisks := readGuideDockerDisks
	defer func() {
		readGuideDockerInstalled = originalInstalled
		readGuideDockerRootPath = originalRootPath
		readGuideDockerDisks = originalDisks
	}()

	readGuideDockerInstalled = func(ctx context.Context) (bool, error) { return true, nil }
	rootErr := errors.New("root path failed")
	readGuideDockerRootPath = func(ctx context.Context) (string, error) { return "", rootErr }

	reader := newDefaultGuideDockerRuntimeReader()
	if _, err := reader.ReadDockerRuntime(context.Background()); !errors.Is(err, rootErr) {
		t.Fatalf("expected root path error, got %v", err)
	}

	readGuideDockerRootPath = func(ctx context.Context) (string, error) { return "/mnt/docker", nil }
	diskErr := errors.New("disks failed")
	readGuideDockerDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) { return nil, diskErr }

	if _, err := reader.ReadDockerRuntime(context.Background()); !errors.Is(err, diskErr) {
		t.Fatalf("expected disk read error, got %v", err)
	}
}

func TestBuildGuideDockerRuntimeWarning(t *testing.T) {
	systemWarning := buildGuideDockerRuntimeWarning([]*models.NasDiskInfo{
		{
			IsDockerRoot: true,
			Childrens: []*models.PartitionInfo{
				{IsDockerRoot: true, IsSystemRoot: true},
			},
		},
	})
	if systemWarning != "当前docker根目录位于系统根目录，可能会占用大量系统空间，影响系统的正常运行，建议使用docker迁移向导将docker根目录迁移到外置硬盘上" {
		t.Fatalf("unexpected system-root warning: %q", systemWarning)
	}

	ntfsWarning := buildGuideDockerRuntimeWarning([]*models.NasDiskInfo{
		{
			IsDockerRoot: true,
			Childrens: []*models.PartitionInfo{
				{IsDockerRoot: true, Filesystem: "ntfs"},
			},
		},
	})
	if ntfsWarning != "当前docker根目录位于ntfs分区，会出现很多奇怪的问题，建议迁移到ext4分区" {
		t.Fatalf("unexpected ntfs warning: %q", ntfsWarning)
	}

	overlayWarning := buildGuideDockerRuntimeWarning([]*models.NasDiskInfo{
		{
			IsDockerRoot: true,
			Childrens: []*models.PartitionInfo{
				{IsDockerRoot: false, Filesystem: "ext4"},
			},
		},
	})
	if overlayWarning != "当前docker根目录位于系统根目录，可能会占用大量系统空间，影响系统的正常运行，建议使用docker迁移向导将docker根目录迁移到外置硬盘上" {
		t.Fatalf("unexpected overlay warning: %q", overlayWarning)
	}
}

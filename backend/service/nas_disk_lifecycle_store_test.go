package service

import (
	"context"
	"errors"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestDefaultNasDiskSnapshotReaderBuildsSnapshotsFromCurrentReaders(t *testing.T) {
	originalAll := readNasAllDisks
	originalDisk := readNasDisk
	originalDiskIncludeFree := readNasDiskIncludeFree
	defer func() {
		readNasAllDisks = originalAll
		readNasDisk = originalDisk
		readNasDiskIncludeFree = originalDiskIncludeFree
	}()

	readNasAllDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
		return []*models.NasDiskInfo{
			{Name: "sda", Path: "/dev/sda", Childrens: []*models.PartitionInfo{{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-sda1"}}},
		}, nil
	}
	readNasDisk = func(name string) (*models.NasDiskInfo, error) {
		return &models.NasDiskInfo{Name: name, Path: "/dev/" + name}, nil
	}
	readNasDiskIncludeFree = func(name string) (*models.NasDiskInfo, error) {
		return &models.NasDiskInfo{
			Name: name,
			Path: "/dev/" + name,
			Childrens: []*models.PartitionInfo{
				{Name: name + "1", Path: "/dev/" + name + "1", Filesystem: "Free Space"},
			},
		}, nil
	}

	reader := newDefaultNasDiskSnapshotReader()

	all, err := reader.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected ReadAll error: %v", err)
	}
	if len(all) != 1 || all[0].Name != "sda" || len(all[0].Partitions) != 1 {
		t.Fatalf("unexpected all-disk snapshots: %#v", all)
	}

	disk, err := reader.ReadDisk(context.Background(), "sdb")
	if err != nil {
		t.Fatalf("unexpected ReadDisk error: %v", err)
	}
	if disk == nil || disk.Name != "sdb" || disk.Path != "/dev/sdb" {
		t.Fatalf("unexpected disk snapshot: %#v", disk)
	}

	diskWithFree, err := reader.ReadDiskIncludeFree(context.Background(), "sdc")
	if err != nil {
		t.Fatalf("unexpected ReadDiskIncludeFree error: %v", err)
	}
	if diskWithFree == nil || len(diskWithFree.Partitions) != 1 || diskWithFree.Partitions[0].Filesystem != "Free Space" {
		t.Fatalf("unexpected disk-with-free snapshot: %#v", diskWithFree)
	}
}

func TestDefaultNasDiskSnapshotReaderPropagatesReaderErrors(t *testing.T) {
	originalAll := readNasAllDisks
	originalDisk := readNasDisk
	originalDiskIncludeFree := readNasDiskIncludeFree
	defer func() {
		readNasAllDisks = originalAll
		readNasDisk = originalDisk
		readNasDiskIncludeFree = originalDiskIncludeFree
	}()

	allErr := errors.New("all disks failed")
	readNasAllDisks = func(ctx context.Context) ([]*models.NasDiskInfo, error) {
		return nil, allErr
	}
	reader := newDefaultNasDiskSnapshotReader()
	if _, err := reader.ReadAll(context.Background()); !errors.Is(err, allErr) {
		t.Fatalf("expected ReadAll error, got %v", err)
	}

	diskErr := errors.New("disk failed")
	readNasDisk = func(name string) (*models.NasDiskInfo, error) {
		return nil, diskErr
	}
	if _, err := reader.ReadDisk(context.Background(), "sda"); !errors.Is(err, diskErr) {
		t.Fatalf("expected ReadDisk error, got %v", err)
	}

	includeFreeErr := errors.New("include free failed")
	readNasDiskIncludeFree = func(name string) (*models.NasDiskInfo, error) {
		return nil, includeFreeErr
	}
	if _, err := reader.ReadDiskIncludeFree(context.Background(), "sda"); !errors.Is(err, includeFreeErr) {
		t.Fatalf("expected ReadDiskIncludeFree error, got %v", err)
	}
}

func TestDefaultNasDiskCommandStoreDelegatesToCurrentHelpers(t *testing.T) {
	t.Parallel()

	store := newDefaultNasDiskCommandStore()
	if _, ok := store.(*defaultNasDiskCommandStore); !ok {
		t.Fatalf("expected default command store implementation")
	}
}

func TestDefaultNasDiskMountPointGeneratorDelegatesToCurrentHelper(t *testing.T) {
	t.Parallel()

	generator := newDefaultNasDiskMountPointGenerator()
	if _, ok := generator.(*defaultNasDiskMountPointGenerator); !ok {
		t.Fatalf("expected default mountpoint generator implementation")
	}
}

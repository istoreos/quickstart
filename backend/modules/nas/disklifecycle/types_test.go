package disklifecycle

import (
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestBuildDiskSnapshotsSkipsNilRecords(t *testing.T) {
	t.Parallel()

	snapshots := BuildDiskSnapshots([]*models.NasDiskInfo{
		nil,
		{
			Name:          "sda",
			PartLabelType: "GPT",
			Path:          "/dev/sda",
			Childrens: []*models.PartitionInfo{
				nil,
				{
					Filesystem: "ext4",
					MountPoint: "/mnt/data_sda1",
					Name:       "sda1",
					Path:       "/dev/sda1",
					SecEnd:     4096,
					SecStart:   2048,
					UUID:       "uuid-sda1",
				},
			},
		},
	})

	if len(snapshots) != 1 {
		t.Fatalf("expected nil disk records to be skipped, got %#v", snapshots)
	}
	if snapshots[0].Name != "sda" || snapshots[0].Path != "/dev/sda" || len(snapshots[0].Partitions) != 1 {
		t.Fatalf("unexpected disk snapshot: %#v", snapshots[0])
	}
	if snapshots[0].Partitions[0].UUID != "uuid-sda1" || snapshots[0].Partitions[0].Filesystem != "ext4" {
		t.Fatalf("unexpected partition snapshot: %#v", snapshots[0].Partitions[0])
	}
}

func TestFindPartitionByIdentityMatchesUUIDAndPath(t *testing.T) {
	t.Parallel()

	disks := []DiskSnapshot{
		{
			Name: "sda",
			Path: "/dev/sda",
			Partitions: []PartitionSnapshot{
				{Name: "sda1", Path: "/dev/sda1", UUID: "uuid-a"},
				{Name: "sda2", Path: "/dev/sda2", UUID: "uuid-b"},
			},
		},
	}

	disk, partition := findPartitionByIdentity(disks, "uuid-b", "/dev/sda2")
	if disk == nil || partition == nil {
		t.Fatalf("expected matching partition, got disk=%#v partition=%#v", disk, partition)
	}
	if disk.Name != "sda" || partition.Name != "sda2" {
		t.Fatalf("unexpected match result: disk=%#v partition=%#v", disk, partition)
	}

	disk, partition = findPartitionByIdentity(disks, "uuid-b", "/dev/sda1")
	if disk != nil || partition != nil {
		t.Fatalf("expected UUID+Path match semantics, got disk=%#v partition=%#v", disk, partition)
	}
}

func TestIsWholeDiskTargetDistinguishesDiskAndPartitionPaths(t *testing.T) {
	t.Parallel()

	disks := []DiskSnapshot{
		{
			Name: "sda",
			Path: "/dev/sda",
			Partitions: []PartitionSnapshot{
				{Name: "sda1", Path: "/dev/sda1"},
			},
		},
	}

	if !isWholeDiskTarget(disks, "/dev/sda") {
		t.Fatalf("expected whole-disk target for /dev/sda")
	}
	if isWholeDiskTarget(disks, "/dev/sda1") {
		t.Fatalf("expected partition path not to be treated as whole-disk target")
	}
}

func TestUsesDirectDeviceFormatDetectsMDDevicesOnly(t *testing.T) {
	t.Parallel()

	if !usesDirectDeviceFormat("md0") {
		t.Fatalf("expected md devices to use direct device format")
	}
	if usesDirectDeviceFormat("sda") {
		t.Fatalf("expected non-md devices not to use direct device format")
	}
}

func TestFindLastPartitionSelectsRefreshTarget(t *testing.T) {
	t.Parallel()

	disk := DiskSnapshot{
		Name: "sda",
		Partitions: []PartitionSnapshot{
			{Name: "sda1", Path: "/dev/sda1", Filesystem: "ext4"},
			{Name: "sda2", Path: "/dev/sda2", Filesystem: "Free Space"},
		},
	}

	last := findLastPartition(disk)
	if last == nil || last.Name != "sda2" {
		t.Fatalf("expected last partition to be selected, got %#v", last)
	}

	if findLastPartition(DiskSnapshot{Name: "empty"}) != nil {
		t.Fatalf("expected nil last partition for empty disk")
	}
}

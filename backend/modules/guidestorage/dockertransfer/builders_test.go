package dockertransfer

import (
	"errors"
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestBuildUpdateCommandsReturnsExactDockerRootCommandList(t *testing.T) {
	cmds := BuildUpdateCommands("/mnt/data/docker")
	expected := []string{
		"uci set dockerd.globals.data_root='/mnt/data/docker'",
		"uci commit dockerd",
		"/etc/init.d/dockerd restart",
	}
	if !reflect.DeepEqual(cmds, expected) {
		t.Fatalf("unexpected docker transfer update commands: %#v", cmds)
	}
}

func TestBuildPartitionCandidatesFromDiskSnapshots(t *testing.T) {
	candidates := BuildPartitionCandidates([]*models.NasDiskInfo{
		{
			Childrens: []*models.PartitionInfo{
				{MountPoint: "/mnt/root", Filesystem: "ext4", IsSystemRoot: true, SizeInt: "17179869184"},
				{MountPoint: "/mnt/ro", Filesystem: "ext4", IsReadOnly: true, SizeInt: "17179869184"},
				{MountPoint: "/mnt/squash", Filesystem: "squashfs", SizeInt: "17179869184"},
				{MountPoint: "/mnt/ntfs", Filesystem: "ntfs", SizeInt: "17179869184"},
				{MountPoint: "/mnt/vfat", Filesystem: "vfat", SizeInt: "17179869184"},
				{MountPoint: "/mnt/exfat", Filesystem: "exfat", SizeInt: "17179869184"},
				{MountPoint: "/mnt/swap", Filesystem: "swap", SizeInt: "17179869184"},
				{MountPoint: "/mnt/small", Filesystem: "ext4", SizeInt: "8589934592"},
				{MountPoint: "/mnt/bad-size", Filesystem: "ext4", SizeInt: "not-a-number"},
				{MountPoint: "/mnt/data", Filesystem: "ext4", SizeInt: "17179869184"},
				{MountPoint: "/mnt/data2", Filesystem: "btrfs", SizeInt: "17179869185"},
				{MountPoint: "", Filesystem: "ext4", SizeInt: "17179869184"},
			},
		},
	})

	expected := []*PartitionCandidate{
		{MountPoint: "/mnt/data", Filesystem: "ext4", SizeBytes: 17179869184, Path: "/mnt/data/docker"},
		{MountPoint: "/mnt/data2", Filesystem: "btrfs", SizeBytes: 17179869185, Path: "/mnt/data2/docker"},
	}
	if !reflect.DeepEqual(candidates, expected) {
		t.Fatalf("unexpected partition candidates: %#v", candidates)
	}
}

func TestBuildEmptyTargetDirectoryWarningResult(t *testing.T) {
	result, err := BuildEmptyTargetDirectoryWarning("/mnt/data/docker")
	if !errors.Is(err, ErrEmptyTargetDirectory) {
		t.Fatalf("expected empty target directory error, got %v", err)
	}
	if err == nil || err.Error() != "目标路径不为空" {
		t.Fatalf("expected legacy warning message, got %v", err)
	}
	if result == nil || !result.EmptyPathWarning || result.Path != "/mnt/data/docker" {
		t.Fatalf("unexpected warning result: %#v", result)
	}
}

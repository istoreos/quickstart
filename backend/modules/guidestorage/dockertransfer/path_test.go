package dockertransfer

import "testing"

func TestValidatePathSnapshotRejectsLegacyDockerTransferPathRules(t *testing.T) {
	tests := []struct {
		name    string
		input   PathSnapshot
		wantErr string
	}{
		{
			name: "same target and origin path",
			input: PathSnapshot{
				TargetPath: "/mnt/data/docker",
				OriginPath: "/mnt/data/docker",
			},
			wantErr: "不能选择同一个目录",
		},
		{
			name: "overlay filesystem",
			input: PathSnapshot{
				TargetPath:   "/mnt/data/docker",
				OriginPath:   "/mnt/origin/docker",
				TargetFSType: "overlay",
			},
			wantErr: "路径不合法，不能在系统目录或者mnt根目录上",
		},
		{
			name: "tmpfs filesystem",
			input: PathSnapshot{
				TargetPath:   "/mnt/data/docker",
				OriginPath:   "/mnt/origin/docker",
				TargetFSType: "tmpfs",
			},
			wantErr: "路径不合法，不能在系统目录或者mnt根目录上",
		},
		{
			name: "ntfs filesystem",
			input: PathSnapshot{
				TargetPath:   "/mnt/data/docker",
				OriginPath:   "/mnt/origin/docker",
				TargetFSType: "ntfs",
			},
			wantErr: "路径不合法，不能在ntfs分区的目录上",
		},
		{
			name: "same overlay source",
			input: PathSnapshot{
				TargetPath:    "/mnt/data/docker",
				OriginPath:    "/mnt/origin/docker",
				TargetSource:  "/dev/mmcblk0p2",
				OverlaySource: "/dev/mmcblk0p2",
			},
			wantErr: "路径不合法，不能在系统目录上",
		},
		{
			name: "same origin partition",
			input: PathSnapshot{
				TargetPath:   "/mnt/data/docker",
				OriginPath:   "/mnt/origin/docker",
				TargetSource: "/dev/sda1",
				OriginSource: "/dev/sda1",
			},
			wantErr: "不能选择原docker根目录所在分区",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePathSnapshot(tt.input)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidatePathSnapshotAllowsValidTarget(t *testing.T) {
	err := ValidatePathSnapshot(PathSnapshot{
		TargetPath:    "/mnt/data/docker",
		OriginPath:    "/mnt/origin/docker",
		TargetSource:  "/dev/sda1",
		TargetFSType:  "ext4",
		OverlaySource: "/dev/mmcblk0p2",
		OriginSource:  "/dev/sdb1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

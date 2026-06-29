package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCheckDockerPathCompatibilityUsesServiceShellSnapshot(t *testing.T) {
	guideDockerTransferWriterTestMu.Lock()
	defer guideDockerTransferWriterTestMu.Unlock()

	originalOutput := runGuideDockerTransferPathOutput
	defer func() { runGuideDockerTransferPathOutput = originalOutput }()

	var gotCommands []string
	runGuideDockerTransferPathOutput = func(ctx context.Context, cmd string) ([]byte, error) {
		gotCommands = append(gotCommands, cmd)
		switch cmd {
		case "[ -d '/mnt/new/docker' ] || mkdir -p '/mnt/new/docker'":
			return []byte{}, nil
		case "findmnt -T '/mnt/new/docker' -o SOURCE,FSTYPE --json":
			return []byte(`{"filesystems":[{"source":"/dev/sda1","fstype":"ext4"}]}`), nil
		case "findmnt -T /overlay -o SOURCE|sed -n 2p":
			return []byte("/dev/mmcblk0p2\n"), nil
		case "findmnt -T '/mnt/origin/docker' -o SOURCE|sed -n 2p":
			return []byte("/dev/sdb1\n"), nil
		default:
			t.Fatalf("unexpected command: %s", cmd)
		}
		return nil, nil
	}

	if err := checkDockerPath(context.Background(), "/mnt/new/docker", "/mnt/origin/docker"); err != nil {
		t.Fatalf("unexpected checkDockerPath error: %v", err)
	}

	expectedCommands := []string{
		"[ -d '/mnt/new/docker' ] || mkdir -p '/mnt/new/docker'",
		"findmnt -T '/mnt/new/docker' -o SOURCE,FSTYPE --json",
		"findmnt -T /overlay -o SOURCE|sed -n 2p",
		"findmnt -T '/mnt/origin/docker' -o SOURCE|sed -n 2p",
	}
	if !reflect.DeepEqual(gotCommands, expectedCommands) {
		t.Fatalf("unexpected command sequence: %#v", gotCommands)
	}
}

func TestCheckDockerPathCompatibilityPreservesModuleRuleErrors(t *testing.T) {
	guideDockerTransferWriterTestMu.Lock()
	defer guideDockerTransferWriterTestMu.Unlock()

	tests := []struct {
		name       string
		targetJSON string
		overlay    string
		origin     string
		wantErr    string
	}{
		{
			name:       "overlay filesystem",
			targetJSON: `{"filesystems":[{"source":"overlay","fstype":"overlay"}]}`,
			overlay:    "/dev/mmcblk0p2\n",
			origin:     "/dev/sdb1\n",
			wantErr:    "路径不合法，不能在系统目录或者mnt根目录上",
		},
		{
			name:       "ntfs filesystem",
			targetJSON: `{"filesystems":[{"source":"/dev/sda1","fstype":"ntfs"}]}`,
			overlay:    "/dev/mmcblk0p2\n",
			origin:     "/dev/sdb1\n",
			wantErr:    "路径不合法，不能在ntfs分区的目录上",
		},
		{
			name:       "same overlay source",
			targetJSON: `{"filesystems":[{"source":"/dev/mmcblk0p2","fstype":"ext4"}]}`,
			overlay:    "/dev/mmcblk0p2\n",
			origin:     "/dev/sdb1\n",
			wantErr:    "路径不合法，不能在系统目录上",
		},
		{
			name:       "same origin partition",
			targetJSON: `{"filesystems":[{"source":"/dev/sdb1","fstype":"ext4"}]}`,
			overlay:    "/dev/mmcblk0p2\n",
			origin:     "/dev/sdb1\n",
			wantErr:    "不能选择原docker根目录所在分区",
		},
	}

	originalOutput := runGuideDockerTransferPathOutput
	defer func() { runGuideDockerTransferPathOutput = originalOutput }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runGuideDockerTransferPathOutput = func(ctx context.Context, cmd string) ([]byte, error) {
				switch cmd {
				case "[ -d '/mnt/new/docker' ] || mkdir -p '/mnt/new/docker'":
					return []byte{}, nil
				case "findmnt -T '/mnt/new/docker' -o SOURCE,FSTYPE --json":
					return []byte(tt.targetJSON), nil
				case "findmnt -T /overlay -o SOURCE|sed -n 2p":
					return []byte(tt.overlay), nil
				case "findmnt -T '/mnt/origin/docker' -o SOURCE|sed -n 2p":
					return []byte(tt.origin), nil
				default:
					t.Fatalf("unexpected command: %s", cmd)
				}
				return nil, nil
			}

			err := checkDockerPath(context.Background(), "/mnt/new/docker", "/mnt/origin/docker")
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCheckDockerPathCompatibilityPreservesShellAndPathInfoErrors(t *testing.T) {
	guideDockerTransferWriterTestMu.Lock()
	defer guideDockerTransferWriterTestMu.Unlock()

	if err := checkDockerPath(context.Background(), "/mnt/same/docker", "/mnt/same/docker"); err == nil || err.Error() != "不能选择同一个目录" {
		t.Fatalf("expected same-dir error, got %v", err)
	}

	originalOutput := runGuideDockerTransferPathOutput
	defer func() { runGuideDockerTransferPathOutput = originalOutput }()

	shellErr := errors.New("mkdir failed")
	runGuideDockerTransferPathOutput = func(ctx context.Context, cmd string) ([]byte, error) {
		return nil, shellErr
	}
	if err := checkDockerPath(context.Background(), "/mnt/new/docker", "/mnt/origin/docker"); !errors.Is(err, shellErr) {
		t.Fatalf("expected shell error, got %v", err)
	}

	runGuideDockerTransferPathOutput = func(ctx context.Context, cmd string) ([]byte, error) {
		switch cmd {
		case "[ -d '/mnt/new/docker' ] || mkdir -p '/mnt/new/docker'":
			return []byte{}, nil
		case "findmnt -T '/mnt/new/docker' -o SOURCE,FSTYPE --json":
			return []byte(`{"filesystems":[]}`), nil
		default:
			t.Fatalf("unexpected command: %s", cmd)
		}
		return nil, nil
	}
	if err := checkDockerPath(context.Background(), "/mnt/new/docker", "/mnt/origin/docker"); err == nil || err.Error() != "路径信息获取失败" {
		t.Fatalf("expected path info error, got %v", err)
	}
}

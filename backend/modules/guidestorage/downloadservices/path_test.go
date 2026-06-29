package downloadservices

import (
	"reflect"
	"testing"
)

func TestValidateDownloadPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "mnt path", path: "/mnt/data"},
		{name: "root path", path: "/root/download"},
		{name: "relative path", path: "mnt/data", wantErr: "路径错误，请输入绝对路径"},
		{name: "unsupported absolute path", path: "/tmp/data", wantErr: "路径错误，必须选择硬盘的路径或者/root的路径"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDownloadPath(tt.path)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("expected error %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestBuildEnsureDownloadDirCommands(t *testing.T) {
	got := BuildEnsureDownloadDirCommands("/mnt/download")
	want := []string{
		"if [ ! -d '/mnt/download' ]; then mkdir -p '/mnt/download'; fi",
		"chmod 777 '/mnt/download'",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

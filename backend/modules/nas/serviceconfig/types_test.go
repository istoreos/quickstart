package serviceconfig

import "testing"

func TestBuildSambaURL(t *testing.T) {
	t.Parallel()

	result := BuildSambaURL("192.168.100.1", "share")
	if result != "smb://192.168.100.1/share" {
		t.Fatalf("unexpected samba url: %q", result)
	}
}

func TestBuildWebdavURL(t *testing.T) {
	t.Parallel()

	result := BuildWebdavURL("192.168.100.1", "5244")
	if result != "http://192.168.100.1:5244" {
		t.Fatalf("unexpected webdav url: %q", result)
	}
}

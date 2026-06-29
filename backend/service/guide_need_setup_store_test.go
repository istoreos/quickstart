package service

import (
	"errors"
	"testing"
)

func TestCheckNeedSetupFromShadowReadsLegacyPaths(t *testing.T) {
	originalReadFile := readGuideSetupShadowFile
	defer func() {
		readGuideSetupShadowFile = originalReadFile
	}()

	paths := make([]string, 0, 2)
	readGuideSetupShadowFile = func(path string) ([]byte, error) {
		paths = append(paths, path)
		switch path {
		case "/rom/etc/shadow":
			return []byte("root:$1$abc:0:0:99999:7:::\n"), nil
		case "/etc/shadow":
			return []byte("root:$1$abc:0:0:99999:7:::\n"), nil
		default:
			return nil, errors.New("unexpected path")
		}
	}

	need, err := checkNeedSetupFromShadow()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !need {
		t.Fatal("expected unchanged root password to need setup")
	}
	if len(paths) != 2 || paths[0] != "/rom/etc/shadow" || paths[1] != "/etc/shadow" {
		t.Fatalf("unexpected paths: %#v", paths)
	}
}

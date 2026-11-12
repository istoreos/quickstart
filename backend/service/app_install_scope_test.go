package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/modules/app/installjob"
)

func TestCardInstallationTrackingDoesNotChangeOtherApplications(t *testing.T) {
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	installer := filepath.Join(dir, "is-opkg")
	if err := os.WriteFile(installer, []byte("#!/bin/sh\nprintf '%s\\n' \"$2\" >> \"$QS_INSTALL_SCOPE_LOG\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QS_INSTALL_SCOPE_LOG", log)
	original := appInstallJobs
	defer func() { appInstallJobs = original }()
	appInstallJobs = installjob.Manager{Root: filepath.Join(dir, "card-jobs"), Installer: installer}

	state, err := (appStore{}).InstallationState(context.Background(), "luci-app-example")
	if err != nil || state != "" {
		t.Fatalf("unrelated application acquired card state: %q %v", state, err)
	}
	if state, err = InstallApp("luci-app-example"); err != nil || state != "installing" {
		t.Fatal(state, err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		data, _ := os.ReadFile(log)
		if strings.Contains(string(data), "luci-app-example\n") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("legacy installation command was not invoked")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err = os.Stat(appInstallJobs.Root); !os.IsNotExist(err) {
		t.Fatalf("unrelated installation created a tracking directory: %v", err)
	}
	if state, err = InstallApp("app-meta-fastnet"); err != nil || state != "installing" {
		t.Fatal(state, err)
	}
	deadline = time.Now().Add(3 * time.Second)
	for {
		state, err = appInstallJobs.State("app-meta-fastnet")
		if err == nil && state == "completed" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("FastNet did not finish its tracked installation: %q %v", state, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	data, _ := os.ReadFile(log)
	if string(data) != "luci-app-example\napp-meta-fastnet\n" {
		t.Fatalf("unexpected or repeated installer calls: %q", data)
	}
}

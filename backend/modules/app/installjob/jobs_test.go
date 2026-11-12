package installjob

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestInstallSurvivesOwnerProcessExit(t *testing.T) {
	if os.Getenv("FN_INSTALL_HELPER") == "1" {
		m := Manager{Root: os.Getenv("FN_INSTALL_ROOT"), Installer: os.Getenv("FN_INSTALL_BIN")}
		if _, err := m.Start("app-meta-fastnet"); err != nil {
			os.Exit(2)
		}
		os.Exit(0)
	}
	root := t.TempDir()
	installer := filepath.Join(root, "installer")
	release := filepath.Join(root, "release")
	count := filepath.Join(root, "count")
	os.WriteFile(installer, []byte("#!/bin/sh\necho run >> "+count+"\nwhile [ ! -f "+release+" ]; do sleep .02; done\n"), 0700)
	m := Manager{Root: filepath.Join(root, "jobs"), Installer: installer}
	child := exec.Command(os.Args[0], "-test.run=^TestInstallSurvivesOwnerProcessExit$")
	child.Env = append(os.Environ(), "FN_INSTALL_HELPER=1", "FN_INSTALL_ROOT="+m.Root, "FN_INSTALL_BIN="+installer)
	if err := child.Run(); err != nil {
		t.Fatal(err)
	}
	if state, err := m.State("app-meta-fastnet"); err != nil || state != "installing" {
		t.Fatal(state, err)
	}
	if state, err := m.Start("app-meta-fastnet"); err != nil || state != "installing" {
		t.Fatal(state, err)
	}
	os.WriteFile(release, nil, 0600)
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		state, _ := m.State("app-meta-fastnet")
		if state == "completed" {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	if state, _ := m.State("app-meta-fastnet"); state != "completed" {
		t.Fatal(state)
	}
	raw, _ := os.ReadFile(count)
	if strings.Count(string(raw), "run") != 1 {
		t.Fatal(string(raw))
	}
}

func TestInstallLeaseAcrossClientsAndNewManager(t *testing.T) {
	root := t.TempDir()
	installer := filepath.Join(root, "installer")
	count := filepath.Join(root, "count")
	release := filepath.Join(root, "release")
	if e := os.WriteFile(installer, []byte("#!/bin/sh\necho run >> "+count+"\nwhile [ ! -f "+release+" ]; do sleep .02; done\n"), 0700); e != nil {
		t.Fatal(e)
	}
	m := Manager{Root: filepath.Join(root, "jobs"), Installer: installer}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state, e := m.Start("app-meta-fastnet")
			if e != nil || state != "installing" {
				t.Errorf("%s %v", state, e)
			}
		}()
	}
	wg.Wait()
	other := Manager{Root: m.Root, Installer: installer}
	if state, e := other.State("app-meta-fastnet"); e != nil || state != "installing" {
		t.Fatalf("new manager: %s %v", state, e)
	}
	os.WriteFile(release, nil, 0600)
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		state, _ := other.State("app-meta-fastnet")
		if state == "completed" {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	if state, _ := other.State("app-meta-fastnet"); state != "completed" {
		t.Fatal(state)
	}
	raw, e := os.ReadFile(count)
	if e != nil || strings.Count(string(raw), "run") != 1 {
		t.Fatalf("duplicate executions: %s %v", raw, e)
	}
}
func TestUncertainAndFailedInstallNeverBecomeIdle(t *testing.T) {
	m := Manager{Root: filepath.Join(t.TempDir(), "jobs"), Installer: "/no/such/installer"}
	state, e := m.Start("fastnet")
	if e != nil || state != "installing" {
		t.Fatal(state, e)
	}
	until := time.Now().Add(time.Second)
	for time.Now().Before(until) {
		state, _ = m.State("fastnet")
		if state == "failed" {
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	if state != "failed" {
		t.Fatal(state)
	}
	os.WriteFile(filepath.Join(m.Root, "fastnet.json"), []byte(`{"schemaVersion":1,"state":"running"}`), 0600)
	if state, _ = m.State("fastnet"); state != "unknown" {
		t.Fatal(state)
	}
	if _, e = m.Start("fastnet"); e == nil {
		t.Fatal("uncertain installation replayed")
	}
	if _, e = m.Start("../../bad"); e == nil {
		t.Fatal("path traversal")
	}
}

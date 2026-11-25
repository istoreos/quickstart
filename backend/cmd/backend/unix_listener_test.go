package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/server"
)

func socketTestDir(t *testing.T) string {
	t.Helper()
	// Unix socket names have a small kernel limit; the build cache's TMPDIR
	// plus a Go test name can exceed it.
	dir, err := os.MkdirTemp("/tmp", "qs-sock-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestUnixListenerRestart(t *testing.T) {
	socketPath := filepath.Join(socketTestDir(t), "run", "http.sock")
	for i := 0; i < 3; i++ {
		lis, err := listenUnix(socketPath)
		if err != nil {
			t.Fatal(err)
		}
		for name, mode := range map[string]os.FileMode{filepath.Dir(socketPath): 0750, socketPath: 0600} {
			info, err := os.Stat(name)
			if err != nil || info.Mode().Perm() != mode {
				t.Fatalf("%s: want mode %o, got %v, %v", name, mode, info, err)
			}
		}
		if other, err := listenUnix(socketPath); err == nil {
			other.Close()
			t.Fatal("replaced a live socket")
		}
		conn, err := net.DialTimeout("unix", socketPath, time.Second)
		if err != nil {
			t.Fatalf("original listener lost its socket: %v", err)
		}
		conn.Close()
		// Simulate the filesystem state left behind by SIGKILL.
		lis.SetUnlinkOnClose(false)
		lis.Close()
	}
	lis, err := listenUnix(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	lis.Close()
	if _, err := os.Lstat(socketPath); !os.IsNotExist(err) {
		t.Fatalf("closed listener left its socket: %v", err)
	}
}

func TestUnixListenerPreservesNonSockets(t *testing.T) {
	dir := socketTestDir(t)
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(file, link); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{file, link, ""} {
		if lis, err := listenUnix(name); err == nil {
			lis.Close()
			t.Fatalf("accepted non-socket path %q", name)
		}
	}
	if data, err := os.ReadFile(link); err != nil || string(data) != "keep" {
		t.Fatalf("changed an existing file or symlink: %q, %v", data, err)
	}
	// An internal socket failure must also tear down the HTTP listener.
	httpPath := filepath.Join(dir, "http.sock")
	if err := startMain(httpPath, file); err == nil {
		t.Fatal("ignored internal listener failure")
	}
	if _, err := os.Lstat(httpPath); !os.IsNotExist(err) {
		t.Fatalf("failed startup left the HTTP socket: %v", err)
	}
}

func TestUnixListenerPreservesExistingDirectoryMode(t *testing.T) {
	dir := socketTestDir(t)
	want := os.FileMode(0777) | os.ModeSticky
	if err := os.Chmod(dir, want); err != nil {
		t.Fatal(err)
	}
	checkMode := func() {
		t.Helper()
		info, err := os.Stat(dir)
		if err != nil || info.Mode()&(os.ModePerm|os.ModeSticky) != want {
			t.Fatalf("changed existing parent directory mode: %v, %v", info, err)
		}
	}
	socketPath := filepath.Join(dir, "http.sock")
	lis, err := listenUnix(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	lis.Close()
	checkMode()
	if err := os.WriteFile(socketPath, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if lis, err := listenUnix(socketPath); err == nil {
		lis.Close()
		t.Fatal("replaced non-socket")
	}
	checkMode()
}

func TestUnixHTTPKeepsAuthentication(t *testing.T) {
	dir := socketTestDir(t)
	// Exercise the assembled HTTP router with a deterministic rpcd response.
	if err := os.WriteFile(filepath.Join(dir, "ubus"), []byte("#!/bin/sh\nprintf '%s\\n' '{\"values\":{\"username\":\"root\",\"token\":\"secret\"}}'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	socketPath := filepath.Join(dir, "http.sock")
	lis, err := listenUnix(socketPath)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: server.RouterInit(nil)}
	defer srv.Close()
	go srv.Serve(lis)
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
	}}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	for _, token := range []string{"", "forged", "secret"} {
		// Internal routes must stay absent even for an authenticated user.
		req, _ := http.NewRequest("GET", "http://localhost/api/dhns/connect/", nil)
		req.Header.Set("X-Forwarded-Sid", strings.Repeat("1", 32))
		req.Header.Set("X-Forwarded-Token", token)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if token == "secret" {
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("internal route exposed on HTTP socket: %d %s", resp.StatusCode, body)
			}
		} else if !strings.Contains(string(body), `"success":-1001`) {
			t.Fatalf("invalid token reached router: %d %s", resp.StatusCode, body)
		}
	}
}

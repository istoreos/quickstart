// Package installjob deduplicates package installation across tabs and host restarts.
package installjob

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
)

type Manager struct{ Root, Installer string }

var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.+-]{0,127}$`)

type record struct {
	SchemaVersion int    `json:"schemaVersion"`
	State         string `json:"state"`
	ExitCode      int    `json:"exitCode"`
	Attempt       int    `json:"attempt"`
}

func (m Manager) lock(name string) (*os.File, error) {
	if !validName.MatchString(name) {
		return nil, errors.New("invalid package name")
	}
	if err := os.MkdirAll(m.Root, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(m.Root)
	if err != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("invalid installation directory")
	}
	fd, err := syscall.Open(filepath.Join(m.Root, name+".lock"), syscall.O_RDWR|syscall.O_CREAT|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}
func (m Manager) read(name string) string {
	f, err := os.Open(filepath.Join(m.Root, name+".json"))
	if os.IsNotExist(err) {
		return "idle"
	}
	if err != nil {
		return "unknown"
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 1025))
	var r record
	if err != nil || len(raw) > 1024 || json.Unmarshal(raw, &r) != nil || r.SchemaVersion != 1 || r.State != "done" {
		return "unknown"
	}
	if r.ExitCode == 0 {
		return "completed"
	}
	return "failed"
}
func (m Manager) State(name string) (string, error) {
	f, err := m.lock(name)
	if err != nil {
		return "unknown", err
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == syscall.EWOULDBLOCK {
		return "installing", nil
	} else if err != nil {
		return "unknown", err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
	return m.read(name), nil
}
func (m Manager) Start(name string) (string, error) {
	f, err := m.lock(name)
	if err != nil {
		return "unknown", err
	}
	defer f.Close()
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err == syscall.EWOULDBLOCK {
		return "installing", nil
	} else if err != nil {
		return "unknown", err
	}
	if m.read(name) == "unknown" {
		return "unknown", errors.New("installation status unknown; inspect software center")
	}
	path := filepath.Join(m.Root, name+".json")
	var previous record
	old, _ := os.ReadFile(path)
	if len(old) <= 1024 {
		_ = json.Unmarshal(old, &previous)
	}
	attempt := previous.Attempt + 1
	if attempt < 1 {
		attempt = 1
	}
	raw, _ := json.Marshal(record{SchemaVersion: 1, State: "running", Attempt: attempt})
	if err = os.WriteFile(path, raw, 0600); err != nil {
		return "unknown", err
	}
	installer := m.Installer
	if installer == "" {
		installer = "is-opkg"
	}
	// Descriptor 3 keeps the per-package lock in the child after Quickstart exits.
	// The child commits its outcome itself. An interrupted outcome remains unknown.
	cmd := exec.Command("sh", "-c", `"$1" install "$2" >/dev/null 2>&1
code=$?
umask 077
printf '{"schemaVersion":1,"state":"done","exitCode":%s,"attempt":%s}' "$code" "$4" > "$3.new.$$" && mv -f "$3.new.$$" "$3"
exit "$code"`, "quickstart-install", installer, name, path, strconv.Itoa(attempt))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.ExtraFiles = []*os.File{f}
	if err = cmd.Start(); err != nil {
		_ = os.WriteFile(path, []byte(`{"schemaVersion":1,"state":"done","exitCode":`+strconv.Itoa(127)+`}`), 0600)
		return "failed", err
	}
	go func() { _ = cmd.Wait() }()
	// Close only this descriptor. LOCK_UN would also release the child's lease.
	return "installing", nil
}

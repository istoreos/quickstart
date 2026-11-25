package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func listenUnix(socketPath string) (*net.UnixListener, error) {
	if socketPath == "" {
		return nil, errors.New("Unix socket path is required")
	}
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, err
	}
	if info, err := os.Lstat(socketPath); err == nil {
		if info.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("refusing to replace non-socket %s", socketPath)
		}
		conn, err := net.DialTimeout("unix", socketPath, time.Second)
		if err == nil {
			conn.Close()
			return nil, fmt.Errorf("Unix socket already in use: %s", socketPath)
		}
		// A killed process can leave its socket behind. Only remove it when
		// the kernel confirms there is no listener; other errors fail closed.
		if !errors.Is(err, syscall.ECONNREFUSED) && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	lis, err := net.ListenUnix("unix", &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(socketPath, 0600); err != nil {
		lis.Close()
		return nil, err
	}
	return lis, nil
}

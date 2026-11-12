package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

type networkWriteLeaseKey struct{}
type lanScopeResolverKey struct{}

var errNetworkConfigRecoveryRequired = errors.New("network configuration recovery required")

type networkRecoveryError struct{ error }

func (err *networkRecoveryError) Is(target error) bool {
	return target == errNetworkConfigRecoveryRequired
}
func (err *networkRecoveryError) Unwrap() error { return err.error }

type networkWriteLease struct {
	path   string
	active atomic.Bool
}

// The lock belongs to the shared configuration directory, not to a LAN or a
// business module. A synchronous nested write reuses its caller's lease.
func lockNetworkConfiguration(ctx context.Context, configDir string) (context.Context, func(), error) {
	canonical, err := filepath.Abs(configDir)
	if err != nil {
		return nil, nil, err
	}
	if resolved, err := filepath.EvalSymlinks(canonical); err == nil {
		canonical = resolved
	}
	sum := sha256.Sum256([]byte(canonical))
	path := filepath.Join("/var/lock", "quickstart-network-"+strconv.Itoa(os.Getuid())+"-"+hex.EncodeToString(sum[:8])+".lock")
	if lease, _ := ctx.Value(networkWriteLeaseKey{}).(*networkWriteLease); lease != nil && lease.path == path && lease.active.Load() {
		return ctx, func() {}, nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, nil, fmt.Errorf("open network configuration lock: %w", err)
	}
	deadline, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			file.Close()
			return nil, nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-deadline.Done():
			timer.Stop()
			file.Close()
			return nil, nil, deadline.Err()
		case <-timer.C:
		}
	}
	lease := &networkWriteLease{path: path}
	lease.active.Store(true)
	var once sync.Once
	release := func() {
		once.Do(func() { lease.active.Store(false); _ = unix.Flock(int(file.Fd()), unix.LOCK_UN); _ = file.Close() })
	}
	return context.WithValue(ctx, networkWriteLeaseKey{}, lease), release, nil
}

func lockLANWrite(ctx context.Context) (context.Context, func(), error) {
	locked, release, err := lockNetworkConfiguration(ctx, deviceRestrictionConfigDir())
	if err != nil {
		return nil, nil, err
	}
	if resolver, _ := ctx.Value(lanScopeResolverKey{}).(*LANScopeModule); resolver != nil {
		if err := resolver.requireWritable(ctx); err != nil {
			release()
			return nil, nil, err
		}
	}
	return locked, release, nil
}

func bindLANVersion(ctx context.Context, version string) string {
	scope := selectedLANScope(ctx)
	if scope == nil {
		return version
	}
	sum := sha256.Sum256([]byte(scope.NetworkID + "\x00" + scope.Version + "\x00" + version))
	return hex.EncodeToString(sum[:])
}

func taskLANRecordKey(network, task, key string) string {
	if network == "" || network == defaultLAN {
		return task + "\x00" + key
	}
	return network + "\x00" + task + "\x00" + key
}

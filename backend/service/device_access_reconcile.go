package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/digineo/go-uci"
)

func accessBindingsNeedCommit(ctx context.Context) (bool, error) {
	dir := deviceRestrictionConfigDir()
	if _, err := os.Stat(filepath.Join(dir, "firewall")); os.IsNotExist(err) {
		return false, nil
	}
	tree := uci.NewTree(dir)
	if err := tree.LoadConfig("firewall", true); err != nil {
		return false, err
	}
	return bindDeviceAccessRules(ctx, tree)
}

// The same compiler serves startup, network rebinding and quota recovery.
// An unchanged binding performs no write or service reload.
func reconcileDeviceAccessBindings(ctx context.Context) error {
	ctx, release, err := lockNetworkConfiguration(ctx, deviceRestrictionConfigDir())
	if err != nil {
		return err
	}
	defer release()
	changed, err := accessBindingsNeedCommit(ctx)
	if err != nil || !changed {
		return err
	}
	store := &systemDevicePolicyStore{}
	backup, err := store.Backup(ctx, "access")
	if err != nil {
		return err
	}
	if err = writeDeviceAccessPolicyBatchAt(ctx, deviceRestrictionConfigDir(), nil); err == nil {
		err = deviceAccessRestrictionApply(ctx)
	}
	if err != nil {
		if restoreErr := store.Restore(ctx, backup); restoreErr != nil {
			return &networkRecoveryError{fmt.Errorf("access binding recovery_required: %v; %v", err, restoreErr)}
		}
	}
	return err
}

// Called by serve, not by constructors or HTTP reads. One worker keeps saved
// pauses scoped even when there is no quota, group, browser or access change.
func (backend *ServiceBackend) StartLANPolicyReconciliation(ctx context.Context) {
	backend.accessBindingsOnce.Do(func() {
		resolver := backend.lanScopeModule()
		lastError := ""
		run := func() {
			cycle, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := reconcileDeviceAccessBindings(context.WithValue(cycle, lanScopeResolverKey{}, resolver))
			cancel()
			message := ""
			if err != nil {
				message = err.Error()
			}
			if message != "" && message != lastError {
				l.Warnln("restore LAN access bindings:", message)
			}
			lastError = message
		}
		run()
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					run()
				}
			}
		}()
	})
}

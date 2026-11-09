package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/istoreos/quickstart/backend/models"
	"sort"
	"strings"
	"time"
)

// One owner replacement gives a LAN batch one revision and one verification.
// Other owners and quota fields are retained; rollback is pinned to this binding.
func applyNativeSpeedBatch(ctx context.Context, module *RateLimitModule, native *nativeRateLimitProvider, changes []groupBatchRestrictionChange) (func(context.Context) error, error) {
	selected, err := native.forContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := selected.probe(ctx); err != nil {
		return nil, err
	}
	before, err := selected.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	original := selected.ownedInputs(before)
	byMAC := map[string]nativePolicyInput{}
	for _, input := range original {
		byMAC[normalizeInventoryMAC(input.Device.Value)] = input
	}
	for _, change := range changes {
		if change.current == nil || change.request == nil || change.request.Speed == nil {
			return nil, errors.New("invalid native speed batch")
		}
		mac := normalizeInventoryMAC(change.current.MAC)
		if mac == "" {
			return nil, errors.New("native batch requires a stable MAC")
		}
		desired := change.request.Speed
		if desired.Enabled {
			if desired.UploadSpeed <= 0 || desired.DownloadSpeed <= 0 {
				return nil, errors.New("invalid native batch rate")
			}
			if module != nil && module.routes != nil {
				route, err := module.routes.Resolve(ctx, mac)
				if err != nil {
					return nil, err
				}
				if !route.Local {
					return nil, &PolicyError{Code: "enforcement_path_unverified", Message: rateLimitReasonMessage("enforcement_path_unverified")}
				}
			}
		}
		input, exists := byMAC[mac]
		if !exists {
			input.Device = nativeDeviceIdentity{Kind: "mac", Value: strings.ToUpper(mac)}
		}
		if desired.Enabled {
			input.RateLimit = desiredNativeRate(*desired)
			byMAC[mac] = input
		} else if len(input.Quota) > 0 && string(input.Quota) != "null" {
			input.RateLimit = nil
			byMAC[mac] = input
		} else {
			delete(byMAC, mac)
		}
	}
	keys := make([]string, 0, len(byMAC))
	for mac := range byMAC {
		keys = append(keys, mac)
	}
	sort.Strings(keys)
	inputs := make([]nativePolicyInput, 0, len(keys))
	for _, mac := range keys {
		inputs = append(inputs, byMAC[mac])
	}
	rollback := func(recovery context.Context) error {
		current, err := selected.snapshot(recovery)
		if err != nil {
			return err
		}
		_, err = selected.replaceOwned(recovery, current, original, fmt.Sprintf("group-restore-%d", time.Now().UnixNano()))
		return err
	}
	if _, err := selected.replaceOwned(ctx, before, inputs, fmt.Sprintf("group-%d", time.Now().UnixNano())); err != nil {
		if restoreErr := rollback(ctx); restoreErr != nil {
			return rollback, &networkRecoveryError{fmt.Errorf("recovery_required: %v; %w", err, restoreErr)}
		}
		return rollback, err
	}
	return rollback, nil
}

func nativeRuleRemoval(mac string) groupBatchRestrictionChange {
	return groupBatchRestrictionChange{request: &models.DevicePolicyApplyRequest{Kind: "speed", Speed: &models.DeviceSpeedPolicy{}}, current: &models.DevicePolicy{MAC: mac}}
}

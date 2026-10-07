package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/istoreos/quickstart/backend/models"
)

type DeviceProfileModule struct {
	mu           sync.Mutex
	inventory    *DeviceInventoryModule
	transactions *TaskTransactionJournal
}

func NewDeviceProfileModule(inventory *DeviceInventoryModule) *DeviceProfileModule {
	return &DeviceProfileModule{inventory: inventory, transactions: newMemoryTaskTransactionJournal()}
}

func (module *DeviceProfileModule) Get(ctx context.Context, deviceID string) (*models.DeviceProfileResponse, error) {
	device, err := module.findDevice(ctx, deviceID)
	if err != nil {
		return deviceProfileFailure("not_found", err.Error()), nil
	}
	record, err := module.profileRecord(device)
	if err != nil {
		return deviceProfileFailure("read_failed", err.Error()), nil
	}
	return deviceProfileSuccess(device, record, false), nil
}

func (module *DeviceProfileModule) Apply(ctx context.Context, request *models.DeviceProfileApplyRequest) (*models.DeviceProfileResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	if request == nil || request.DeviceID == "" {
		return attachDeviceProfileTransaction(deviceProfileFailure("validation_failed", "deviceId is required"), rejectedTaskTransaction("profile", "")), nil
	}
	device, err := module.findDevice(ctx, request.DeviceID)
	if err != nil {
		return deviceProfileFailure("not_found", err.Error()), nil
	}
	if module.inventory.classificationOverrides == nil {
		return deviceProfileFailure("write_failed", "device profile store is unavailable"), nil
	}
	if err := validateDeviceProfileMutation(request); err != nil {
		return attachDeviceProfileTransaction(deviceProfileFailure("validation_failed", err.Error()), rejectedTaskTransaction("profile", request.IdempotencyKey)), nil
	}
	fingerprint, _ := json.Marshal(request)
	begin, err := module.transactions.Begin(ctx, "profile", request.IdempotencyKey, string(fingerprint))
	if err != nil {
		return deviceProfileFailure("transaction_unavailable", "could not create transaction record"), nil
	}
	transaction := begin.Transaction
	if begin.Conflict {
		return attachDeviceProfileTransaction(deviceProfileFailure("conflict", "idempotency key was already used for a different change"), transaction), nil
	}
	if begin.Replay {
		if transaction.Status != "committed" && transaction.Status != "unchanged" {
			return attachDeviceProfileTransaction(deviceProfileFailure(transaction.Status, "previous transaction requires review"), transaction), nil
		}
		record, readErr := module.profileRecord(device)
		if readErr != nil {
			return attachDeviceProfileTransaction(deviceProfileFailure("read_failed", readErr.Error()), transaction), nil
		}
		return attachDeviceProfileTransaction(deviceProfileSuccess(device, record, false), transaction), nil
	}
	_ = module.transactions.Advance(ctx, transaction, "apply", "in_progress", "retry_profile_save")
	scope := deviceProfileScope(device)
	var changed bool
	switch request.Action {
	case "patch":
		if request.Alias == nil && request.Brand == nil && request.Category == nil && request.IconMode == nil && request.IconKey == nil {
			return deviceProfileFailure("validation_failed", "at least one profile field is required"), nil
		}
		changed, err = module.inventory.classificationOverrides.PatchProfileFields(request.DeviceID, scope, deviceProfilePatch{
			Alias: request.Alias, Brand: request.Brand, Category: request.Category, IconMode: request.IconMode, IconKey: request.IconKey,
		})
	case "reset":
		changed, err = module.inventory.classificationOverrides.ResetProfile(request.DeviceID, scope)
	case "reset_brand":
		changed, err = module.inventory.classificationOverrides.ResetProfileField(request.DeviceID, scope, "brand")
	case "reset_category":
		changed, err = module.inventory.classificationOverrides.ResetProfileField(request.DeviceID, scope, "category")
	case "reset_icon":
		changed, err = module.inventory.classificationOverrides.ResetProfileField(request.DeviceID, scope, "icon")
	default:
		return deviceProfileFailure("validation_failed", "unsupported device profile action"), nil
	}
	if err != nil {
		_ = module.transactions.Advance(ctx, transaction, "rollback", "rolled_back", "retry")
		return attachDeviceProfileTransaction(deviceProfileFailure("rolled_back", "device profile was not changed"), transaction), nil
	}
	_ = module.transactions.Advance(ctx, transaction, "verify", "in_progress", "retry_profile_read")
	module.inventory.Invalidate()
	device, err = module.findDevice(ctx, request.DeviceID)
	if err != nil {
		_ = module.transactions.Advance(ctx, transaction, "recover", "recovery_required", "retry_profile_read")
		return attachDeviceProfileTransaction(deviceProfileFailure("recovery_required", "profile saved but verification failed"), transaction), nil
	}
	record, err := module.profileRecord(device)
	if err != nil {
		_ = module.transactions.Advance(ctx, transaction, "recover", "recovery_required", "retry_profile_read")
		return attachDeviceProfileTransaction(deviceProfileFailure("recovery_required", "profile saved but verification failed"), transaction), nil
	}
	status := "committed"
	if !changed {
		status = "unchanged"
	}
	_ = module.transactions.Advance(ctx, transaction, "verify", status, "")
	return attachDeviceProfileTransaction(deviceProfileSuccess(device, record, changed), transaction), nil
}

func validateDeviceProfileMutation(request *models.DeviceProfileApplyRequest) error {
	switch request.Action {
	case "patch":
		if request.Alias == nil && request.Brand == nil && request.Category == nil && request.IconMode == nil && request.IconKey == nil {
			return errors.New("at least one profile field is required")
		}
		if request.Alias != nil {
			if _, err := normalizeDeviceAlias(*request.Alias); err != nil {
				return err
			}
		}
		if request.Brand != nil {
			if _, err := normalizeDeviceBrand(*request.Brand); err != nil {
				return err
			}
		}
		if request.Category != nil && *request.Category != "" && !validDeviceCategory(*request.Category) {
			return errors.New("invalid classification override")
		}
		if request.IconMode != nil && *request.IconMode != "auto" && *request.IconMode != "manual" {
			return errors.New("icon mode must be auto or manual")
		}
		if request.IconKey != nil && *request.IconKey != "" && !validDeviceIconKey(*request.IconKey) {
			return errors.New("invalid device icon key")
		}
		if request.IconMode != nil && *request.IconMode == "manual" && (request.IconKey == nil || *request.IconKey == "") {
			return errors.New("manual icon mode requires an icon key")
		}
	case "reset", "reset_brand", "reset_category", "reset_icon":
	default:
		return errors.New("unsupported device profile action")
	}
	return nil
}

func (module *DeviceProfileModule) profileRecord(device *models.DeviceInventoryItem) (deviceProfileRecord, error) {
	if module == nil || module.inventory == nil || module.inventory.classificationOverrides == nil {
		return deviceProfileRecord{}, errors.New("device profile store is unavailable")
	}
	record, _, err := module.inventory.classificationOverrides.GetProfile(device.DeviceID, deviceProfileScope(device))
	return record, err
}

func (module *DeviceProfileModule) findDevice(ctx context.Context, deviceID string) (*models.DeviceInventoryItem, error) {
	if module == nil || module.inventory == nil || deviceID == "" {
		return nil, errors.New("device was not found")
	}
	response, err := module.inventory.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	if response == nil || response.Result == nil {
		return nil, errors.New("device inventory is unavailable")
	}
	for _, device := range response.Result.Devices {
		if device != nil && device.DeviceID == deviceID {
			return device, nil
		}
	}
	return nil, errors.New("device was not found")
}

func deviceProfileScope(device *models.DeviceInventoryItem) string {
	if device != nil && device.Identity != nil && device.Identity.Scope == "boot" {
		return "boot"
	}
	return "persistent"
}

func deviceProfileSuccess(device *models.DeviceInventoryItem, record deviceProfileRecord, changed bool) *models.DeviceProfileResponse {
	alias := ""
	if device.DisplayName != device.Hostname {
		alias = device.DisplayName
	}
	return &models.DeviceProfileResponse{Result: &models.DeviceProfileResult{
		Changed: changed,
		Profile: &models.DeviceProfile{
			DeviceID: device.DeviceID, Alias: alias, OriginalHostname: device.Hostname,
			Scope: deviceProfileScope(device), Classification: device.Classification,
			ManualBrand: record.Brand, ManualCategory: record.Category,
			Icon: deviceIconForProfile(device.Classification, record),
		},
	}}
}

func deviceProfileFailure(code, message string) *models.DeviceProfileResponse {
	return &models.DeviceProfileResponse{Result: &models.DeviceProfileResult{
		Error: &models.DevicePolicyError{Code: code, Message: message},
	}}
}

func attachDeviceProfileTransaction(response *models.DeviceProfileResponse, transaction *models.TaskTransaction) *models.DeviceProfileResponse {
	if response != nil && response.Result != nil {
		response.Result.Transaction = transaction
	}
	return response
}

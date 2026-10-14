package service

import (
	"context"
	"errors"

	"github.com/istoreos/quickstart/backend/models"
)

type DeviceClassificationModule struct {
	inventory *DeviceInventoryModule
}

func NewDeviceClassificationModule(inventory *DeviceInventoryModule) *DeviceClassificationModule {
	return &DeviceClassificationModule{inventory: inventory}
}

func (module *DeviceClassificationModule) Get(ctx context.Context, deviceID string) (*models.DeviceClassificationResponse, error) {
	device, err := module.findDevice(ctx, deviceID)
	if err != nil {
		return classificationFailure(deviceID, "not_found", err.Error()), nil
	}
	return classificationSuccess(device, false), nil
}

func (module *DeviceClassificationModule) Apply(ctx context.Context, request *models.DeviceClassificationApplyRequest) (*models.DeviceClassificationResponse, error) {
	if request == nil || request.DeviceID == "" {
		return classificationFailure("", "validation_failed", "deviceId is required"), nil
	}
	device, err := module.findDevice(ctx, request.DeviceID)
	if err != nil {
		return classificationFailure(request.DeviceID, "not_found", err.Error()), nil
	}
	if module.inventory.classificationOverrides == nil {
		return classificationFailure(request.DeviceID, "write_failed", "classification override store is unavailable"), nil
	}
	scope := "persistent"
	if device.Identity != nil && device.Identity.Scope == "boot" {
		scope = "boot"
	}
	var changed bool
	switch request.Action {
	case "set":
		if !validDeviceCategory(request.Category) {
			return classificationFailure(request.DeviceID, "validation_failed", "invalid device category"), nil
		}
		changed, err = module.inventory.classificationOverrides.Set(request.DeviceID, scope, request.Category)
	case "reset":
		changed, err = module.inventory.classificationOverrides.Reset(request.DeviceID, scope)
	default:
		return classificationFailure(request.DeviceID, "validation_failed", "action must be set or reset"), nil
	}
	if err != nil {
		return classificationFailure(request.DeviceID, "write_failed", err.Error()), nil
	}
	module.inventory.Invalidate()
	device, err = module.findDevice(ctx, request.DeviceID)
	if err != nil {
		return classificationFailure(request.DeviceID, "not_found", err.Error()), nil
	}
	return classificationSuccess(device, changed), nil
}

func (module *DeviceClassificationModule) findDevice(ctx context.Context, deviceID string) (*models.DeviceInventoryItem, error) {
	if module == nil || module.inventory == nil || deviceID == "" {
		return nil, errors.New("device was not found")
	}
	response, err := module.inventory.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	for _, device := range response.Result.Devices {
		if device.DeviceID == deviceID {
			return device, nil
		}
	}
	return nil, errors.New("device was not found")
}

func classificationSuccess(device *models.DeviceInventoryItem, changed bool) *models.DeviceClassificationResponse {
	scope := "persistent"
	if device.Identity != nil && device.Identity.Scope == "boot" {
		scope = "boot"
	}
	return &models.DeviceClassificationResponse{Result: &models.DeviceClassificationResult{
		DeviceID: device.DeviceID, Classification: device.Classification, Changed: changed,
		Override: &models.DeviceClassificationOverride{Active: device.Classification != nil && device.Classification.Source == "manual", Category: device.Classification.Category, Scope: scope},
	}}
}

func classificationFailure(deviceID, code, message string) *models.DeviceClassificationResponse {
	return &models.DeviceClassificationResponse{Result: &models.DeviceClassificationResult{
		DeviceID: deviceID, Error: &models.DevicePolicyError{Code: code, Message: message},
	}}
}

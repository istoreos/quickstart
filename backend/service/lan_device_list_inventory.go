package service

import (
	"context"

	"github.com/linkease/quick-start/istore-backend/models"
)

type DeviceInventoryReader interface {
	ReadInventory(ctx context.Context) (models.LANDevices, error)
}

type defaultDeviceInventoryReader struct{}

var _ DeviceInventoryReader = (*defaultDeviceInventoryReader)(nil)

func NewDefaultDeviceInventoryReader() DeviceInventoryReader {
	return &defaultDeviceInventoryReader{}
}

func (reader *defaultDeviceInventoryReader) ReadInventory(ctx context.Context) (models.LANDevices, error) {
	resp, err := NetworkDeviceList(ctx)
	if err != nil {
		return nil, err
	}

	devices := models.LANDevices{}
	if resp == nil || resp.Result == nil {
		return devices, nil
	}

	devices = make(models.LANDevices, 0, len(resp.Result.Devices))
	for _, item := range resp.Result.Devices {
		if item == nil {
			continue
		}
		device, ok := buildDeviceInventoryItem(item.Ipv4addr, item.Mac)
		if !ok {
			continue
		}
		devices = append(devices, device)
	}

	return devices, nil
}

package service

import (
	"context"
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
	systemruntime "github.com/linkease/quick-start/istore-backend/modules/system/runtime"
	"github.com/shirou/gopsutil/v3/cpu"
)

type systemRuntimeFacade interface {
	Time(ctx context.Context) (*models.SystemTimeResponseResult, error)
	CPU(ctx context.Context) (*models.SystemCPUStatusResponseResult, error)
	Memory(ctx context.Context) (*models.SystemMemeryStatusResponseResult, error)
	Status(ctx context.Context) (*models.SystemStatusResponseResult, error)
}

var newSystemRuntimeService = func() systemRuntimeFacade {
	return systemruntime.NewService(defaultSystemRuntimeStore{})
}

type defaultSystemRuntimeStore struct{}

func (store defaultSystemRuntimeStore) ReadTime(ctx context.Context) (systemruntime.TimeSnapshot, error) {
	o, err := System(ctx)
	if err != nil {
		return systemruntime.TimeSnapshot{}, err
	}
	localtime, err := o.Get("localtime").Int64()
	if err != nil {
		return systemruntime.TimeSnapshot{}, err
	}
	uptime, err := o.Get("uptime").Int64()
	if err != nil {
		return systemruntime.TimeSnapshot{}, err
	}
	return systemruntime.TimeSnapshot{Localtime: localtime, Uptime: uptime}, nil
}

func (store defaultSystemRuntimeStore) ReadMemory(ctx context.Context) (systemruntime.MemorySnapshot, error) {
	o, err := System(ctx)
	if err != nil {
		return systemruntime.MemorySnapshot{}, err
	}
	total, err := o.Get("memory").Get("total").Int64()
	if err != nil {
		return systemruntime.MemorySnapshot{}, err
	}

	snapshot := systemruntime.MemorySnapshot{Total: total}
	data, ok := o.Get("memory").CheckGet("available")
	if ok {
		available, err := data.Int64()
		if err != nil {
			return systemruntime.MemorySnapshot{}, err
		}
		snapshot.Available = available
		snapshot.HasAvailable = true
		return snapshot, nil
	}

	free, err := o.Get("memory").Get("free").Int64()
	if err != nil {
		return systemruntime.MemorySnapshot{}, err
	}
	buffered, err := o.Get("memory").Get("buffered").Int64()
	if err != nil {
		return systemruntime.MemorySnapshot{}, err
	}
	cached, err := o.Get("memory").Get("cached").Int64()
	if err != nil {
		return systemruntime.MemorySnapshot{}, err
	}
	snapshot.Free = free
	snapshot.Buffered = buffered
	snapshot.Cached = cached
	return snapshot, nil
}

func (store defaultSystemRuntimeStore) ReadCPUUsage(ctx context.Context) (float64, error) {
	percent, err := cpu.Percent(time.Second, true)
	if err != nil {
		return 0, err
	}
	var usage float64
	for _, item := range percent {
		usage += item
	}
	return usage / float64(len(percent)), nil
}

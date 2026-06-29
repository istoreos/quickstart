package runtime

import (
	"context"
	"math"
	"strconv"
	"time"

	"github.com/linkease/quick-start/istore-backend/models"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type TimeSnapshot struct {
	Localtime int64
	Uptime    int64
}

type MemorySnapshot struct {
	Total        int64
	Available    int64
	HasAvailable bool
	Free         int64
	Buffered     int64
	Cached       int64
}

type Store interface {
	ReadTime(ctx context.Context) (TimeSnapshot, error)
	ReadMemory(ctx context.Context) (MemorySnapshot, error)
	ReadCPUUsage(ctx context.Context) (float64, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (svc *Service) Time(ctx context.Context) (*models.SystemTimeResponseResult, error) {
	snapshot, err := svc.store.ReadTime(ctx)
	if err != nil {
		return nil, err
	}
	return BuildTimeStatus(snapshot.Localtime, snapshot.Uptime), nil
}

func (svc *Service) Memory(ctx context.Context) (*models.SystemMemeryStatusResponseResult, error) {
	snapshot, err := svc.store.ReadMemory(ctx)
	if err != nil {
		return nil, err
	}
	available := snapshot.Available
	if !snapshot.HasAvailable {
		available = snapshot.Free + snapshot.Buffered + snapshot.Cached
	}
	return BuildMemoryStatus(snapshot.Total, available), nil
}

func (svc *Service) CPU(ctx context.Context) (*models.SystemCPUStatusResponseResult, error) {
	usage, err := svc.store.ReadCPUUsage(ctx)
	if err != nil {
		return nil, err
	}
	return &models.SystemCPUStatusResponseResult{Usage: int64(usage)}, nil
}

func (svc *Service) Status(ctx context.Context) (*models.SystemStatusResponseResult, error) {
	cpuStatus, err := svc.CPU(ctx)
	if err != nil {
		return nil, err
	}
	memoryStatus, err := svc.Memory(ctx)
	if err != nil {
		return nil, err
	}
	timeStatus, err := svc.Time(ctx)
	if err != nil {
		return nil, err
	}
	return BuildStatus(cpuStatus.Usage, timeStatus, memoryStatus), nil
}

func BuildTimeStatus(localtime int64, uptime int64) *models.SystemTimeResponseResult {
	return &models.SystemTimeResponseResult{
		Localtime:   time.Unix(localtime, 0).UTC().Format(utils.ChineseTimeLayout),
		Uptime:      uptime,
		UptimeHuman: utils.SecondsToHuman(uptime),
	}
}

func BuildMemoryStatus(total int64, available int64) *models.SystemMemeryStatusResponseResult {
	return &models.SystemMemeryStatusResponseResult{
		Available:           strconv.FormatInt(available/(1024*1024), 10) + "MB",
		Total:               strconv.FormatInt(total/(1024*1024), 10) + "MB",
		AvailablePercentage: int64(math.Round(100 * float64(available) / float64(total))),
	}
}

func BuildStatus(cpuUsage int64, timeStatus *models.SystemTimeResponseResult, memoryStatus *models.SystemMemeryStatusResponseResult) *models.SystemStatusResponseResult {
	return &models.SystemStatusResponseResult{
		CPUUsage:               cpuUsage,
		MemTotal:               memoryStatus.Total,
		MemAvailable:           memoryStatus.Available,
		MemAvailablePercentage: memoryStatus.AvailablePercentage,
		Localtime:              timeStatus.Localtime,
		Uptime:                 timeStatus.Uptime,
		UptimeHuman:            timeStatus.UptimeHuman,
	}
}

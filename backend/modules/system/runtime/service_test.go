package runtime

import (
	"context"
	"errors"
	"testing"
)

type fakeStore struct {
	timeSnapshot   TimeSnapshot
	timeErr        error
	memorySnapshot MemorySnapshot
	memoryErr      error
	cpuUsage       float64
	cpuErr         error
}

func (store fakeStore) ReadTime(ctx context.Context) (TimeSnapshot, error) {
	return store.timeSnapshot, store.timeErr
}

func (store fakeStore) ReadMemory(ctx context.Context) (MemorySnapshot, error) {
	return store.memorySnapshot, store.memoryErr
}

func (store fakeStore) ReadCPUUsage(ctx context.Context) (float64, error) {
	return store.cpuUsage, store.cpuErr
}

func TestBuildTimeStatusFormatsLocaltimeAndUptime(t *testing.T) {
	t.Parallel()

	result := BuildTimeStatus(1704067200, 3661)

	if result.Localtime != "2024-01-01 00:00:00" {
		t.Fatalf("Localtime = %q, want 2024-01-01 00:00:00", result.Localtime)
	}
	if result.Uptime != 3661 {
		t.Fatalf("Uptime = %d, want 3661", result.Uptime)
	}
	if result.UptimeHuman != "1h 1m 1s " {
		t.Fatalf("UptimeHuman = %q, want 1h 1m 1s ", result.UptimeHuman)
	}
}

func TestBuildMemoryStatusUsesAvailableMemory(t *testing.T) {
	t.Parallel()

	result := BuildMemoryStatus(512*1024*1024, 128*1024*1024)

	if result.Total != "512MB" {
		t.Fatalf("Total = %q, want 512MB", result.Total)
	}
	if result.Available != "128MB" {
		t.Fatalf("Available = %q, want 128MB", result.Available)
	}
	if result.AvailablePercentage != 25 {
		t.Fatalf("AvailablePercentage = %d, want 25", result.AvailablePercentage)
	}
}

func TestBuildMemoryStatusRoundsAvailablePercentage(t *testing.T) {
	t.Parallel()

	result := BuildMemoryStatus(3*1024*1024, 2*1024*1024)

	if result.AvailablePercentage != 67 {
		t.Fatalf("AvailablePercentage = %d, want 67", result.AvailablePercentage)
	}
}

func TestBuildStatusCombinesCPUTimeAndMemory(t *testing.T) {
	t.Parallel()

	timeStatus := BuildTimeStatus(1704067200, 42)
	memoryStatus := BuildMemoryStatus(512*1024*1024, 128*1024*1024)

	result := BuildStatus(55, timeStatus, memoryStatus)

	if result.CPUUsage != 55 {
		t.Fatalf("CPUUsage = %d, want 55", result.CPUUsage)
	}
	if result.Localtime != timeStatus.Localtime || result.Uptime != 42 || result.UptimeHuman != timeStatus.UptimeHuman {
		t.Fatalf("time fields not copied: %#v", result)
	}
	if result.MemTotal != "512MB" || result.MemAvailable != "128MB" || result.MemAvailablePercentage != 25 {
		t.Fatalf("memory fields not copied: %#v", result)
	}
}

func TestServiceTimeBuildsTimeStatus(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeStore{timeSnapshot: TimeSnapshot{Localtime: 1704067200, Uptime: 3661}})

	result, err := svc.Time(context.Background())
	if err != nil {
		t.Fatalf("Time returned error: %v", err)
	}
	if result.Localtime != "2024-01-01 00:00:00" || result.UptimeHuman != "1h 1m 1s " {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceMemoryUsesAvailableWhenPresent(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeStore{memorySnapshot: MemorySnapshot{
		Total:        512 * 1024 * 1024,
		Available:    128 * 1024 * 1024,
		HasAvailable: true,
	}})

	result, err := svc.Memory(context.Background())
	if err != nil {
		t.Fatalf("Memory returned error: %v", err)
	}
	if result.Total != "512MB" || result.Available != "128MB" || result.AvailablePercentage != 25 {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceMemoryFallsBackToFreeBufferedCached(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeStore{memorySnapshot: MemorySnapshot{
		Total:    512 * 1024 * 1024,
		Free:     64 * 1024 * 1024,
		Buffered: 32 * 1024 * 1024,
		Cached:   32 * 1024 * 1024,
	}})

	result, err := svc.Memory(context.Background())
	if err != nil {
		t.Fatalf("Memory returned error: %v", err)
	}
	if result.Available != "128MB" || result.AvailablePercentage != 25 {
		t.Fatalf("result = %#v", result)
	}
}

func TestServiceCPUStatusCastsUsageToInt(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeStore{cpuUsage: 42.9})

	result, err := svc.CPU(context.Background())
	if err != nil {
		t.Fatalf("CPU returned error: %v", err)
	}
	if result.Usage != 42 {
		t.Fatalf("Usage = %d, want 42", result.Usage)
	}
}

func TestServiceStatusCombinesRuntimeData(t *testing.T) {
	t.Parallel()

	svc := NewService(fakeStore{
		timeSnapshot: TimeSnapshot{Localtime: 1704067200, Uptime: 42},
		memorySnapshot: MemorySnapshot{
			Total:        512 * 1024 * 1024,
			Available:    128 * 1024 * 1024,
			HasAvailable: true,
		},
		cpuUsage: 55.8,
	})

	result, err := svc.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if result.CPUUsage != 55 || result.MemTotal != "512MB" || result.Localtime != "2024-01-01 00:00:00" {
		t.Fatalf("result = %#v", result)
	}
}

func TestServicePropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("read failed")

	if _, err := NewService(fakeStore{timeErr: expectedErr}).Time(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("Time error = %v, want expectedErr", err)
	}
	if _, err := NewService(fakeStore{memoryErr: expectedErr}).Memory(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("Memory error = %v, want expectedErr", err)
	}
	if _, err := NewService(fakeStore{cpuErr: expectedErr}).CPU(context.Background()); !errors.Is(err, expectedErr) {
		t.Fatalf("CPU error = %v, want expectedErr", err)
	}
}

package service

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type fakeTrafficInventory struct {
	responses []*models.DeviceInventoryResponse
	index     int
}

func (inventory *fakeTrafficInventory) Snapshot(context.Context) (*models.DeviceInventoryResponse, error) {
	index := inventory.index
	if index >= len(inventory.responses) {
		index = len(inventory.responses) - 1
	}
	inventory.index++
	return inventory.responses[index], nil
}

type fakeTrafficSampler struct {
	samples []lanStatsSnapshot
	index   int
}

type churnTrafficInventory struct{ index int }

func (inventory *churnTrafficInventory) Snapshot(context.Context) (*models.DeviceInventoryResponse, error) {
	inventory.index++
	return trafficInventoryResponse(fmt.Sprintf("mac:churn:%06d", inventory.index), fmt.Sprintf("fd00::%x", inventory.index)), nil
}

func (sampler *fakeTrafficSampler) Sample(context.Context) lanStatsSnapshot {
	index := sampler.index
	if index >= len(sampler.samples) {
		index = len(sampler.samples) - 1
	}
	sampler.index++
	return sampler.samples[index]
}

func trafficInventoryResponse(deviceID string, addresses ...string) *models.DeviceInventoryResponse {
	current := make([]*models.DeviceInventoryAddress, 0, len(addresses))
	for index, address := range addresses {
		family := int64(4)
		if len(address) > 0 && address[0] == 'f' {
			family = 6
		}
		current = append(current, &models.DeviceInventoryAddress{Family: family, Address: address, Primary: index == 0})
	}
	return &models.DeviceInventoryResponse{Result: &models.DeviceInventoryResult{Devices: []*models.DeviceInventoryItem{{
		DeviceID: deviceID, Addresses: &models.DeviceInventoryAddresses{Current: current},
	}}}}
}

func trafficHost(ip string, txBytes, rxBytes, txSpeed, rxSpeed, connections int64, sampledAt time.Time) *LanHostRet {
	return &LanHostRet{
		ip: ip, txBytes: txBytes, rxBytes: rxBytes, connectionCount: connections, sampledAt: sampledAt,
		items: []*NetworkStatisticsItem{{txAvg: txSpeed, rxAvg: rxSpeed}},
	}
}

func TestDeviceTrafficAggregatesAddressesAndAccumulatesDeltas(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	inventory := &fakeTrafficInventory{responses: []*models.DeviceInventoryResponse{
		trafficInventoryResponse("mac:device", "192.168.100.20", "fd5f:e357:7969::20"),
	}}
	sampler := &fakeTrafficSampler{samples: []lanStatsSnapshot{
		{samples: 1, sampledAt: now, hosts: []*LanHostRet{
			trafficHost("192.168.100.20", 100, 200, 0, 0, 1, now),
			trafficHost("fd5f:e357:7969::20", 10, 20, 0, 0, 1, now),
		}},
		{samples: 2, sampledAt: now.Add(3 * time.Second), hosts: []*LanHostRet{
			trafficHost("192.168.100.20", 160, 280, 30, 40, 2, now),
			trafficHost("fd5f:e357:7969::20", 25, 35, 5, 6, 1, now),
		}},
	}}
	module := newDeviceTrafficModuleForTest(inventory, sampler, func() time.Time { return now })
	first, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Result.Health.State != "warming_up" || first.Result.Items[0].UploadBytes != 0 {
		t.Fatalf("first sample = %#v", first.Result)
	}
	now = now.Add(3 * time.Second)
	second, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	item := second.Result.Items[0]
	if item.State != "ready" || item.UploadSpeed != 35 || item.DownloadSpeed != 46 || item.UploadBytes != 75 || item.DownloadBytes != 95 || item.ConnectionCount != 3 {
		t.Fatalf("aggregated item = %#v", item)
	}
}

func TestDeviceTrafficResetsBaselineWhenAddressOwnerChanges(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	inventory := &fakeTrafficInventory{responses: []*models.DeviceInventoryResponse{
		trafficInventoryResponse("mac:old", "192.168.100.20"),
		trafficInventoryResponse("mac:new", "192.168.100.20"),
		trafficInventoryResponse("mac:new", "192.168.100.20"),
	}}
	sampler := &fakeTrafficSampler{samples: []lanStatsSnapshot{
		{samples: 1, sampledAt: now, hosts: []*LanHostRet{trafficHost("192.168.100.20", 100, 100, 0, 0, 1, now)}},
		{samples: 2, sampledAt: now.Add(3 * time.Second), hosts: []*LanHostRet{trafficHost("192.168.100.20", 180, 190, 10, 10, 1, now)}},
		{samples: 3, sampledAt: now.Add(6 * time.Second), hosts: []*LanHostRet{trafficHost("192.168.100.20", 210, 240, 10, 10, 1, now)}},
	}}
	module := newDeviceTrafficModuleForTest(inventory, sampler, func() time.Time { return now })
	if _, err := module.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	now = now.Add(3 * time.Second)
	changed, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if changed.Result.Items[0].UploadBytes != 0 || changed.Result.Items[0].DownloadBytes != 0 {
		t.Fatalf("new owner inherited bytes: %#v", changed.Result.Items[0])
	}
	now = now.Add(3 * time.Second)
	next, err := module.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if next.Result.Items[0].UploadBytes != 30 || next.Result.Items[0].DownloadBytes != 50 {
		t.Fatalf("new owner delta = %#v", next.Result.Items[0])
	}
}

func TestDeviceTrafficReportsPartialAndStaleStates(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 24, 3, 0, 20, 0, time.UTC)
	inventory := &fakeTrafficInventory{responses: []*models.DeviceInventoryResponse{trafficInventoryResponse("mac:device", "192.168.100.20")}}
	sampler := &fakeTrafficSampler{samples: []lanStatsSnapshot{
		{samples: 2, sampledAt: now.Add(-20 * time.Second)},
		{samples: 2, sampledAt: now, err: errors.New("conntrack failed")},
	}}
	module := newDeviceTrafficModuleForTest(inventory, sampler, func() time.Time { return now })
	stale, err := module.Snapshot(context.Background())
	if err != nil || stale.Result.Health.State != "stale" {
		t.Fatalf("stale response = %#v, %v", stale, err)
	}
	partial, err := module.Snapshot(context.Background())
	if err != nil || partial.Result.Health.State != "partial" || partial.Result.Health.Reasons[0] != "conntrack_unavailable" {
		t.Fatalf("partial response = %#v, %v", partial, err)
	}
}

func TestDeviceTrafficSimulatedTwoHourStateIsBounded(t *testing.T) {
	now := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	devices := make([]*models.DeviceInventoryItem, 0, 20)
	hosts := make([]*LanHostRet, 0, 20)
	for index := 0; index < 20; index++ {
		address := fmt.Sprintf("192.168.100.%d", index+10)
		devices = append(devices, &models.DeviceInventoryItem{DeviceID: fmt.Sprintf("mac:%02d", index), Addresses: &models.DeviceInventoryAddresses{
			Current: []*models.DeviceInventoryAddress{{Family: 4, Address: address, Primary: true}},
		}})
		hosts = append(hosts, trafficHost(address, 1000, 1000, 1, 1, 1, now))
	}
	inventory := &fakeTrafficInventory{responses: []*models.DeviceInventoryResponse{{Result: &models.DeviceInventoryResult{Devices: devices}}}}
	sampler := &fakeTrafficSampler{samples: []lanStatsSnapshot{{samples: 2, sampledAt: now, hosts: hosts}}}
	module := newDeviceTrafficModuleForTest(inventory, sampler, func() time.Time { return now })
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for sample := 0; sample < 2400; sample++ {
		now = now.Add(3 * time.Second)
		sampler.samples[0].sampledAt = now
		for _, host := range hosts {
			host.txBytes += 10
			host.rxBytes += 20
		}
		if _, err := module.Snapshot(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if len(module.addresses) > 20 || len(module.addresses) > deviceTrafficAddressLimit {
		t.Fatalf("address state grew to %d", len(module.addresses))
	}
	if len(module.totals) > 20 || len(module.totals) > deviceTrafficTotalLimit {
		t.Fatalf("total state grew to %d", len(module.totals))
	}
	if growth := int64(after.HeapAlloc) - int64(before.HeapAlloc); growth > 10*1024*1024 {
		t.Fatalf("heap grew by %d bytes", growth)
	}
}

func TestDeviceTrafficHundredThousandIdentityChurnIsBounded(t *testing.T) {
	now := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	inventory := &churnTrafficInventory{}
	sampler := &fakeTrafficSampler{samples: []lanStatsSnapshot{{samples: 2, sampledAt: now}}}
	module := newDeviceTrafficModuleForTest(inventory, sampler, func() time.Time { return now })
	for index := 0; index < 100_000; index++ {
		now = now.Add(time.Millisecond)
		sampler.samples[0].sampledAt = now
		if _, err := module.Snapshot(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if len(module.addresses) > deviceTrafficAddressLimit || len(module.totals) > deviceTrafficTotalLimit {
		t.Fatalf("unbounded churn state: addresses=%d totals=%d", len(module.addresses), len(module.totals))
	}
	if module.addressExpiry.Len() > deviceTrafficAddressLimit*4 || module.totalExpiry.Len() > deviceTrafficTotalLimit*4 {
		t.Fatalf("unbounded expiry indexes: addresses=%d totals=%d", module.addressExpiry.Len(), module.totalExpiry.Len())
	}
}

func benchmarkDeviceTrafficSnapshot(b *testing.B, deviceCount int) {
	now := time.Date(2026, 9, 24, 3, 0, 0, 0, time.UTC)
	devices := make([]*models.DeviceInventoryItem, 0, deviceCount)
	hosts := make([]*LanHostRet, 0, deviceCount)
	for index := 0; index < deviceCount; index++ {
		address := fmt.Sprintf("192.168.%d.%d", index/250+10, index%250+1)
		devices = append(devices, &models.DeviceInventoryItem{
			DeviceID:  fmt.Sprintf("mac:benchmark:%03d", index),
			Addresses: &models.DeviceInventoryAddresses{Current: []*models.DeviceInventoryAddress{{Family: 4, Address: address}}},
		})
		hosts = append(hosts, trafficHost(address, 1000, 2000, 100, 200, 2, now))
	}
	inventory := &fakeTrafficInventory{responses: []*models.DeviceInventoryResponse{{Result: &models.DeviceInventoryResult{Devices: devices}}}}
	sampler := &fakeTrafficSampler{samples: []lanStatsSnapshot{{samples: 2, sampledAt: now, hosts: hosts}}}
	module := newDeviceTrafficModuleForTest(inventory, sampler, func() time.Time { return now })
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := module.Snapshot(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeviceTrafficSnapshot20(b *testing.B)   { benchmarkDeviceTrafficSnapshot(b, 20) }
func BenchmarkDeviceTrafficSnapshot256(b *testing.B)  { benchmarkDeviceTrafficSnapshot(b, 256) }
func BenchmarkDeviceTrafficSnapshot2048(b *testing.B) { benchmarkDeviceTrafficSnapshot(b, 2048) }

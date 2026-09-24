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
	if growth := int64(after.HeapAlloc) - int64(before.HeapAlloc); growth > 10*1024*1024 {
		t.Fatalf("heap grew by %d bytes", growth)
	}
}

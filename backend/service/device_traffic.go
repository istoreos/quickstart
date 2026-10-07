package service

import (
	"container/heap"
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	deviceTrafficAddressLimit = 2048
	deviceTrafficAddressTTL   = 10 * time.Minute
	deviceTrafficTotalLimit   = 2048
	deviceTrafficTotalTTL     = 10 * time.Minute
	deviceTrafficStaleAfter   = 10 * time.Second
)

type deviceInventorySnapshotter interface {
	Snapshot(context.Context) (*models.DeviceInventoryResponse, error)
}

type deviceTrafficSampler interface {
	Sample(context.Context) lanStatsSnapshot
}

type lanStatsDeviceTrafficSampler struct{ stats *LanStats }

func (sampler lanStatsDeviceTrafficSampler) Sample(ctx context.Context) lanStatsSnapshot {
	if sampler.stats == nil {
		return lanStatsSnapshot{err: errors.New("LAN traffic sampler is unavailable")}
	}
	if err := ctx.Err(); err != nil {
		return lanStatsSnapshot{err: err}
	}
	return sampler.stats.reqSnapshotContext(ctx, "", true)
}

type deviceTrafficAddressState struct {
	Owner         string
	UploadBytes   int64
	DownloadBytes int64
	LastSeen      time.Time
	Generation    uint64
}

type deviceTrafficTotals struct {
	UploadBytes   int64
	DownloadBytes int64
	LastSeen      time.Time
	Generation    uint64
}

type deviceTrafficStateRef struct {
	Key        string
	LastSeen   time.Time
	Generation uint64
}

type deviceTrafficStateHeap []deviceTrafficStateRef

func (values deviceTrafficStateHeap) Len() int { return len(values) }
func (values deviceTrafficStateHeap) Less(i, j int) bool {
	if values[i].LastSeen.Equal(values[j].LastSeen) {
		return values[i].Key < values[j].Key
	}
	return values[i].LastSeen.Before(values[j].LastSeen)
}
func (values deviceTrafficStateHeap) Swap(i, j int) { values[i], values[j] = values[j], values[i] }
func (values *deviceTrafficStateHeap) Push(value any) {
	*values = append(*values, value.(deviceTrafficStateRef))
}
func (values *deviceTrafficStateHeap) Pop() any {
	old := *values
	value := old[len(old)-1]
	*values = old[:len(old)-1]
	return value
}

// DeviceTrafficModule owns address-to-device attribution and traffic baselines.
// Callers provide no IPs, so an address can never be attributed outside the
// current DeviceInventory ownership snapshot.
type DeviceTrafficModule struct {
	mu            sync.Mutex
	inventory     deviceInventorySnapshotter
	sampler       deviceTrafficSampler
	now           func() time.Time
	addresses     map[string]*deviceTrafficAddressState
	totals        map[string]*deviceTrafficTotals
	addressExpiry deviceTrafficStateHeap
	totalExpiry   deviceTrafficStateHeap
	generation    uint64
}

func NewDeviceTrafficModule(inventory deviceInventorySnapshotter, stats *LanStats) *DeviceTrafficModule {
	return &DeviceTrafficModule{
		inventory: inventory, sampler: lanStatsDeviceTrafficSampler{stats: stats}, now: time.Now,
		addresses: map[string]*deviceTrafficAddressState{}, totals: map[string]*deviceTrafficTotals{},
	}
}

func newDeviceTrafficModuleForTest(inventory deviceInventorySnapshotter, sampler deviceTrafficSampler, now func() time.Time) *DeviceTrafficModule {
	return &DeviceTrafficModule{
		inventory: inventory, sampler: sampler, now: now,
		addresses: map[string]*deviceTrafficAddressState{}, totals: map[string]*deviceTrafficTotals{},
	}
}

func (module *DeviceTrafficModule) Snapshot(ctx context.Context) (*models.DeviceTrafficResponse, error) {
	if module == nil || module.inventory == nil || module.sampler == nil {
		return nil, errors.New("device traffic module is unavailable")
	}
	inventory, err := module.inventory.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	sample := module.sampler.Sample(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	module.mu.Lock()
	defer module.mu.Unlock()
	now := module.currentTime().UTC()
	module.pruneState(now)

	hosts := make(map[string]*LanHostRet, len(sample.hosts))
	for _, host := range sample.hosts {
		if host != nil {
			hosts[canonicalInventoryAddress(host.ip)] = host
		}
	}
	state, reasons := deviceTrafficHealth(sample, now)
	items := make([]*models.DeviceTrafficItem, 0, len(inventory.Result.Devices))
	for _, device := range inventory.Result.Devices {
		if device == nil {
			continue
		}
		total := module.totals[device.DeviceID]
		if total == nil {
			total = &deviceTrafficTotals{}
			module.totals[device.DeviceID] = total
		}
		module.touchTotal(device.DeviceID, total, now)
		var uploadSpeed, downloadSpeed, connections int64
		for _, address := range device.Addresses.Current {
			if address == nil {
				continue
			}
			value := canonicalInventoryAddress(address.Address)
			if value == "" {
				continue
			}
			host := hosts[value]
			module.updateAddressOwner(value, device.DeviceID, host, total, now)
			if host == nil {
				continue
			}
			connections += host.connectionCount
			if len(host.items) > 0 {
				latest := host.items[len(host.items)-1]
				uploadSpeed += latest.txAvg
				downloadSpeed += latest.rxAvg
			}
		}
		itemState := state
		items = append(items, &models.DeviceTrafficItem{
			DeviceID: device.DeviceID, UploadSpeed: uploadSpeed, DownloadSpeed: downloadSpeed,
			UploadBytes: total.UploadBytes, DownloadBytes: total.DownloadBytes,
			ConnectionCount: connections, State: itemState, SampledAt: formatTrafficSampleTime(sample.sampledAt),
		})
	}
	module.enforceStateLimits()
	sort.Slice(items, func(i, j int) bool { return items[i].DeviceID < items[j].DeviceID })
	return &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{
		Items: items, Health: &models.DeviceInventoryHealth{State: state, Reasons: reasons},
	}}, nil
}

func (module *DeviceTrafficModule) updateAddressOwner(address, owner string, host *LanHostRet, total *deviceTrafficTotals, now time.Time) {
	state := module.addresses[address]
	if state == nil || state.Owner != owner {
		state = &deviceTrafficAddressState{Owner: owner, LastSeen: now}
		if host != nil {
			state.UploadBytes = host.txBytes
			state.DownloadBytes = host.rxBytes
		}
		module.addresses[address] = state
		module.touchAddress(address, state, now)
		return
	}
	module.touchAddress(address, state, now)
	if host == nil {
		return
	}
	if host.txBytes >= state.UploadBytes {
		total.UploadBytes += host.txBytes - state.UploadBytes
	}
	if host.rxBytes >= state.DownloadBytes {
		total.DownloadBytes += host.rxBytes - state.DownloadBytes
	}
	state.UploadBytes = host.txBytes
	state.DownloadBytes = host.rxBytes
}

func deviceTrafficHealth(sample lanStatsSnapshot, now time.Time) (string, []string) {
	if sample.err != nil {
		return "partial", []string{"conntrack_unavailable"}
	}
	if sample.samples < 2 || sample.sampledAt.IsZero() {
		return "warming_up", nil
	}
	if now.Sub(sample.sampledAt) > deviceTrafficStaleAfter {
		return "stale", []string{"sample_stale"}
	}
	return "ready", nil
}

func (module *DeviceTrafficModule) touchAddress(address string, state *deviceTrafficAddressState, now time.Time) {
	module.generation++
	state.LastSeen = now
	state.Generation = module.generation
	heap.Push(&module.addressExpiry, deviceTrafficStateRef{Key: address, LastSeen: now, Generation: state.Generation})
}

func (module *DeviceTrafficModule) touchTotal(deviceID string, state *deviceTrafficTotals, now time.Time) {
	module.generation++
	state.LastSeen = now
	state.Generation = module.generation
	heap.Push(&module.totalExpiry, deviceTrafficStateRef{Key: deviceID, LastSeen: now, Generation: state.Generation})
}

func (module *DeviceTrafficModule) pruneState(now time.Time) {
	module.pruneAddressExpiry(now.Add(-deviceTrafficAddressTTL))
	module.pruneTotalExpiry(now.Add(-deviceTrafficTotalTTL))
	module.enforceStateLimits()
}

func (module *DeviceTrafficModule) pruneAddressExpiry(cutoff time.Time) {
	for module.addressExpiry.Len() > 0 && module.addressExpiry[0].LastSeen.Before(cutoff) {
		entry := heap.Pop(&module.addressExpiry).(deviceTrafficStateRef)
		if current := module.addresses[entry.Key]; current != nil && current.Generation == entry.Generation {
			delete(module.addresses, entry.Key)
		}
	}
}

func (module *DeviceTrafficModule) pruneTotalExpiry(cutoff time.Time) {
	for module.totalExpiry.Len() > 0 && module.totalExpiry[0].LastSeen.Before(cutoff) {
		entry := heap.Pop(&module.totalExpiry).(deviceTrafficStateRef)
		if current := module.totals[entry.Key]; current != nil && current.Generation == entry.Generation {
			delete(module.totals, entry.Key)
		}
	}
}

func (module *DeviceTrafficModule) enforceStateLimits() {
	for len(module.addresses) > deviceTrafficAddressLimit && module.addressExpiry.Len() > 0 {
		entry := heap.Pop(&module.addressExpiry).(deviceTrafficStateRef)
		if current := module.addresses[entry.Key]; current != nil && current.Generation == entry.Generation {
			delete(module.addresses, entry.Key)
		}
	}
	for len(module.totals) > deviceTrafficTotalLimit && module.totalExpiry.Len() > 0 {
		entry := heap.Pop(&module.totalExpiry).(deviceTrafficStateRef)
		if current := module.totals[entry.Key]; current != nil && current.Generation == entry.Generation {
			delete(module.totals, entry.Key)
		}
	}
	if module.addressExpiry.Len() > deviceTrafficAddressLimit*4 {
		module.addressExpiry = rebuildAddressExpiry(module.addresses)
	}
	if module.totalExpiry.Len() > deviceTrafficTotalLimit*4 {
		module.totalExpiry = rebuildTotalExpiry(module.totals)
	}
}

func rebuildAddressExpiry(values map[string]*deviceTrafficAddressState) deviceTrafficStateHeap {
	result := make(deviceTrafficStateHeap, 0, len(values))
	for key, value := range values {
		if value != nil {
			result = append(result, deviceTrafficStateRef{Key: key, LastSeen: value.LastSeen, Generation: value.Generation})
		}
	}
	heap.Init(&result)
	return result
}

func rebuildTotalExpiry(values map[string]*deviceTrafficTotals) deviceTrafficStateHeap {
	result := make(deviceTrafficStateHeap, 0, len(values))
	for key, value := range values {
		if value != nil {
			result = append(result, deviceTrafficStateRef{Key: key, LastSeen: value.LastSeen, Generation: value.Generation})
		}
	}
	heap.Init(&result)
	return result
}

func (module *DeviceTrafficModule) currentTime() time.Time {
	if module.now != nil {
		return module.now()
	}
	return time.Now()
}

func (module *DeviceTrafficModule) diagnostics() *models.DeviceTrafficRuntimeDiagnostics {
	result := &models.DeviceTrafficRuntimeDiagnostics{
		AddressLimit: deviceTrafficAddressLimit,
		TotalLimit:   deviceTrafficTotalLimit,
	}
	if module == nil {
		return result
	}
	module.mu.Lock()
	result.AddressStates = int64(len(module.addresses))
	result.TotalStates = int64(len(module.totals))
	module.mu.Unlock()
	return result
}

func formatTrafficSampleTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

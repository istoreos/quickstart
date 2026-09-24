package service

import (
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
	return sampler.stats.reqSnapshot("", true)
}

type deviceTrafficAddressState struct {
	Owner         string
	UploadBytes   int64
	DownloadBytes int64
	LastSeen      time.Time
}

type deviceTrafficTotals struct {
	UploadBytes   int64
	DownloadBytes int64
}

// DeviceTrafficModule owns address-to-device attribution and traffic baselines.
// Callers provide no IPs, so an address can never be attributed outside the
// current DeviceInventory ownership snapshot.
type DeviceTrafficModule struct {
	mu        sync.Mutex
	inventory deviceInventorySnapshotter
	sampler   deviceTrafficSampler
	now       func() time.Time
	addresses map[string]*deviceTrafficAddressState
	totals    map[string]*deviceTrafficTotals
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
	module.pruneAddresses(now)

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
		module.enforceAddressLimit()
		return
	}
	state.LastSeen = now
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

func (module *DeviceTrafficModule) pruneAddresses(now time.Time) {
	for address, state := range module.addresses {
		if state == nil || now.Sub(state.LastSeen) > deviceTrafficAddressTTL {
			delete(module.addresses, address)
		}
	}
	module.enforceAddressLimit()
}

func (module *DeviceTrafficModule) enforceAddressLimit() {
	for len(module.addresses) > deviceTrafficAddressLimit {
		oldest := ""
		for address, state := range module.addresses {
			if oldest == "" || state.LastSeen.Before(module.addresses[oldest].LastSeen) ||
				(state.LastSeen.Equal(module.addresses[oldest].LastSeen) && address < oldest) {
				oldest = address
			}
		}
		delete(module.addresses, oldest)
	}
}

func (module *DeviceTrafficModule) currentTime() time.Time {
	if module.now != nil {
		return module.now()
	}
	return time.Now()
}

func formatTrafficSampleTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

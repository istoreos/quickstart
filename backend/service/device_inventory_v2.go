package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	defaultDeviceInventoryHistoryPath = "/tmp/quickstart-device-inventory-v2.json"
	defaultManualDevicePath           = "/etc/quickstart/manual-devices.json"
	defaultDeviceInventoryLimit       = 2048
	defaultDeviceInventoryCacheTTL    = 10 * time.Second
	deviceInventoryPersistDebounce    = 10 * time.Minute
)

type deviceInventoryObservation struct {
	MAC      string
	IPv4     string
	IPv6     string
	DUID     string
	IAID     string
	Hostname string
	Online   bool
}

type deviceInventorySourceSnapshot struct {
	ARP       []deviceInventoryObservation
	DHCPv4    []deviceInventoryObservation
	NDP       []deviceInventoryObservation
	DHCPv6    []deviceInventoryObservation
	HostHints map[string]HostHintSnapshot
	WifiMACs  map[string]struct{}
	Reasons   []string
}

type deviceInventorySource interface {
	Read(context.Context) deviceInventorySourceSnapshot
}

type deviceInventoryHistory struct {
	BootID  string                                 `json:"bootId"`
	Devices map[string]*deviceInventoryHistoryItem `json:"devices"`
}

type deviceInventoryHistoryItem struct {
	DeviceID      string   `json:"deviceId"`
	MAC           string   `json:"mac"`
	Hostname      string   `json:"hostname,omitempty"`
	IdentityKind  string   `json:"identityKind,omitempty"`
	IdentityScope string   `json:"identityScope,omitempty"`
	LastSeenAt    string   `json:"lastSeenAt"`
	Addresses     []string `json:"addresses,omitempty"`
}

type manualDeviceDocument struct {
	SchemaVersion int                           `json:"schemaVersion"`
	Devices       map[string]manualDeviceRecord `json:"devices"`
}

type manualDeviceRecord struct {
	MAC       string `json:"mac"`
	CreatedAt string `json:"createdAt"`
}

type deviceInventoryAggregate struct {
	DeviceID      string
	MAC           string
	Hostname      string
	IdentityKind  string
	IdentityScope string
	Online        bool
	Addresses     map[string]struct{}
}

// DeviceInventoryModule is the deep device identity module. Callers only need
// Snapshot; source precedence, identity, history and partial-health rules stay
// inside its implementation.
type DeviceInventoryModule struct {
	mu                      sync.Mutex
	cacheMu                 sync.Mutex
	source                  deviceInventorySource
	classifier              deviceClassifier
	classificationOverrides *DeviceClassificationOverrideStore
	historyPath             string
	manualPath              string
	bootID                  string
	now                     func() time.Time
	limit                   int
	loaded                  bool
	history                 deviceInventoryHistory
	cacheTTL                time.Duration
	cached                  *models.DeviceInventoryResponse
	cachedAt                time.Time
	refreshing              chan struct{}
	refreshErr              error
	lastPersisted           []byte
	lastPersistAt           time.Time
}

func NewDeviceInventoryModule() *DeviceInventoryModule {
	return &DeviceInventoryModule{
		source:                  systemDeviceInventorySource{},
		classifier:              NewDeviceClassifier(),
		classificationOverrides: NewDeviceProfileStore(defaultDeviceProfilePath, defaultDeviceClassificationPath),
		historyPath:             defaultDeviceInventoryHistoryPath,
		manualPath:              defaultManualDevicePath,
		bootID:                  readDeviceInventoryBootID(),
		now:                     time.Now,
		limit:                   defaultDeviceInventoryLimit,
		cacheTTL:                defaultDeviceInventoryCacheTTL,
	}
}

func newDeviceInventoryModuleForTest(source deviceInventorySource, historyPath, bootID string, now func() time.Time, limit int) *DeviceInventoryModule {
	return &DeviceInventoryModule{source: source, classifier: NewDeviceClassifier(), classificationOverrides: NewDeviceClassificationOverrideStore(historyPath + ".classifications"), historyPath: historyPath, manualPath: historyPath + ".manual", bootID: bootID, now: now, limit: limit}
}

func (module *DeviceInventoryModule) Snapshot(ctx context.Context) (*models.DeviceInventoryResponse, error) {
	if module == nil || module.source == nil {
		return nil, errors.New("device inventory source is unavailable")
	}
	now := module.currentTime()
	module.cacheMu.Lock()
	if module.cached != nil && module.cacheTTL > 0 && now.Sub(module.cachedAt) < module.cacheTTL {
		cached := module.cached
		module.cacheMu.Unlock()
		return cached, nil
	}
	if refresh := module.refreshing; refresh != nil {
		module.cacheMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-refresh:
		}
		module.cacheMu.Lock()
		cached, err := module.cached, module.refreshErr
		module.cacheMu.Unlock()
		return cached, err
	}
	refresh := make(chan struct{})
	module.refreshing = refresh
	module.cacheMu.Unlock()

	response, err := module.refreshSnapshot(ctx)
	module.cacheMu.Lock()
	if err == nil {
		module.cached = response
		module.cachedAt = module.currentTime()
	}
	module.refreshErr = err
	module.refreshing = nil
	close(refresh)
	module.cacheMu.Unlock()
	return response, err
}

func (module *DeviceInventoryModule) Invalidate() {
	if module == nil {
		return
	}
	module.cacheMu.Lock()
	module.cached = nil
	module.cachedAt = time.Time{}
	module.cacheMu.Unlock()
}

func (module *DeviceInventoryModule) diagnostics() *models.DeviceInventoryRuntimeDiagnostics {
	result := &models.DeviceInventoryRuntimeDiagnostics{HistoryLimit: defaultDeviceInventoryLimit}
	if module == nil {
		return result
	}
	result.HistoryLimit = int64(module.deviceLimit())
	now := module.currentTime()
	module.cacheMu.Lock()
	result.CacheReady = module.cached != nil
	result.RefreshInFlight = module.refreshing != nil
	result.CacheTTLMS = module.cacheTTL.Milliseconds()
	if !module.cachedAt.IsZero() {
		result.CacheAgeMS = max(now.Sub(module.cachedAt).Milliseconds(), 0)
	}
	module.cacheMu.Unlock()
	module.mu.Lock()
	result.HistoryEntries = int64(len(module.history.Devices))
	if !module.lastPersistAt.IsZero() {
		result.LastPersistAgeMS = max(now.Sub(module.lastPersistAt).Milliseconds(), 0)
	}
	module.mu.Unlock()
	return result
}

func (module *DeviceInventoryModule) refreshSnapshot(ctx context.Context) (*models.DeviceInventoryResponse, error) {
	source := module.source.Read(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	module.mu.Lock()
	defer module.mu.Unlock()
	module.loadHistory()

	now := module.currentTime().UTC()
	current := mergeDeviceInventoryObservations(source)
	items := make(map[string]*models.DeviceInventoryItem, len(current)+len(module.history.Devices))

	for deviceID, observation := range current {
		hostname := strings.TrimSpace(observation.Hostname)
		connection := "lan"
		if _, ok := source.WifiMACs[observation.MAC]; ok {
			connection = "wifi"
		}

		lastSeen := now
		if !observation.Online {
			if previous := module.history.Devices[deviceID]; previous != nil {
				if parsed, err := time.Parse(time.RFC3339, previous.LastSeenAt); err == nil {
					lastSeen = parsed
				}
			}
		}

		currentAddresses := inventoryAddresses(observation.Addresses, true)
		historicalAddresses := historicalInventoryAddresses(module.history.Devices[deviceID], observation.Addresses)
		manufacturer := GomanufSearch(observation.MAC)
		classification := module.classifier.Classify(DeviceClassificationInput{DisplayName: hostname, Hostname: hostname, Manufacturer: manufacturer})
		displayName := hostname
		profile := deviceProfileRecord{}
		if module.classificationOverrides != nil {
			storedProfile, ok, err := module.classificationOverrides.GetProfile(deviceID, observation.IdentityScope)
			if err != nil {
				source.Reasons = append(source.Reasons, "device_profile_store_invalid")
			} else if ok {
				profile = storedProfile
				if profile.Alias != "" {
					displayName = profile.Alias
				}
				if profile.Brand != "" {
					classification.Brand = profile.Brand
					classification.Source = "manual"
					classification.Confidence = "high"
				}
				if profile.Category != "" {
					classification.Category = profile.Category
					classification.Source = "manual"
					classification.Confidence = "high"
				}
			}
		}
		presenceState := "offline"
		if observation.Online {
			presenceState = "online"
		}
		items[deviceID] = &models.DeviceInventoryItem{
			DeviceID:       deviceID,
			DisplayName:    displayName,
			Hostname:       hostname,
			Online:         observation.Online,
			PresenceState:  presenceState,
			LastSeenAt:     lastSeen.Format(time.RFC3339),
			Mac:            observation.MAC,
			Vendor:         manufacturer,
			Classification: classification,
			Icon:           deviceIconForProfile(classification, profile),
			Identity:       &models.DeviceInventoryIdentity{Kind: observation.IdentityKind, Scope: observation.IdentityScope},
			Addresses: &models.DeviceInventoryAddresses{
				Current:    currentAddresses,
				Historical: historicalAddresses,
			},
			Connection: &models.DeviceInventoryConnection{Kind: connection},
		}
	}

	for deviceID, previous := range module.history.Devices {
		if _, ok := items[deviceID]; ok || previous == nil {
			continue
		}
		item, overrideErr := historyItemToOfflineDevice(previous, module.classifier, module.classificationOverrides)
		if overrideErr != nil {
			source.Reasons = append(source.Reasons, "device_profile_store_invalid")
		}
		items[deviceID] = item
	}
	for deviceID, manual := range module.readManualDevices() {
		if _, exists := items[deviceID]; exists {
			continue
		}
		classification := module.classifier.Classify(DeviceClassificationInput{})
		displayName := ""
		profile := deviceProfileRecord{}
		if module.classificationOverrides != nil {
			stored, ok, profileErr := module.classificationOverrides.GetProfile(deviceID, "persistent")
			if profileErr != nil {
				source.Reasons = append(source.Reasons, "device_profile_store_invalid")
			} else if ok {
				profile = stored
				displayName = stored.Alias
				if stored.Brand != "" {
					classification.Brand, classification.Source, classification.Confidence = stored.Brand, "manual", "high"
				}
				if stored.Category != "" {
					classification.Category, classification.Source, classification.Confidence = stored.Category, "manual", "high"
				}
			}
		}
		items[deviceID] = &models.DeviceInventoryItem{
			DeviceID: deviceID, DisplayName: displayName, Online: false, PresenceState: "never_seen", LastSeenAt: manual.CreatedAt,
			Mac: manual.MAC, Classification: classification, Icon: deviceIconForProfile(classification, profile),
			Identity:   &models.DeviceInventoryIdentity{Kind: "mac", Scope: "persistent"},
			Addresses:  &models.DeviceInventoryAddresses{Current: []*models.DeviceInventoryAddress{}, Historical: []*models.DeviceInventoryAddress{}},
			Connection: &models.DeviceInventoryConnection{Kind: "unknown"},
		}
	}

	devices := sortedDeviceInventoryItems(items)
	if limit := module.deviceLimit(); len(devices) > limit {
		devices = devices[:limit]
		source.Reasons = append(source.Reasons, "inventory_capacity_reached")
	}
	module.replaceHistory(devices)
	if err := module.persistHistory(now); err != nil {
		source.Reasons = append(source.Reasons, "history_persist_failed")
	}

	reasons := uniqueSortedStrings(source.Reasons)
	healthState := "ready"
	if len(reasons) > 0 {
		healthState = "partial"
	}
	return &models.DeviceInventoryResponse{Result: &models.DeviceInventoryResult{
		Devices: devices,
		Health:  &models.DeviceInventoryHealth{State: healthState, Reasons: reasons},
	}}, nil
}

func mergeDeviceInventoryObservations(source deviceInventorySourceSnapshot) map[string]*deviceInventoryAggregate {
	merged := make(map[string]*deviceInventoryAggregate)
	addMACObservation := func(observation deviceInventoryObservation) {
		mac := normalizeInventoryMAC(observation.MAC)
		if mac == "" {
			return
		}
		deviceID := "mac:" + strings.ToLower(mac)
		item := merged[deviceID]
		if item == nil {
			item = &deviceInventoryAggregate{
				DeviceID: deviceID, MAC: mac, IdentityKind: "mac", IdentityScope: "persistent", Addresses: map[string]struct{}{},
			}
			merged[deviceID] = item
		}
		if item.Hostname == "" {
			item.Hostname = strings.TrimSpace(observation.Hostname)
		}
		item.Online = item.Online || observation.Online
		addInventoryAddress(item.Addresses, observation.IPv4)
		addInventoryAddress(item.Addresses, observation.IPv6)
	}
	for _, observation := range source.DHCPv4 {
		addMACObservation(observation)
	}
	for _, observation := range source.ARP {
		addMACObservation(observation)
	}
	for _, observation := range source.NDP {
		addMACObservation(observation)
	}
	for mac, hint := range source.HostHints {
		observation := deviceInventoryObservation{MAC: mac, Hostname: hint.Hostname}
		addMACObservation(observation)
		deviceID := "mac:" + strings.ToLower(normalizeInventoryMAC(mac))
		item := merged[deviceID]
		if item == nil {
			continue
		}
		for _, address := range hint.IPAddrs {
			addInventoryAddress(item.Addresses, address)
		}
		for _, address := range hint.IPv6Addrs {
			addInventoryAddress(item.Addresses, address)
		}
	}

	addressOwners := make(map[string]*deviceInventoryAggregate)
	for _, item := range merged {
		for address := range item.Addresses {
			addressOwners[address] = item
		}
	}
	for _, observation := range source.DHCPv6 {
		address := canonicalInventoryAddress(observation.IPv6)
		if address == "" {
			continue
		}
		if owner := addressOwners[address]; owner != nil {
			if owner.Hostname == "" {
				owner.Hostname = strings.TrimSpace(observation.Hostname)
			}
			continue
		}
		duid := normalizeInventoryHexID(observation.DUID)
		iaid := normalizeInventoryHexID(observation.IAID)
		if duid == "" || iaid == "" {
			continue
		}
		deviceID := "duid:" + duid + ":" + iaid
		item := merged[deviceID]
		if item == nil {
			item = &deviceInventoryAggregate{
				DeviceID: deviceID, IdentityKind: "duid_iaid", IdentityScope: "boot", Addresses: map[string]struct{}{},
			}
			merged[deviceID] = item
		}
		if item.Hostname == "" {
			item.Hostname = strings.TrimSpace(observation.Hostname)
		}
		item.Addresses[address] = struct{}{}
	}
	return merged
}

func historicalInventoryAddresses(previous *deviceInventoryHistoryItem, current map[string]struct{}) []*models.DeviceInventoryAddress {
	if previous == nil {
		return []*models.DeviceInventoryAddress{}
	}
	addresses := make(map[string]struct{}, len(previous.Addresses))
	for _, address := range uniqueSortedStrings(previous.Addresses) {
		address = canonicalInventoryAddress(address)
		if address == "" {
			continue
		}
		if _, ok := current[address]; ok {
			continue
		}
		addresses[address] = struct{}{}
	}
	return inventoryAddresses(addresses, false)
}

func historyItemToOfflineDevice(item *deviceInventoryHistoryItem, classifier deviceClassifier, overrides *DeviceClassificationOverrideStore) (*models.DeviceInventoryItem, error) {
	addresses := make(map[string]struct{}, len(item.Addresses))
	for _, address := range uniqueSortedStrings(item.Addresses) {
		if address = canonicalInventoryAddress(address); address != "" {
			addresses[address] = struct{}{}
		}
	}
	identityKind, identityScope := inventoryHistoryIdentity(item)
	manufacturer := GomanufSearch(item.MAC)
	classification := classifier.Classify(DeviceClassificationInput{DisplayName: item.Hostname, Hostname: item.Hostname, Manufacturer: manufacturer})
	var overrideErr error
	displayName := item.Hostname
	profile := deviceProfileRecord{}
	if overrides != nil {
		storedProfile, ok, err := overrides.GetProfile(item.DeviceID, identityScope)
		overrideErr = err
		if err == nil && ok {
			profile = storedProfile
			if profile.Alias != "" {
				displayName = profile.Alias
			}
			if profile.Brand != "" {
				classification.Brand = profile.Brand
				classification.Source = "manual"
				classification.Confidence = "high"
			}
			if profile.Category != "" {
				classification.Category = profile.Category
				classification.Source = "manual"
				classification.Confidence = "high"
			}
		}
	}
	return &models.DeviceInventoryItem{
		DeviceID:       item.DeviceID,
		DisplayName:    displayName,
		Hostname:       item.Hostname,
		Online:         false,
		PresenceState:  "offline",
		LastSeenAt:     item.LastSeenAt,
		Mac:            item.MAC,
		Vendor:         manufacturer,
		Classification: classification,
		Icon:           deviceIconForProfile(classification, profile),
		Identity:       &models.DeviceInventoryIdentity{Kind: identityKind, Scope: identityScope},
		Addresses:      &models.DeviceInventoryAddresses{Current: []*models.DeviceInventoryAddress{}, Historical: inventoryAddresses(addresses, false)},
		Connection:     &models.DeviceInventoryConnection{Kind: "unknown"},
	}, overrideErr
}

func sortedDeviceInventoryItems(items map[string]*models.DeviceInventoryItem) []*models.DeviceInventoryItem {
	result := make([]*models.DeviceInventoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Online != result[j].Online {
			return result[i].Online
		}
		if result[i].LastSeenAt != result[j].LastSeenAt {
			return result[i].LastSeenAt > result[j].LastSeenAt
		}
		return result[i].DeviceID < result[j].DeviceID
	})
	return result
}

func (module *DeviceInventoryModule) replaceHistory(devices []*models.DeviceInventoryItem) {
	next := make(map[string]*deviceInventoryHistoryItem, len(devices))
	for _, device := range devices {
		if device.PresenceState == "never_seen" {
			continue
		}
		addresses := make([]string, 0)
		for _, address := range device.Addresses.Current {
			addresses = append(addresses, address.Address)
		}
		for _, address := range device.Addresses.Historical {
			addresses = append(addresses, address.Address)
		}
		next[device.DeviceID] = &deviceInventoryHistoryItem{
			DeviceID: device.DeviceID, MAC: device.Mac, Hostname: device.Hostname,
			IdentityKind: device.Identity.Kind, IdentityScope: device.Identity.Scope,
			LastSeenAt: device.LastSeenAt, Addresses: uniqueSortedStrings(addresses),
		}
	}
	module.history = deviceInventoryHistory{BootID: module.bootID, Devices: next}
}

func (module *DeviceInventoryModule) AddManual(ctx context.Context, request *models.DeviceInventoryAddRequest) (*models.DeviceInventoryResponse, error) {
	if request == nil {
		return nil, errors.New("manual device request is required")
	}
	mac := normalizeInventoryMAC(request.MAC)
	firstOctet, _ := strconv.ParseUint(strings.TrimSpace(strings.Split(mac, ":")[0]), 16, 8)
	if mac == "" || firstOctet&1 != 0 {
		return nil, errors.New("a valid unicast MAC address is required")
	}
	alias, err := normalizeDeviceAlias(request.Alias)
	if err != nil {
		return nil, err
	}
	deviceID := "mac:" + strings.ToLower(mac)
	module.mu.Lock()
	document := module.readManualDeviceDocument()
	if _, exists := document.Devices[deviceID]; !exists && len(document.Devices) >= module.deviceLimit() {
		module.mu.Unlock()
		return nil, errors.New("manual device limit reached")
	}
	document.Devices[deviceID] = manualDeviceRecord{MAC: mac, CreatedAt: module.currentTime().UTC().Format(time.RFC3339)}
	raw, err := json.Marshal(document)
	if err == nil {
		err = persistClassificationOverrides(module.manualPath, raw)
	}
	module.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if module.classificationOverrides != nil && alias != "" {
		if _, err := module.classificationOverrides.PatchProfileFields(deviceID, "persistent", deviceProfilePatch{Alias: &alias}); err != nil {
			return nil, err
		}
	}
	module.Invalidate()
	return module.Snapshot(ctx)
}

func (module *DeviceInventoryModule) readManualDevices() map[string]manualDeviceRecord {
	return module.readManualDeviceDocument().Devices
}

func (module *DeviceInventoryModule) readManualDeviceDocument() manualDeviceDocument {
	document := manualDeviceDocument{SchemaVersion: 1, Devices: map[string]manualDeviceRecord{}}
	raw, err := os.ReadFile(module.manualPath)
	if err != nil {
		return document
	}
	var stored manualDeviceDocument
	if json.Unmarshal(raw, &stored) != nil || stored.SchemaVersion != 1 || stored.Devices == nil {
		return document
	}
	return stored
}

func (module *DeviceInventoryModule) loadHistory() {
	if module.loaded {
		return
	}
	module.loaded = true
	module.history = deviceInventoryHistory{BootID: module.bootID, Devices: map[string]*deviceInventoryHistoryItem{}}
	data, err := os.ReadFile(module.historyPath)
	if err != nil {
		return
	}
	var history deviceInventoryHistory
	if json.Unmarshal(data, &history) != nil || history.BootID != module.bootID || history.Devices == nil {
		return
	}
	module.history = history
	module.lastPersisted, _ = json.Marshal(history)
	if info, statErr := os.Stat(module.historyPath); statErr == nil {
		module.lastPersistAt = info.ModTime()
	}
}

func (module *DeviceInventoryModule) persistHistory(now time.Time) error {
	data, err := json.Marshal(module.history)
	if err != nil {
		return err
	}
	if bytes.Equal(data, module.lastPersisted) {
		return nil
	}
	if !module.lastPersistAt.IsZero() && now.Sub(module.lastPersistAt) < deviceInventoryPersistDebounce {
		return nil
	}
	tmp := module.historyPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, module.historyPath); err != nil {
		return err
	}
	module.lastPersisted = append(module.lastPersisted[:0], data...)
	module.lastPersistAt = now
	return nil
}

func (module *DeviceInventoryModule) currentTime() time.Time {
	if module.now != nil {
		return module.now()
	}
	return time.Now()
}

func (module *DeviceInventoryModule) deviceLimit() int {
	if module.limit > 0 {
		return module.limit
	}
	return defaultDeviceInventoryLimit
}

func normalizeInventoryMAC(mac string) string {
	parts := strings.FieldsFunc(strings.TrimSpace(mac), func(r rune) bool { return r == ':' || r == '-' })
	if len(parts) != 6 {
		return ""
	}
	for _, part := range parts {
		if len(part) != 2 {
			return ""
		}
		if _, err := netip.ParseAddr("::" + part); err != nil {
			// ParseAddr is intentionally not used as the final hex validator.
			for _, char := range part {
				if !strings.ContainsRune("0123456789abcdefABCDEF", char) {
					return ""
				}
			}
		}
	}
	return strings.ToUpper(strings.Join(parts, ":"))
}

func validIPv4(value string) bool {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	return err == nil && address.Is4()
}

func addInventoryAddress(addresses map[string]struct{}, value string) {
	if address := canonicalInventoryAddress(value); address != "" {
		addresses[address] = struct{}{}
	}
}

func canonicalInventoryAddress(value string) string {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil || address.IsUnspecified() || address.IsMulticast() {
		return ""
	}
	return address.Unmap().String()
}

func inventoryAddresses(values map[string]struct{}, primary bool) []*models.DeviceInventoryAddress {
	addresses := make([]netip.Addr, 0, len(values))
	for value := range values {
		if address, err := netip.ParseAddr(value); err == nil {
			addresses = append(addresses, address.Unmap())
		}
	}
	sort.Slice(addresses, func(i, j int) bool {
		rank := func(address netip.Addr) int {
			if address.Is4() {
				return 0
			}
			if !address.IsLinkLocalUnicast() {
				return 1
			}
			return 2
		}
		left, right := rank(addresses[i]), rank(addresses[j])
		if left != right {
			return left < right
		}
		return addresses[i].Less(addresses[j])
	})
	result := make([]*models.DeviceInventoryAddress, 0, len(addresses))
	for index, address := range addresses {
		family := int64(6)
		if address.Is4() {
			family = 4
		}
		result = append(result, &models.DeviceInventoryAddress{Family: family, Address: address.String(), Primary: primary && index == 0})
	}
	return result
}

func inventoryHistoryIdentity(item *deviceInventoryHistoryItem) (string, string) {
	if item.IdentityKind != "" && item.IdentityScope != "" {
		return item.IdentityKind, item.IdentityScope
	}
	if strings.HasPrefix(item.DeviceID, "duid:") {
		return "duid_iaid", "boot"
	}
	return "mac", "persistent"
}

func uniqueSortedStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func readDeviceInventoryBootID() string {
	data, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

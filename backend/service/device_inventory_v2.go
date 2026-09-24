package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/netip"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	defaultDeviceInventoryHistoryPath = "/tmp/quickstart-device-inventory-v2.json"
	defaultDeviceInventoryLimit       = 2048
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
	source                  deviceInventorySource
	classifier              deviceClassifier
	classificationOverrides *DeviceClassificationOverrideStore
	historyPath             string
	bootID                  string
	now                     func() time.Time
	limit                   int
	loaded                  bool
	history                 deviceInventoryHistory
}

func NewDeviceInventoryModule() *DeviceInventoryModule {
	return &DeviceInventoryModule{
		source:                  systemDeviceInventorySource{},
		classifier:              NewDeviceClassifier(),
		classificationOverrides: NewDeviceClassificationOverrideStore(defaultDeviceClassificationPath),
		historyPath:             defaultDeviceInventoryHistoryPath,
		bootID:                  readDeviceInventoryBootID(),
		now:                     time.Now,
		limit:                   defaultDeviceInventoryLimit,
	}
}

func newDeviceInventoryModuleForTest(source deviceInventorySource, historyPath, bootID string, now func() time.Time, limit int) *DeviceInventoryModule {
	return &DeviceInventoryModule{source: source, classifier: NewDeviceClassifier(), classificationOverrides: NewDeviceClassificationOverrideStore(historyPath + ".classifications"), historyPath: historyPath, bootID: bootID, now: now, limit: limit}
}

func (module *DeviceInventoryModule) Snapshot(ctx context.Context) (*models.DeviceInventoryResponse, error) {
	if module == nil || module.source == nil {
		return nil, errors.New("device inventory source is unavailable")
	}
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
		if module.classificationOverrides != nil {
			category, ok, err := module.classificationOverrides.Get(deviceID, observation.IdentityScope)
			if err != nil {
				source.Reasons = append(source.Reasons, "classification_override_store_invalid")
			} else if ok {
				classification.Category = category
				classification.Source = "manual"
				classification.Confidence = "high"
			}
		}
		items[deviceID] = &models.DeviceInventoryItem{
			DeviceID:       deviceID,
			DisplayName:    hostname,
			Hostname:       hostname,
			Online:         observation.Online,
			LastSeenAt:     lastSeen.Format(time.RFC3339),
			Mac:            observation.MAC,
			Vendor:         manufacturer,
			Classification: classification,
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
			source.Reasons = append(source.Reasons, "classification_override_store_invalid")
		}
		items[deviceID] = item
	}

	devices := sortedDeviceInventoryItems(items)
	if limit := module.deviceLimit(); len(devices) > limit {
		devices = devices[:limit]
		source.Reasons = append(source.Reasons, "inventory_capacity_reached")
	}
	module.replaceHistory(devices)
	if err := module.persistHistory(); err != nil {
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
	if overrides != nil {
		category, ok, err := overrides.Get(item.DeviceID, identityScope)
		overrideErr = err
		if err == nil && ok {
			classification.Category = category
			classification.Source = "manual"
			classification.Confidence = "high"
		}
	}
	return &models.DeviceInventoryItem{
		DeviceID:       item.DeviceID,
		DisplayName:    item.Hostname,
		Hostname:       item.Hostname,
		Online:         false,
		LastSeenAt:     item.LastSeenAt,
		Mac:            item.MAC,
		Vendor:         manufacturer,
		Classification: classification,
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
}

func (module *DeviceInventoryModule) persistHistory() error {
	data, err := json.Marshal(module.history)
	if err != nil {
		return err
	}
	tmp := module.historyPath + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, module.historyPath)
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

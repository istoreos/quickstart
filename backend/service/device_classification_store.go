package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	defaultDeviceClassificationPath  = "/etc/quickstart/device-classifications.json"
	defaultDeviceProfilePath         = "/etc/quickstart/device-profiles.json"
	defaultClassificationOverrideMax = 2048
	deviceAliasMaxRunes              = 64
	deviceAliasMaxBytes              = 256
	deviceBrandMaxRunes              = 48
	deviceBrandMaxBytes              = 128
)

type classificationOverrideFile struct {
	Version int               `json:"version"`
	Records map[string]string `json:"records"`
}

type deviceProfileRecord struct {
	Alias    string `json:"alias,omitempty"`
	Brand    string `json:"brand,omitempty"`
	Category string `json:"category,omitempty"`
	IconMode string `json:"iconMode,omitempty"`
	IconKey  string `json:"iconKey,omitempty"`
}

type deviceProfilePatch struct {
	Alias    *string
	Brand    *string
	Category *string
	IconMode *string
	IconKey  *string
}

type deviceProfileFile struct {
	Version int                            `json:"version"`
	Records map[string]deviceProfileRecord `json:"records"`
}

// DeviceClassificationOverrideStore now owns all user-authored device profile
// fields. Its historical name is retained for source compatibility with the
// existing classification API; the on-disk format is Device Profile v3.
type DeviceClassificationOverrideStore struct {
	mu         sync.Mutex
	path       string
	legacyPath string
	limit      int
	loaded     bool
	loadErr    error
	persistent map[string]deviceProfileRecord
	boot       map[string]deviceProfileRecord
	persist    func(string, []byte) error
}

func NewDeviceClassificationOverrideStore(path string) *DeviceClassificationOverrideStore {
	return newDeviceProfileStore(path, "")
}

func NewDeviceProfileStore(path, legacyPath string) *DeviceClassificationOverrideStore {
	return newDeviceProfileStore(path, legacyPath)
}

func newDeviceProfileStore(path, legacyPath string) *DeviceClassificationOverrideStore {
	return &DeviceClassificationOverrideStore{
		path: path, legacyPath: legacyPath, limit: defaultClassificationOverrideMax,
		persistent: map[string]deviceProfileRecord{}, boot: map[string]deviceProfileRecord{}, persist: persistClassificationOverrides,
	}
}

func (store *DeviceClassificationOverrideStore) Get(deviceID, scope string) (string, bool, error) {
	record, ok, err := store.GetProfile(deviceID, scope)
	return record.Category, ok && record.Category != "", err
}

func (store *DeviceClassificationOverrideStore) GetProfile(deviceID, scope string) (deviceProfileRecord, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.load(); err != nil {
		return deviceProfileRecord{}, false, err
	}
	records := store.persistent
	if scope == "boot" {
		records = store.boot
	}
	record, ok := records[deviceID]
	return record, ok, nil
}

func (store *DeviceClassificationOverrideStore) Set(deviceID, scope, category string) (bool, error) {
	return store.PatchProfile(deviceID, scope, nil, &category)
}

func (store *DeviceClassificationOverrideStore) PatchProfile(deviceID, scope string, alias, category *string) (bool, error) {
	return store.PatchProfileFields(deviceID, scope, deviceProfilePatch{Alias: alias, Category: category})
}

func (store *DeviceClassificationOverrideStore) PatchProfileFields(deviceID, scope string, patch deviceProfilePatch) (bool, error) {
	if deviceID == "" || (patch.Alias == nil && patch.Brand == nil && patch.Category == nil && patch.IconMode == nil && patch.IconKey == nil) {
		return false, errors.New("invalid device profile")
	}
	if patch.Alias != nil {
		normalized, err := normalizeDeviceAlias(*patch.Alias)
		if err != nil {
			return false, err
		}
		patch.Alias = &normalized
	}
	if patch.Brand != nil {
		normalized, err := normalizeDeviceBrand(*patch.Brand)
		if err != nil {
			return false, err
		}
		patch.Brand = &normalized
	}
	if patch.Category != nil && *patch.Category != "" && !validDeviceCategory(*patch.Category) {
		return false, errors.New("invalid classification override")
	}
	if patch.IconMode != nil && *patch.IconMode != "auto" && *patch.IconMode != "manual" {
		return false, errors.New("icon mode must be auto or manual")
	}
	if patch.IconKey != nil && *patch.IconKey != "" && !validDeviceIconKey(*patch.IconKey) {
		return false, errors.New("invalid device icon key")
	}
	if patch.IconMode != nil && *patch.IconMode == "manual" && (patch.IconKey == nil || *patch.IconKey == "") {
		return false, errors.New("manual icon mode requires an icon key")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.load(); err != nil {
		return false, err
	}
	records := store.persistent
	if scope == "boot" {
		records = store.boot
	}
	current := records[deviceID]
	nextRecord := current
	if patch.Alias != nil {
		nextRecord.Alias = *patch.Alias
	}
	if patch.Brand != nil {
		nextRecord.Brand = *patch.Brand
	}
	if patch.Category != nil {
		nextRecord.Category = *patch.Category
	}
	if patch.IconMode != nil {
		nextRecord.IconMode = *patch.IconMode
		if *patch.IconMode == "auto" {
			nextRecord.IconKey = ""
		}
	}
	if patch.IconKey != nil {
		nextRecord.IconKey = *patch.IconKey
		if *patch.IconKey != "" {
			nextRecord.IconMode = "manual"
		}
	}
	if nextRecord == current {
		return false, nil
	}
	if _, exists := records[deviceID]; !exists && len(records) >= store.limit && !profileRecordEmpty(nextRecord) {
		return false, errors.New("device profile capacity reached")
	}
	if scope == "boot" {
		if profileRecordEmpty(nextRecord) {
			delete(store.boot, deviceID)
		} else {
			store.boot[deviceID] = nextRecord
		}
		return true, nil
	}
	next := cloneDeviceProfiles(store.persistent)
	if profileRecordEmpty(nextRecord) {
		delete(next, deviceID)
	} else {
		next[deviceID] = nextRecord
	}
	if err := store.write(next); err != nil {
		return false, err
	}
	store.persistent = next
	return true, nil
}

func (store *DeviceClassificationOverrideStore) Reset(deviceID, scope string) (bool, error) {
	category := ""
	return store.PatchProfile(deviceID, scope, nil, &category)
}

func (store *DeviceClassificationOverrideStore) ResetProfile(deviceID, scope string) (bool, error) {
	alias, brand, category, iconMode, iconKey := "", "", "", "auto", ""
	return store.PatchProfileFields(deviceID, scope, deviceProfilePatch{Alias: &alias, Brand: &brand, Category: &category, IconMode: &iconMode, IconKey: &iconKey})
}

func (store *DeviceClassificationOverrideStore) ResetProfileField(deviceID, scope, field string) (bool, error) {
	empty, automatic := "", "auto"
	switch field {
	case "brand":
		return store.PatchProfileFields(deviceID, scope, deviceProfilePatch{Brand: &empty})
	case "category":
		return store.PatchProfileFields(deviceID, scope, deviceProfilePatch{Category: &empty})
	case "icon":
		return store.PatchProfileFields(deviceID, scope, deviceProfilePatch{IconMode: &automatic, IconKey: &empty})
	default:
		return false, errors.New("unknown device profile field")
	}
}

func (store *DeviceClassificationOverrideStore) load() error {
	if store.loaded {
		return store.loadErr
	}
	store.loaded = true
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) && store.legacyPath != "" {
		return store.migrateLegacy()
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		store.loadErr = err
		return err
	}
	var file deviceProfileFile
	if err := json.Unmarshal(data, &file); err != nil || (file.Version != 2 && file.Version != 3) || file.Records == nil {
		if err == nil {
			err = errors.New("invalid device profile file")
		}
		store.loadErr = err
		return err
	}
	if err := validateDeviceProfiles(file.Records, store.limit); err != nil {
		store.loadErr = err
		return err
	}
	store.persistent = cloneDeviceProfiles(file.Records)
	return nil
}

func (store *DeviceClassificationOverrideStore) migrateLegacy() error {
	data, err := os.ReadFile(store.legacyPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		store.loadErr = err
		return err
	}
	var legacy classificationOverrideFile
	if err := json.Unmarshal(data, &legacy); err != nil || legacy.Version != 1 || legacy.Records == nil {
		if err == nil {
			err = errors.New("invalid legacy classification override file")
		}
		store.loadErr = err
		return err
	}
	next := make(map[string]deviceProfileRecord, len(legacy.Records))
	for deviceID, category := range legacy.Records {
		if deviceID == "" || !validDeviceCategory(category) {
			store.loadErr = errors.New("invalid legacy classification override record")
			return store.loadErr
		}
		next[deviceID] = deviceProfileRecord{Category: category}
	}
	if len(next) > store.limit {
		store.loadErr = errors.New("device profile capacity reached")
		return store.loadErr
	}
	if err := store.write(next); err != nil {
		store.loadErr = err
		return err
	}
	store.persistent = next
	return nil
}

func (store *DeviceClassificationOverrideStore) write(records map[string]deviceProfileRecord) error {
	data, err := json.Marshal(deviceProfileFile{Version: 3, Records: records})
	if err != nil {
		return err
	}
	return store.persist(store.path, data)
}

func normalizeDeviceAlias(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || len(value) > deviceAliasMaxBytes || utf8.RuneCountInString(value) > deviceAliasMaxRunes {
		return "", errors.New("device alias must be valid UTF-8 and no longer than 64 characters")
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return "", errors.New("device alias cannot contain control characters")
		}
	}
	return value, nil
}

func normalizeDeviceBrand(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || len(value) > deviceBrandMaxBytes || utf8.RuneCountInString(value) > deviceBrandMaxRunes {
		return "", errors.New("device brand must be valid UTF-8 and no longer than 48 characters")
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return "", errors.New("device brand cannot contain control characters")
		}
	}
	return value, nil
}

func validateDeviceProfiles(records map[string]deviceProfileRecord, limit int) error {
	if len(records) > limit {
		return errors.New("device profile capacity reached")
	}
	for deviceID, record := range records {
		if deviceID == "" || profileRecordEmpty(record) {
			return errors.New("invalid device profile record")
		}
		if _, err := normalizeDeviceAlias(record.Alias); err != nil {
			return err
		}
		if _, err := normalizeDeviceBrand(record.Brand); err != nil {
			return err
		}
		if record.Category != "" && !validDeviceCategory(record.Category) {
			return errors.New("invalid device profile category")
		}
		if record.IconMode != "" && record.IconMode != "auto" && record.IconMode != "manual" {
			return errors.New("invalid device profile icon mode")
		}
		if record.IconKey != "" && !validDeviceIconKey(record.IconKey) {
			return errors.New("invalid device profile icon key")
		}
		if record.IconMode == "manual" && record.IconKey == "" {
			return errors.New("manual device icon is missing a key")
		}
	}
	return nil
}

func profileRecordEmpty(record deviceProfileRecord) bool {
	return record.Alias == "" && record.Brand == "" && record.Category == "" && (record.IconMode == "" || record.IconMode == "auto") && record.IconKey == ""
}

func cloneDeviceProfiles(source map[string]deviceProfileRecord) map[string]deviceProfileRecord {
	result := make(map[string]deviceProfileRecord, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func persistClassificationOverrides(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".device-profiles-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

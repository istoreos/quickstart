package service

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const (
	defaultDeviceClassificationPath  = "/etc/quickstart/device-classifications.json"
	defaultClassificationOverrideMax = 2048
)

type classificationOverrideFile struct {
	Version int               `json:"version"`
	Records map[string]string `json:"records"`
}

type DeviceClassificationOverrideStore struct {
	mu         sync.Mutex
	path       string
	limit      int
	loaded     bool
	loadErr    error
	persistent map[string]string
	boot       map[string]string
	persist    func(string, []byte) error
}

func NewDeviceClassificationOverrideStore(path string) *DeviceClassificationOverrideStore {
	return &DeviceClassificationOverrideStore{
		path: path, limit: defaultClassificationOverrideMax,
		persistent: map[string]string{}, boot: map[string]string{}, persist: persistClassificationOverrides,
	}
}

func (store *DeviceClassificationOverrideStore) Get(deviceID, scope string) (string, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.load(); err != nil {
		return "", false, err
	}
	records := store.persistent
	if scope == "boot" {
		records = store.boot
	}
	category, ok := records[deviceID]
	return category, ok, nil
}

func (store *DeviceClassificationOverrideStore) Set(deviceID, scope, category string) (bool, error) {
	if deviceID == "" || !validDeviceCategory(category) {
		return false, errors.New("invalid classification override")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.load(); err != nil {
		return false, err
	}
	if scope == "boot" {
		if store.boot[deviceID] == category {
			return false, nil
		}
		if len(store.boot) >= store.limit {
			return false, errors.New("classification override capacity reached")
		}
		store.boot[deviceID] = category
		return true, nil
	}
	if store.persistent[deviceID] == category {
		return false, nil
	}
	if _, exists := store.persistent[deviceID]; !exists && len(store.persistent) >= store.limit {
		return false, errors.New("classification override capacity reached")
	}
	next := cloneClassificationOverrides(store.persistent)
	next[deviceID] = category
	if err := store.write(next); err != nil {
		return false, err
	}
	store.persistent = next
	return true, nil
}

func (store *DeviceClassificationOverrideStore) Reset(deviceID, scope string) (bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	if err := store.load(); err != nil {
		return false, err
	}
	if scope == "boot" {
		if _, ok := store.boot[deviceID]; !ok {
			return false, nil
		}
		delete(store.boot, deviceID)
		return true, nil
	}
	if _, ok := store.persistent[deviceID]; !ok {
		return false, nil
	}
	next := cloneClassificationOverrides(store.persistent)
	delete(next, deviceID)
	if err := store.write(next); err != nil {
		return false, err
	}
	store.persistent = next
	return true, nil
}

func (store *DeviceClassificationOverrideStore) load() error {
	if store.loaded {
		return store.loadErr
	}
	store.loaded = true
	data, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		store.loadErr = err
		return err
	}
	var file classificationOverrideFile
	if err := json.Unmarshal(data, &file); err != nil || file.Version != 1 || file.Records == nil {
		if err == nil {
			err = errors.New("invalid classification override file")
		}
		store.loadErr = err
		return err
	}
	for deviceID, category := range file.Records {
		if deviceID == "" || !validDeviceCategory(category) {
			store.loadErr = errors.New("invalid classification override record")
			return store.loadErr
		}
	}
	store.persistent = cloneClassificationOverrides(file.Records)
	return nil
}

func (store *DeviceClassificationOverrideStore) write(records map[string]string) error {
	data, err := json.Marshal(classificationOverrideFile{Version: 1, Records: records})
	if err != nil {
		return err
	}
	return store.persist(store.path, data)
}

func cloneClassificationOverrides(source map[string]string) map[string]string {
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func persistClassificationOverrides(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".device-classifications-*")
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

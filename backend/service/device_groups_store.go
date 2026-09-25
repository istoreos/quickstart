package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"sync"

	"github.com/istoreos/quickstart/backend/models"
)

const defaultDeviceGroupStorePath = "/etc/quickstart/device-groups.json"

type deviceGroupDocument struct {
	SchemaVersion  int                            `json:"schemaVersion"`
	GlobalPolicy   *models.GroupPolicy            `json:"globalPolicy,omitempty"`
	Groups         []*models.DeviceGroup          `json:"groups"`
	DevicePolicies map[string]*models.GroupPolicy `json:"devicePolicies"`
}

type jsonDeviceGroupStore struct {
	mu      sync.Mutex
	path    string
	persist func(string, []byte) error
}

func newJSONDeviceGroupStore(path string) *jsonDeviceGroupStore {
	return &jsonDeviceGroupStore{path: path, persist: persistClassificationOverrides}
}

func (store *jsonDeviceGroupStore) Read(_ context.Context) (deviceGroupState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	document, raw, err := store.readDocument()
	if err != nil {
		return deviceGroupState{}, err
	}
	return documentDeviceGroupState(document, raw), nil
}

func (store *jsonDeviceGroupStore) Mutate(_ context.Context, request *models.DeviceGroupMutationRequest, expectedVersion string) (deviceGroupState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	document, raw, err := store.readDocument()
	if err != nil {
		return deviceGroupState{}, err
	}
	if expectedVersion != "" && expectedVersion != versionDeviceGroups(raw) {
		return deviceGroupState{}, errors.New("device groups changed during update")
	}
	switch request.Action {
	case "upsert_group":
		replaced := false
		for index, group := range document.Groups {
			if group.ID == request.Group.ID {
				document.Groups[index] = request.Group
				replaced = true
				break
			}
		}
		if !replaced {
			document.Groups = append(document.Groups, request.Group)
		}
	case "delete_group":
		groups := document.Groups[:0]
		for _, group := range document.Groups {
			if group.ID != request.GroupID {
				groups = append(groups, group)
			}
		}
		document.Groups = groups
	case "set_device_policy":
		if request.DevicePolicy == nil {
			delete(document.DevicePolicies, request.DeviceID)
		} else {
			document.DevicePolicies[request.DeviceID] = request.DevicePolicy
		}
	case "set_global_policy":
		document.GlobalPolicy = request.GlobalPolicy
	default:
		return deviceGroupState{}, errors.New("unsupported device group action")
	}
	sort.Slice(document.Groups, func(i, j int) bool {
		if document.Groups[i].Priority != document.Groups[j].Priority {
			return document.Groups[i].Priority > document.Groups[j].Priority
		}
		return document.Groups[i].ID < document.Groups[j].ID
	})
	nextRaw, err := json.Marshal(document)
	if err != nil {
		return deviceGroupState{}, err
	}
	if err := store.persist(store.path, nextRaw); err != nil {
		return deviceGroupState{}, err
	}
	return documentDeviceGroupState(document, nextRaw), nil
}

func (store *jsonDeviceGroupStore) Replace(document deviceGroupDocument, expectedVersion string) (deviceGroupState, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	_, raw, err := store.readDocument()
	if err != nil {
		return deviceGroupState{}, err
	}
	if expectedVersion != "" && expectedVersion != versionDeviceGroups(raw) {
		return deviceGroupState{}, errors.New("device groups changed during import")
	}
	if err := validateDeviceGroupDocument(document); err != nil {
		return deviceGroupState{}, err
	}
	document.SchemaVersion = 1
	nextRaw, err := json.Marshal(document)
	if err != nil {
		return deviceGroupState{}, err
	}
	if err := store.persist(store.path, nextRaw); err != nil {
		return deviceGroupState{}, err
	}
	return documentDeviceGroupState(document, nextRaw), nil
}

func (store *jsonDeviceGroupStore) readDocument() (deviceGroupDocument, []byte, error) {
	raw, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		document := emptyDeviceGroupDocument()
		raw, _ = json.Marshal(document)
		return document, raw, nil
	}
	if err != nil {
		return deviceGroupDocument{}, nil, err
	}
	var document deviceGroupDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		return deviceGroupDocument{}, nil, errors.New("invalid device group file")
	}
	if document.SchemaVersion != 1 || document.Groups == nil || document.DevicePolicies == nil {
		return deviceGroupDocument{}, nil, errors.New("unsupported device group file")
	}
	if err := validateDeviceGroupDocument(document); err != nil {
		return deviceGroupDocument{}, nil, err
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return deviceGroupDocument{}, nil, err
	}
	return document, canonical, nil
}

func emptyDeviceGroupDocument() deviceGroupDocument {
	return deviceGroupDocument{SchemaVersion: 1, Groups: []*models.DeviceGroup{}, DevicePolicies: map[string]*models.GroupPolicy{}}
}

func validateDeviceGroupDocument(document deviceGroupDocument) error {
	if len(document.Groups) > deviceGroupLimit || len(document.DevicePolicies) > deviceGroupPolicyLimit {
		return errors.New("device group capacity reached")
	}
	members := 0
	policies := len(document.DevicePolicies)
	seen := map[string]bool{}
	if err := validateGroupPolicy(document.GlobalPolicy); err != nil {
		return err
	}
	for _, group := range document.Groups {
		if group == nil {
			return errors.New("invalid device group record")
		}
		if err := validateGroup(group); err != nil {
			return err
		}
		if seen[group.ID] {
			return errors.New("invalid device group record")
		}
		seen[group.ID] = true
		members += len(group.Members)
		if group.Policy != nil {
			policies++
		}
	}
	for deviceID, policy := range document.DevicePolicies {
		if !validDeviceIdentityKey(deviceID) || policy == nil {
			return errors.New("invalid device policy record")
		}
		if err := validateGroupPolicy(policy); err != nil {
			return err
		}
	}
	if members > deviceGroupMemberLimit || policies > deviceGroupPolicyLimit {
		return errors.New("device group capacity reached")
	}
	return nil
}

func documentDeviceGroupState(document deviceGroupDocument, raw []byte) deviceGroupState {
	return deviceGroupState{
		GlobalPolicy: document.GlobalPolicy, Groups: document.Groups,
		DevicePolicies: document.DevicePolicies, Version: versionDeviceGroups(raw),
	}
}

func versionDeviceGroups(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

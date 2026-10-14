package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	defaultPolicyEffectPath  = "/etc/quickstart/device-policy-effects-v1.json"
	defaultPolicyEffectLimit = 2048
)

type policyEffectRecord struct {
	Effect *models.PolicyEffect `json:"effect"`
}

type policyEffectDocument struct {
	SchemaVersion int                           `json:"schemaVersion"`
	Records       map[string]policyEffectRecord `json:"records"`
}

type policyEffectStore interface {
	Read(context.Context, string) (*models.PolicyEffect, bool, error)
	Write(context.Context, string, *models.PolicyEffect) error
}

type jsonPolicyEffectStore struct {
	mu      sync.Mutex
	path    string
	limit   int
	persist func(string, []byte) error
}

func newJSONPolicyEffectStore(path string) *jsonPolicyEffectStore {
	return &jsonPolicyEffectStore{path: path, limit: defaultPolicyEffectLimit, persist: persistClassificationOverrides}
}

func (store *jsonPolicyEffectStore) Read(_ context.Context, deviceID string) (*models.PolicyEffect, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	document, err := store.read()
	if err != nil {
		return nil, false, err
	}
	record, ok := document.Records[deviceID]
	return clonePolicyEffect(record.Effect), ok, nil
}

func (store *jsonPolicyEffectStore) Write(_ context.Context, deviceID string, effect *models.PolicyEffect) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	document, err := store.read()
	if err != nil {
		return err
	}
	if _, exists := document.Records[deviceID]; !exists && len(document.Records) >= store.limit {
		return errors.New("policy effect capacity reached")
	}
	document.Records[deviceID] = policyEffectRecord{Effect: clonePolicyEffect(effect)}
	raw, err := json.Marshal(document)
	if err != nil {
		return err
	}
	return store.persist(store.path, raw)
}

func (store *jsonPolicyEffectStore) read() (policyEffectDocument, error) {
	raw, err := os.ReadFile(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return policyEffectDocument{SchemaVersion: 1, Records: map[string]policyEffectRecord{}}, nil
	}
	if err != nil {
		return policyEffectDocument{}, err
	}
	var document policyEffectDocument
	if json.Unmarshal(raw, &document) != nil || document.SchemaVersion != 1 || document.Records == nil || len(document.Records) > store.limit {
		return policyEffectDocument{}, errors.New("invalid policy effect store")
	}
	return document, nil
}

type memoryPolicyEffectStore struct {
	mu      sync.Mutex
	records map[string]*models.PolicyEffect
}

func newMemoryPolicyEffectStore() *memoryPolicyEffectStore {
	return &memoryPolicyEffectStore{records: map[string]*models.PolicyEffect{}}
}

func (store *memoryPolicyEffectStore) Read(_ context.Context, deviceID string) (*models.PolicyEffect, bool, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	effect, ok := store.records[deviceID]
	return clonePolicyEffect(effect), ok, nil
}

func (store *memoryPolicyEffectStore) Write(_ context.Context, deviceID string, effect *models.PolicyEffect) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.records[deviceID] = clonePolicyEffect(effect)
	return nil
}

type leaseObservation struct {
	SourceAvailable bool
	Observed        bool
	ObservedAt      time.Time
}

type policyLeaseObserver interface {
	Observe(context.Context, string, time.Time) (leaseObservation, error)
}

type dnsmasqLeaseObserver struct{ path string }

func (observer *dnsmasqLeaseObserver) Observe(_ context.Context, mac string, since time.Time) (leaseObservation, error) {
	info, err := os.Stat(observer.path)
	if errors.Is(err, os.ErrNotExist) {
		return leaseObservation{}, nil
	}
	if err != nil {
		return leaseObservation{}, err
	}
	file, err := os.Open(observer.path)
	if err != nil {
		return leaseObservation{}, err
	}
	defer file.Close()
	result := leaseObservation{SourceAvailable: true}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && normalizeInventoryMAC(fields[1]) == normalizeInventoryMAC(mac) && !info.ModTime().Before(since) {
			result.Observed = true
			result.ObservedAt = info.ModTime().UTC()
			break
		}
	}
	return result, scanner.Err()
}

type PolicyEffectModule struct {
	store    policyEffectStore
	observer policyLeaseObserver
	now      func() time.Time
}

func NewPolicyEffectModule(store policyEffectStore, observer policyLeaseObserver, now func() time.Time) *PolicyEffectModule {
	return &PolicyEffectModule{store: store, observer: observer, now: now}
}

func NewDefaultPolicyEffectModule() *PolicyEffectModule {
	return NewPolicyEffectModule(newJSONPolicyEffectStore(defaultPolicyEffectPath), &dnsmasqLeaseObserver{path: "/tmp/dhcp.leases"}, time.Now)
}

func (module *PolicyEffectModule) RecordApplied(ctx context.Context, deviceID, targetID, version string, static *models.DeviceAddressPolicy) error {
	now := module.now().UTC().Format(time.RFC3339Nano)
	effect := &models.PolicyEffect{
		Desired:  &models.DesiredNetworkPolicy{TargetID: targetID, Static: cloneAddressPolicy(static), UpdatedAt: now},
		Applied:  &models.AppliedNetworkConfiguration{TargetID: targetID, ConfigVersion: version, State: "server_applied", AppliedAt: now},
		Observed: &models.ObservedNetworkEffect{State: "pending_renewal", Reason: "device_renewal_required", Source: "dnsmasq_lease"},
	}
	return module.store.Write(ctx, deviceID, effect)
}

func (module *PolicyEffectModule) RecordFailure(ctx context.Context, deviceID, desiredTarget, appliedTarget, version string, static *models.DeviceAddressPolicy, reason string) error {
	now := module.now().UTC().Format(time.RFC3339Nano)
	effect := &models.PolicyEffect{
		Desired:        &models.DesiredNetworkPolicy{TargetID: desiredTarget, Static: cloneAddressPolicy(static), UpdatedAt: now},
		Applied:        &models.AppliedNetworkConfiguration{TargetID: appliedTarget, ConfigVersion: version, State: "apply_failed"},
		Observed:       &models.ObservedNetworkEffect{State: "failed", Reason: reason},
		NeedsAttention: true, AttentionReason: "network_policy_apply_failed",
	}
	return module.store.Write(ctx, deviceID, effect)
}

func (module *PolicyEffectModule) Resolve(ctx context.Context, deviceID, mac, actualTarget, version string, static *models.DeviceAddressPolicy) *models.PolicyEffect {
	effect, ok, err := module.store.Read(ctx, deviceID)
	if err != nil {
		return unavailablePolicyEffect(actualTarget, version, static, "effect_store_unavailable")
	}
	if !ok || effect == nil || effect.Applied == nil || effect.Observed == nil {
		return unavailablePolicyEffect(actualTarget, version, static, "no_apply_observation")
	}
	if effect.Applied.State != "server_applied" || effect.Observed.State == "lease_observed_unverifiable" {
		return effect
	}
	appliedAt, parseErr := time.Parse(time.RFC3339Nano, effect.Applied.AppliedAt)
	if parseErr != nil {
		effect.Observed = &models.ObservedNetworkEffect{State: "observation_error", Reason: "invalid_apply_timestamp", Source: "dnsmasq_lease"}
		effect.NeedsAttention, effect.AttentionReason = true, "observation_failed"
		return effect
	}
	observation, observeErr := module.observer.Observe(ctx, mac, appliedAt)
	previous := effect.Observed.State + ":" + effect.Observed.Reason
	switch {
	case observeErr != nil:
		effect.Observed = &models.ObservedNetworkEffect{State: "observation_error", Reason: "lease_read_failed", Source: "dnsmasq_lease"}
		effect.NeedsAttention, effect.AttentionReason = true, "observation_failed"
	case !observation.SourceAvailable:
		effect.Observed = &models.ObservedNetworkEffect{State: "unverifiable", Reason: "lease_source_unavailable", Source: "dnsmasq_lease"}
		effect.NeedsAttention, effect.AttentionReason = false, ""
	case observation.Observed:
		effect.Observed = &models.ObservedNetworkEffect{State: "lease_observed_unverifiable", Reason: "terminal_gateway_and_dns_unverifiable", Source: "dnsmasq_lease", ObservedAt: observation.ObservedAt.Format(time.RFC3339Nano)}
		effect.NeedsAttention, effect.AttentionReason = false, ""
	default:
		effect.Observed = &models.ObservedNetworkEffect{State: "pending_renewal", Reason: "device_renewal_required", Source: "dnsmasq_lease"}
		effect.NeedsAttention, effect.AttentionReason = true, "waiting_for_device_renewal"
	}
	if current := effect.Observed.State + ":" + effect.Observed.Reason; current != previous {
		_ = module.store.Write(ctx, deviceID, effect)
	}
	return effect
}

func unavailablePolicyEffect(targetID, version string, static *models.DeviceAddressPolicy, reason string) *models.PolicyEffect {
	return &models.PolicyEffect{
		Desired:  &models.DesiredNetworkPolicy{TargetID: targetID, Static: cloneAddressPolicy(static)},
		Applied:  &models.AppliedNetworkConfiguration{TargetID: targetID, ConfigVersion: version, State: "server_configuration_observed"},
		Observed: &models.ObservedNetworkEffect{State: "unverifiable", Reason: reason},
	}
}

func cloneAddressPolicy(value *models.DeviceAddressPolicy) *models.DeviceAddressPolicy {
	if value == nil {
		return &models.DeviceAddressPolicy{}
	}
	copy := *value
	return &copy
}

func clonePolicyEffect(value *models.PolicyEffect) *models.PolicyEffect {
	if value == nil {
		return nil
	}
	raw, _ := json.Marshal(value)
	var result models.PolicyEffect
	_ = json.Unmarshal(raw, &result)
	return &result
}

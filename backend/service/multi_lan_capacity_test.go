package service

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type capacityPolicyStore struct {
	*fakeDevicePolicyStore
	allowed map[string]bool
}

func (store *capacityPolicyStore) Get(ctx context.Context, id string) (*models.DevicePolicy, error) {
	policy, err := store.fakeDevicePolicyStore.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	policy.DeviceID = id
	if value, ok := store.allowed[id]; ok {
		policy.Access.NetworkAccess = value
	} else {
		policy.Access.NetworkAccess = true
	}
	return policy, nil
}
func (store *capacityPolicyStore) Apply(_ context.Context, request *models.DevicePolicyApplyRequest, _ *models.DevicePolicy) error {
	store.applyCalls++
	store.allowed[request.DeviceID] = request.Access.NetworkAccess
	return nil
}

func TestMultiLAN256QuotasShareDurableUsageCheckpoint(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	writes := 0
	store := newTrafficInsightsFileStore(filepath.Join(t.TempDir(), "history.json"))
	store.persist = func(path string, raw []byte) error { writes++; return persistClassificationOverrides(path, raw) }
	policyStore := &capacityPolicyStore{fakeDevicePolicyStore: availablePolicyStore(), allowed: map[string]bool{}}
	module := NewTrafficInsightsModule(store, newDevicePolicyModuleForTest(policyStore))
	module.now = func() time.Time { return now }
	if err := module.load(); err != nil {
		t.Fatal(err)
	}
	response := &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{Items: []*models.DeviceTrafficItem{}}}
	for i := 0; i < 256; i++ {
		id := fmt.Sprintf("mac:02:00:00:00:%02x:%02x", i/256, i%256)
		module.document.Quotas[id] = &models.TrafficQuota{DeviceID: id, Enabled: true, Period: "daily", LimitBytes: 100, Action: "block"}
		module.document.Baselines[id] = trafficCounterBaseline{}
		response.Result.Items = append(response.Result.Items, &models.DeviceTrafficItem{DeviceID: id, State: "ready", UploadBytes: 120})
	}
	if err := module.Record(context.Background(), response); err != nil {
		t.Fatal(err)
	}
	if writes != 2 || policyStore.applyCalls != 256 {
		t.Fatalf("writes=%d applies=%d; want two durable checkpoints and 256 independent blocks", writes, policyStore.applyCalls)
	}
	persisted, _, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	for id, runtime := range persisted.QuotaRuntime {
		if !runtime.Blocked || policyStore.allowed[id] {
			t.Fatalf("device %s lost enforcement", id)
		}
	}
	now = now.Add(24 * time.Hour)
	if err := module.Record(context.Background(), response); err != nil {
		t.Fatal(err)
	}
	if writes > 4 || policyStore.applyCalls != 512 {
		t.Fatalf("period reset writes=%d applies=%d", writes, policyStore.applyCalls)
	}
}

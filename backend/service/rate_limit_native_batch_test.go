package service

import (
	"context"
	"encoding/json"
	"github.com/istoreos/quickstart/backend/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMultiLANNativeGroupUsesOneRevisionAndPreservesOtherOwners(t *testing.T) {
	owner := nativePolicyOwner{Product: "quickstart", Instance: "local-router"}
	external := nativeStoredPolicy{ID: "external", Owner: nativePolicyOwner{Product: "other", Instance: "app"}, Device: nativeDeviceIdentity{Kind: "mac", Value: "02:00:00:00:00:77"}, RateLimit: &nativeRateLimit{Enabled: true, UploadBitsPerSecond: 1_000_000, DownloadBitsPerSecond: 1_000_000}, State: "effective"}
	snapshot := nativePolicySnapshot{SchemaVersion: 1, Policies: []nativeStoredPolicy{external, {ID: "old", Owner: owner, Device: nativeDeviceIdentity{Kind: "mac", Value: "02:00:00:00:00:55"}, RateLimit: &nativeRateLimit{Enabled: true, UploadBitsPerSecond: 5_000_000, DownloadBitsPerSecond: 5_000_000}, State: "effective"}}}
	applies := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data any
		switch r.URL.Path {
		case "/api/v1/system/capabilities":
			data = map[string]any{"product": "quickstart-netpolicy", "interfaceVersion": "1", "engine": "quickstart-native"}
		case "/api/v1/system/health":
			data = map[string]any{"state": "healthy"}
		case "/api/v1/policies":
			data = snapshot
		case "/api/v1/policies/plan":
			data = nativePolicyPlan{ID: "plan", BaseRevision: snapshot.Revision, Changes: []nativePolicyChange{}, Warnings: []string{}}
		case "/api/v1/policies/apply":
			var input struct {
				Request nativePolicyMutation `json:"request"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Error(err)
			}
			if input.Request.ExpectedRevision != snapshot.Revision {
				t.Error("stale batch revision")
			}
			snapshot.Revision++
			snapshot.Policies = []nativeStoredPolicy{external}
			for _, policy := range input.Request.Policies {
				snapshot.Policies = append(snapshot.Policies, nativeStoredPolicy{Owner: input.Request.Owner, Device: policy.Device, RateLimit: policy.RateLimit, Quota: policy.Quota, State: "effective"})
			}
			applies++
			data = snapshot
		case "/api/v1/policies/verify":
			data = snapshot
		default:
			t.Errorf("unexpected call %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": data})
	}))
	defer server.Close()
	provider := NewNativeRateLimitProvider(server.URL, server.Client()).(*nativeRateLimitProvider)
	provider.offload = nil
	module := NewRateLimitModule(provider, &fakeRateLimitRouteResolver{route: rateLimitRoute{Local: true}})
	changes := []groupBatchRestrictionChange{}
	for _, mac := range []string{"02:00:00:00:00:55", "02:00:00:00:00:66"} {
		changes = append(changes, groupBatchRestrictionChange{current: &models.DevicePolicy{MAC: mac}, request: &models.DevicePolicyApplyRequest{Kind: "speed", Speed: &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 20}}})
	}
	rollback, err := applyProviderGroupSpeedBatch(context.Background(), module, changes)
	if err != nil || applies != 1 || len(snapshot.Policies) != 3 {
		t.Fatalf("batch not atomic: applies=%d snapshot=%+v err=%v", applies, snapshot, err)
	}
	if err := rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if applies != 2 || len(snapshot.Policies) != 2 || snapshot.Policies[0].Owner != external.Owner || !strings.EqualFold(snapshot.Policies[1].Device.Value, "02:00:00:00:00:55") || snapshot.Policies[1].RateLimit.UploadBitsPerSecond != 5_000_000 {
		t.Fatalf("rollback changed another owner: %+v", snapshot)
	}
}

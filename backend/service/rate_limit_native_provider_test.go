package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/istoreos/quickstart/backend/models"
)

func nativeTestServer(t *testing.T, interfaceVersion string, conflict bool, calls *[]string) *httptest.Server {
	t.Helper()
	snapshot := nativePolicySnapshot{SchemaVersion: 1, Revision: 4, Policies: []nativeStoredPolicy{{
		ID: "one", Owner: nativePolicyOwner{Product: "quickstart", Instance: "local-router"},
		Device:    nativeDeviceIdentity{Kind: "mac", Value: "AA:BB:CC:DD:EE:20"},
		RateLimit: &nativeRateLimit{Enabled: true, UploadBitsPerSecond: 5_000_000, DownloadBitsPerSecond: 30_000_000}, State: "effective",
	}}}
	return httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		*calls = append(*calls, request.Method+" "+request.URL.Path)
		response.Header().Set("Content-Type", "application/json")
		write := func(value any) {
			_ = json.NewEncoder(response).Encode(map[string]any{"status": "success", "data": value})
		}
		switch request.URL.Path {
		case "/api/v1/system/capabilities":
			write(map[string]any{"product": "quickstart-netpolicy", "interfaceVersion": interfaceVersion, "engine": "quickstart-native"})
		case "/api/v1/system/health":
			write(map[string]any{"state": "healthy"})
		case "/api/v1/policies":
			write(snapshot)
		case "/api/v1/policies/plan":
			if conflict {
				response.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(response).Encode(map[string]any{"status": "error", "error": map[string]any{"code": "revision_conflict", "message": "changed"}})
				return
			}
			var mutation nativePolicyMutation
			_ = json.NewDecoder(request.Body).Decode(&mutation)
			if mutation.ExpectedRevision != 4 || len(mutation.Policies) != 1 || mutation.Policies[0].RateLimit.UploadBitsPerSecond != 10_000_000 {
				t.Fatalf("unexpected mutation: %#v", mutation)
			}
			write(nativePolicyPlan{ID: "plan", BaseRevision: 4, NextRevision: 5})
		case "/api/v1/policies/apply":
			snapshot.Revision = 5
			write(snapshot)
		case "/api/v1/policies/verify":
			write(snapshot)
		default:
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
	}))
}

func TestNativeProviderInspectsVerifiedMACPolicy(t *testing.T) {
	calls := []string{}
	server := nativeTestServer(t, "1", false, &calls)
	defer server.Close()
	provider := NewNativeRateLimitProvider(server.URL, server.Client()).(*nativeRateLimitProvider)
	provider.offload = nil
	observation, err := provider.Inspect(context.Background(), rateLimitTarget{MAC: "aa:bb:cc:dd:ee:20"})
	if err != nil || !observation.Verified || observation.Policy.UploadSpeed != 5 || observation.Policy.DownloadSpeed != 30 {
		t.Fatalf("unexpected observation: %#v, %v", observation, err)
	}
}

func TestNativeProviderAppliesPlanAndVerifies(t *testing.T) {
	calls := []string{}
	server := nativeTestServer(t, "1", false, &calls)
	defer server.Close()
	provider := NewNativeRateLimitProvider(server.URL, server.Client()).(*nativeRateLimitProvider)
	provider.offload = nil
	if err := provider.Apply(context.Background(), rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"}, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 100}); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(calls, ",")
	for _, expected := range []string{"POST /api/v1/policies/plan", "POST /api/v1/policies/apply", "POST /api/v1/policies/verify"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %s in %s", expected, joined)
		}
	}
}

func TestNativeProviderReportsIncompatibleAndRevisionConflict(t *testing.T) {
	for _, test := range []struct {
		version, contains string
		conflict          bool
	}{{"2", "incompatible", false}, {"1", "revision_conflict", true}} {
		calls := []string{}
		server := nativeTestServer(t, test.version, test.conflict, &calls)
		provider := NewNativeRateLimitProvider(server.URL, server.Client()).(*nativeRateLimitProvider)
		provider.offload = nil
		err := provider.Apply(context.Background(), rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"}, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 100})
		server.Close()
		if err == nil || !strings.Contains(err.Error(), test.contains) {
			t.Fatalf("expected %s, got %v", test.contains, err)
		}
	}
}

func TestNativeProviderFailsClosedWhenOffline(t *testing.T) {
	provider := NewNativeRateLimitProvider("http://127.0.0.1:1", &http.Client{}).(*nativeRateLimitProvider)
	provider.offload = nil
	err := provider.Apply(context.Background(), rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"}, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 100})
	if err == nil || !strings.Contains(err.Error(), "netpolicy_unavailable") {
		t.Fatalf("expected unavailable, got %v", err)
	}
}

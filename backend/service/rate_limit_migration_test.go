package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func fullWeekBandixRule(mac string, upload, download int64) bandixScheduleRule {
	rule := bandixScheduleRule{ID: "rule-1", MAC: mac, UploadBytes: upload, DownloadBytes: download}
	rule.TimeSlot.Start = "00:00"
	rule.TimeSlot.End = "23:59"
	rule.TimeSlot.Days = []int{1, 2, 3, 4, 5, 6, 7}
	return rule
}

func TestRateLimitMigrationPlanConvertsExactBitsWithoutWriting(t *testing.T) {
	rules := []bandixScheduleRule{fullWeekBandixRule("AA:BB:CC:DD:EE:20", 1_250_001, 12_500_001)}
	plan := buildRateLimitMigrationPlan(rules)
	if !plan.CanApply || plan.ConvertibleCount != 1 || plan.Items[0].UploadBitsPerSecond != 10_000_008 || plan.Items[0].DownloadBitsPerSecond != 100_000_008 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if plan.Version == "" || len(plan.RequiredActions) != 3 {
		t.Fatalf("missing audit metadata: %#v", plan)
	}
}

func TestRateLimitMigrationDoesNotDropUnsupportedOrConflictingRules(t *testing.T) {
	duplicateOne := fullWeekBandixRule("AA:BB:CC:DD:EE:20", 1, 2)
	duplicateTwo := fullWeekBandixRule("AA:BB:CC:DD:EE:20", 3, 4)
	unknown := fullWeekBandixRule("AA:BB:CC:DD:EE:21", 5, 6)
	unknown.UnknownFields = []string{"future_mode"}
	scheduled := fullWeekBandixRule("AA:BB:CC:DD:EE:22", 7, 8)
	scheduled.TimeSlot.Start = "08:00"
	plan := buildRateLimitMigrationPlan([]bandixScheduleRule{duplicateOne, duplicateTwo, unknown, scheduled})
	if plan.CanApply || plan.ConflictCount != 2 || plan.UnsupportedCount != 2 || len(plan.Items) != 4 {
		t.Fatalf("unexpected classification: %#v", plan)
	}
}

func TestBandixDecoderRetainsUnknownFieldsForMigrationReview(t *testing.T) {
	var rule bandixScheduleRule
	if err := json.Unmarshal([]byte(`{"id":1,"mac":"AA:BB:CC:DD:EE:20","future_mode":"burst"}`), &rule); err != nil {
		t.Fatal(err)
	}
	if len(rule.UnknownFields) != 1 || rule.UnknownFields[0] != "future_mode" {
		t.Fatalf("unknown fields = %v", rule.UnknownFields)
	}
}

func migrationNativeServer(t *testing.T, calls *[]string) *httptest.Server {
	t.Helper()
	snapshot := nativePolicySnapshot{SchemaVersion: 1, Revision: 3, Policies: []nativeStoredPolicy{{
		ID: "old", Owner: nativePolicyOwner{Product: "quickstart", Instance: "local-router"},
		Device:    nativeDeviceIdentity{Kind: "mac", Value: "AA:BB:CC:DD:EE:99"},
		RateLimit: &nativeRateLimit{Enabled: true, UploadBitsPerSecond: 1_000_000, DownloadBitsPerSecond: 2_000_000}, State: "effective",
	}}}
	return httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		*calls = append(*calls, request.Method+" "+request.URL.Path)
		response.Header().Set("Content-Type", "application/json")
		write := func(value any) {
			_ = json.NewEncoder(response).Encode(map[string]any{"status": "success", "data": value})
		}
		switch request.URL.Path {
		case "/api/v1/system/capabilities":
			write(map[string]any{"product": "quickstart-netpolicy", "interfaceVersion": "1", "engine": "quickstart-native"})
		case "/api/v1/system/health":
			write(map[string]any{"state": "healthy"})
		case "/api/v1/policies":
			write(snapshot)
		case "/api/v1/policies/plan":
			var mutation nativePolicyMutation
			_ = json.NewDecoder(request.Body).Decode(&mutation)
			write(nativePolicyPlan{ID: "plan", BaseRevision: mutation.ExpectedRevision, NextRevision: mutation.ExpectedRevision + 1})
		case "/api/v1/policies/apply":
			var payload struct {
				Request nativePolicyMutation `json:"request"`
			}
			_ = json.NewDecoder(request.Body).Decode(&payload)
			snapshot.Revision++
			snapshot.Policies = nil
			for index, input := range payload.Request.Policies {
				snapshot.Policies = append(snapshot.Policies, nativeStoredPolicy{
					ID: "policy-" + string(rune('a'+index)), Owner: payload.Request.Owner, Device: input.Device,
					RateLimit: input.RateLimit, Quota: input.Quota, State: "effective",
				})
			}
			write(snapshot)
		case "/api/v1/policies/verify":
			write(snapshot)
		default:
			t.Fatalf("unexpected path %s", request.URL.Path)
		}
	}))
}

func TestRateLimitMigrationAppliesThenSwitchesAndCanRollback(t *testing.T) {
	directory := t.TempDir()
	preference := filepath.Join(directory, "provider")
	if err := os.WriteFile(preference, []byte("bandix\n"), 0600); err != nil {
		t.Fatal(err)
	}
	calls := []string{}
	server := migrationNativeServer(t, &calls)
	defer server.Close()
	native := NewNativeRateLimitProvider(server.URL, server.Client()).(*nativeRateLimitProvider)
	native.offload = nil
	module := &RateLimitMigrationModule{
		native: native, preferencePath: preference, receiptPath: filepath.Join(directory, "receipt.json"),
		now: func() time.Time { return time.Unix(100, 0) },
	}
	plan := buildRateLimitMigrationPlan([]bandixScheduleRule{fullWeekBandixRule("AA:BB:CC:DD:EE:20", 1_250_000, 12_500_000)})
	result, err := module.Apply(context.Background(), &models.RateLimitMigrationRequest{ExpectedVersion: plan.Version, Plan: plan})
	if err != nil || result.Result.Error != nil || !result.Result.Changed || result.Result.MigrationID == "" {
		t.Fatalf("unexpected apply: %#v %v", result, err)
	}
	if selected := strings.TrimSpace(string(mustRead(t, preference))); selected != nativePolicyProviderName {
		t.Fatalf("provider = %s", selected)
	}
	joined := strings.Join(calls, ",")
	if strings.Index(joined, "/plan") > strings.Index(joined, "/apply") || strings.Index(joined, "/apply") > strings.Index(joined, "/verify") {
		t.Fatalf("incorrect transaction order: %s", joined)
	}
	rolledBack, err := module.Rollback(context.Background(), &models.RateLimitMigrationRollbackRequest{MigrationID: result.Result.MigrationID})
	if err != nil || rolledBack.Result.Error != nil || !rolledBack.Result.RolledBack {
		t.Fatalf("unexpected rollback: %#v %v", rolledBack, err)
	}
	if selected := strings.TrimSpace(string(mustRead(t, preference))); selected != "bandix" {
		t.Fatalf("provider after rollback = %s", selected)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

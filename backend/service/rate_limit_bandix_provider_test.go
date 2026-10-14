package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

func TestBandixProviderInspectsBoundedMACRule(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != bandixSchedulePath {
			t.Fatalf("unexpected request %s %s", request.Method, request.URL.Path)
		}
		_, _ = response.Write([]byte(`{"status":"success","data":{"limits":[{"id":"one","mac":"AA:BB:CC:DD:EE:20","time_slot":{"start":"00:00","end":"23:59","days":[1,2,3,4,5,6,7]},"wan_tx_rate_limit":625000,"wan_rx_rate_limit":3750000}]},"message":null}`))
	}))
	defer server.Close()
	provider := NewBandixRateLimitProvider(server.URL, server.Client()).(*bandixRateLimitProvider)
	provider.runtime = nil
	result, err := provider.Inspect(context.Background(), rateLimitTarget{MAC: "aa:bb:cc:dd:ee:20"})
	if err != nil || !result.Verified || result.Policy.UploadSpeed != 5 || result.Policy.DownloadSpeed != 30 {
		t.Fatalf("unexpected observation: %#v, %v", result, err)
	}
}

func TestBandixProviderCreatesAndDeletesFullWeekRule(t *testing.T) {
	requests := make([]string, 0, 3)
	rules := `[]`
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			_, _ = response.Write([]byte(rules))
			return
		}
		requests = append(requests, request.Method)
		bodyBytes, _ := io.ReadAll(request.Body)
		body := string(bodyBytes)
		if request.Method == http.MethodPost {
			if !strings.Contains(body, `"end":"23:59"`) || !strings.Contains(body, `"wan_tx_rate_limit":1250000`) {
				t.Fatalf("unexpected create payload: %s", body)
			}
			rules = `[{"id":"managed","mac":"AA:BB:CC:DD:EE:20","wan_tx_rate_limit":1250000,"wan_rx_rate_limit":12500000}]`
		}
		if request.Method == http.MethodDelete && !strings.Contains(body, `"id":"managed"`) {
			t.Fatalf("unexpected delete payload: %s", body)
		}
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	provider := NewBandixRateLimitProvider(server.URL, server.Client())
	target := rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"}
	if err := provider.Apply(context.Background(), target, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 10, DownloadSpeed: 100}); err != nil {
		t.Fatal(err)
	}
	if err := provider.Apply(context.Background(), target, models.DeviceSpeedPolicy{}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(requests, ",") != "POST,DELETE" {
		t.Fatalf("requests = %v", requests)
	}
}

func TestBandixProviderUpdatesExistingRule(t *testing.T) {
	method := ""
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			_, _ = response.Write([]byte(`{"data":[{"id":7,"mac":"AA:BB:CC:DD:EE:20"}]}`))
			return
		}
		method = request.Method
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	provider := NewBandixRateLimitProvider(server.URL, server.Client())
	if err := provider.Apply(context.Background(), rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"}, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 1, DownloadSpeed: 2}); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodPut {
		t.Fatalf("method = %s", method)
	}
}

func TestBandixProviderRejectsUnboundedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = fmt.Fprint(response, strings.Repeat("x", bandixMaximumResponseBytes+1))
	}))
	defer server.Close()
	provider := NewBandixRateLimitProvider(server.URL, server.Client())
	_, err := provider.Inspect(context.Background(), rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"})
	if err == nil || !strings.Contains(err.Error(), "too_large") {
		t.Fatalf("expected bounded read error, got %v", err)
	}
}

func TestBandixProviderHonorsHTTPTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) { time.Sleep(50 * time.Millisecond) }))
	defer server.Close()
	provider := NewBandixRateLimitProvider(server.URL, &http.Client{Timeout: 5 * time.Millisecond})
	_, err := provider.Inspect(context.Background(), rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"})
	if err == nil || !strings.Contains(err.Error(), "bandix_unavailable") {
		t.Fatalf("expected timeout, got %v", err)
	}
}

func TestBandixProviderDoesNotTreatHTTP200ErrorAsSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			_, _ = response.Write([]byte(`[]`))
			return
		}
		_, _ = response.Write([]byte(`{"status":"error","data":null,"message":"invalid rule"}`))
	}))
	defer server.Close()
	provider := NewBandixRateLimitProvider(server.URL, server.Client())
	err := provider.Apply(context.Background(), rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"}, models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 1, DownloadSpeed: 2})
	if err == nil || !strings.Contains(err.Error(), "bandix_rejected") {
		t.Fatalf("expected provider rejection, got %v", err)
	}
}

func TestRateLimitModuleUsesMACIdentityForBandix(t *testing.T) {
	provider := &fakeRateLimitProvider{}
	module := NewRateLimitModule(provider, &fakeRateLimitRouteResolver{route: rateLimitRoute{TargetID: "self", Local: true}})
	providerIdentity := &fakeMACRateLimitProvider{fakeRateLimitProvider: provider}
	module.providers = map[string]rateLimitProvider{"mac": providerIdentity}
	module.selected = func(context.Context) string { return "mac" }
	if err := module.Apply(context.Background(), rateLimitTarget{MAC: "AA:BB:CC:DD:EE:20"}, models.DeviceSpeedPolicy{Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if provider.applyCalls != 1 {
		t.Fatalf("apply calls = %d", provider.applyCalls)
	}
}

func TestProviderGroupBatchReturnsCompensatingRollback(t *testing.T) {
	provider := &fakeRateLimitProvider{}
	module := NewRateLimitModule(provider, &fakeRateLimitRouteResolver{route: rateLimitRoute{TargetID: "self", Local: true}})
	changes := []groupBatchRestrictionChange{
		{current: &models.DevicePolicy{DeviceID: "one", MAC: "AA:BB:CC:DD:EE:01", CurrentIPv4: "192.168.1.21", Static: &models.DeviceStaticPolicy{AssignedIP: "192.168.1.21"}, Speed: &models.DeviceSpeedPolicy{}}, request: &models.DevicePolicyApplyRequest{Kind: "speed", Speed: &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 5, DownloadSpeed: 30}}},
		{current: &models.DevicePolicy{DeviceID: "two", MAC: "AA:BB:CC:DD:EE:02", CurrentIPv4: "192.168.1.22", Static: &models.DeviceStaticPolicy{AssignedIP: "192.168.1.22"}, Speed: &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 2, DownloadSpeed: 10}}, request: &models.DevicePolicyApplyRequest{Kind: "speed", Speed: &models.DeviceSpeedPolicy{Enabled: true, UploadSpeed: 5, DownloadSpeed: 30}}},
	}
	rollback, err := applyProviderGroupSpeedBatch(context.Background(), module, changes)
	if err != nil || provider.applyCalls != 2 {
		t.Fatalf("apply calls=%d err=%v", provider.applyCalls, err)
	}
	if err := rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	if provider.applyCalls != 4 {
		t.Fatalf("expected two compensating writes, calls=%d", provider.applyCalls)
	}
}

type fakeMACRateLimitProvider struct{ *fakeRateLimitProvider }

func (*fakeMACRateLimitProvider) Name() string         { return "mac" }
func (*fakeMACRateLimitProvider) IdentityKind() string { return "mac" }
func (*fakeMACRateLimitProvider) SupportsIPv6() bool   { return true }

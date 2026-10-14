package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

type capturingWebhookSender struct {
	events chan *models.NetworkAuditEvent
}

func (sender *capturingWebhookSender) Send(_ context.Context, _ string, event *models.NetworkAuditEvent) error {
	copyValue := *event
	sender.events <- &copyValue
	return nil
}

type retryWebhookSender struct {
	attempts int
	events   chan *models.NetworkAuditEvent
}

type blockingWebhookSender struct {
	entered chan struct{}
	release chan struct{}
}

func (sender *blockingWebhookSender) Send(_ context.Context, _ string, _ *models.NetworkAuditEvent) error {
	select {
	case sender.entered <- struct{}{}:
	default:
	}
	<-sender.release
	return nil
}

func (sender *retryWebhookSender) Send(_ context.Context, _ string, event *models.NetworkAuditEvent) error {
	sender.attempts++
	copyValue := *event
	sender.events <- &copyValue
	if sender.attempts == 1 {
		return errors.New("temporary failure")
	}
	return nil
}

func receiveWebhookEvent(t *testing.T, sender *capturingWebhookSender) *models.NetworkAuditEvent {
	t.Helper()
	select {
	case event := <-sender.events:
		return event
	case <-time.After(time.Second):
		t.Fatal("webhook event was not delivered")
		return nil
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func advancedTestInventory(t *testing.T) (*DeviceInventoryModule, string) {
	t.Helper()
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	source := &fakeDeviceInventorySource{snapshots: []deviceInventorySourceSnapshot{{ARP: []deviceInventoryObservation{{MAC: "AA:BB:CC:DD:EE:01", IPv4: "192.168.100.20", Online: true}}}}}
	inventory := newTestDeviceInventory(t, source, &now, 100)
	response, err := inventory.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return inventory, response.Result.Devices[0].DeviceID
}

func TestManagementProbeAllowsInventoryLANOnly(t *testing.T) {
	inventory, deviceID := advancedTestInventory(t)
	audit := NewNetworkAuditModule(filepath.Join(t.TempDir(), "audit.json"), nil)
	module := NewManagementProbeModule(inventory, audit)
	module.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.Host != "192.168.100.20:80" {
			t.Fatalf("unexpected target %s", request.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ignored")), Header: http.Header{}}, nil
	}), Timeout: managementProbeTimeout}
	response, err := module.Probe(context.Background(), &models.ManagementProbeRequest{DeviceID: deviceID, Address: "192.168.100.20", Scheme: "http", Port: 80})
	if err != nil || response.Result.Error != nil || !response.Result.Reachable || response.Result.StatusCode != 200 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	for _, request := range []*models.ManagementProbeRequest{
		{DeviceID: deviceID, Address: "8.8.8.8", Scheme: "http", Port: 80},
		{DeviceID: deviceID, Address: "192.168.100.20", Scheme: "http", Port: 22},
		{DeviceID: "mac:other", Address: "192.168.100.20", Scheme: "http", Port: 80},
		{DeviceID: deviceID, Address: "192.168.100.20", Scheme: "file", Port: 80},
	} {
		rejected, _ := module.Probe(context.Background(), request)
		if rejected.Result.Error == nil || rejected.Result.Error.Code != "validation_failed" {
			t.Fatalf("unsafe probe accepted: %#v", request)
		}
	}
	if cap(module.probeSlots) != 4 {
		t.Fatalf("probe concurrency=%d", cap(module.probeSlots))
	}
}

func TestManagementProbeDefaultClientDoesNotFollowRedirects(t *testing.T) {
	module := NewManagementProbeModule(nil, nil)
	if module.client.CheckRedirect == nil {
		t.Fatal("management probe must define a redirect policy")
	}
	request, _ := http.NewRequest(http.MethodHead, "http://192.168.1.1/next", nil)
	if err := module.client.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect policy returned %v", err)
	}
}

func TestStaticAddressConflictIsAuditedWithoutLeakingAddress(t *testing.T) {
	t.Parallel()

	store := availablePolicyStore()
	store.rules = &models.DevicePolicyRulesResult{Static: []*models.LANStaticAssigned{{AssignedIP: "192.168.100.50", AssignedMac: "AA:BB:CC:DD:EE:01"}}}
	audit := NewNetworkAuditModule(filepath.Join(t.TempDir(), "audit.json"), nil)
	backend := &ServiceBackend{devicePolicy: newDevicePolicyModuleForTest(store), networkAudit: audit}
	request := httptest.NewRequest(http.MethodPost, "/device-policy", strings.NewReader(`{"deviceId":"mac:aa:bb:cc:dd:ee:ff","kind":"static","static":{"enabled":true,"assignedIP":"192.168.100.50","bindIP":true}}`))
	response, err := backend.PostDevicePolicyV2(context.Background(), request)
	if err != nil || response.Result == nil || response.Result.Error == nil || response.Result.Error.Code != "conflict" {
		t.Fatalf("response=%#v err=%v", response, err)
	}
	events, err := audit.List(store.policy.DeviceID, 10)
	if err != nil || len(events.Result.Events) != 1 {
		t.Fatalf("events=%#v err=%v", events, err)
	}
	event := events.Result.Events[0]
	if event.Action != "address_conflict" || event.State != "rejected" || event.Reason != "" {
		t.Fatalf("unexpected conflict audit event: %#v", event)
	}
}

func TestNetworkAuditIsBoundedFilteredAndDNSPrivate(t *testing.T) {
	module := NewNetworkAuditModule(filepath.Join(t.TempDir(), "audit.json"), nil)
	module.persist = func(string, []byte) error { return nil }
	for index := 0; index < networkAuditLimit+20; index++ {
		resource, reason := "device_policy", ""
		if index == networkAuditLimit+19 {
			resource, reason = "dns", "secret.example"
		}
		module.Record("mac:a", resource, "policy_changed", "success", reason)
	}
	response, err := module.List("mac:a", networkAuditLimit)
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Result.Events) != networkAuditLimit || response.Result.Events[0].Reason != "" || response.Result.Events[0].Resource != "dns" {
		t.Fatalf("audit result=%#v", response.Result.Events[:1])
	}
	if response.Result.Webhook.Enabled || len(module.queue) != 0 || cap(module.queue) != webhookQueueLimit {
		t.Fatalf("unsafe webhook defaults: %#v", response.Result.Webhook)
	}
}

func TestNetworkAuditTracksNewOnlineOfflineAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	module := NewNetworkAuditModule(path, nil)
	response := func(online bool) *models.DeviceInventoryResponse {
		return &models.DeviceInventoryResponse{Result: &models.DeviceInventoryResult{Devices: []*models.DeviceInventoryItem{{DeviceID: "mac:a", Online: online}}}}
	}
	module.ObserveInventory(response(true))
	module.ObserveInventory(response(false))
	restarted := NewNetworkAuditModule(path, nil)
	restarted.ObserveInventory(response(true))
	events, err := restarted.List("mac:a", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events.Result.Events) != 3 || events.Result.Events[0].Action != "device_online" || events.Result.Events[1].Action != "device_offline" || events.Result.Events[2].Action != "new_device" {
		t.Fatalf("unexpected lifecycle events: %#v", events.Result.Events)
	}
}

func TestNetworkAuditDeviceLifecycleStateIsBounded(t *testing.T) {
	module := NewNetworkAuditModule(filepath.Join(t.TempDir(), "audit.json"), nil)
	module.persist = func(string, []byte) error { return nil }
	for index := 0; index < networkAuditDeviceLimit+20; index++ {
		module.ObserveInventory(&models.DeviceInventoryResponse{Result: &models.DeviceInventoryResult{Devices: []*models.DeviceInventoryItem{{DeviceID: fmt.Sprintf("mac:%04d", index), Online: true}}}})
	}
	if len(module.document.SeenDevices) != networkAuditDeviceLimit || len(module.document.DeviceOnline) != networkAuditDeviceLimit || len(module.document.DeviceOrder) != networkAuditDeviceLimit {
		t.Fatalf("unbounded audit state: seen=%d online=%d order=%d", len(module.document.SeenDevices), len(module.document.DeviceOnline), len(module.document.DeviceOrder))
	}
}

func TestNetworkAuditRecordsObservedFloatingGatewayHolderChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	response := func(holder string) *models.FloatingGatewayResponse {
		return &models.FloatingGatewayResponse{Result: &models.FloatingGatewayResult{
			Config: &models.FloatingGatewayConfig{Enabled: true},
			Status: &models.FloatingGatewayStatus{Capability: "available", Holder: holder},
		}}
	}
	module := NewNetworkAuditModule(path, nil)
	module.ObserveFloatingGateway(response("local"))
	restarted := NewNetworkAuditModule(path, nil)
	restarted.ObserveFloatingGateway(response("unknown"))
	events, err := restarted.List("", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events.Result.Events) != 1 || events.Result.Events[0].Action != "gateway_failover" || events.Result.Events[0].State != "unknown" {
		t.Fatalf("unexpected failover events: %#v", events.Result.Events)
	}
}

func TestWebhookValidationAndRedactedReadback(t *testing.T) {
	module := NewNetworkAuditModule(filepath.Join(t.TempDir(), "audit.json"), nil)
	module.persist = func(string, []byte) error { return nil }
	bad, _ := module.SetWebhook(&models.WebhookConfig{Enabled: true, URL: "http://user:pass@example.com/hook", Events: []string{"new_device"}})
	if bad.Result.Error == nil {
		t.Fatal("credential URL accepted")
	}
	response, err := module.SetWebhook(&models.WebhookConfig{Enabled: true, URL: "https://example.com/hook", Events: []string{"new_device", "new_device"}})
	if err != nil || response.Result.Error != nil || response.Result.Webhook.URL != "configured" || len(response.Result.Webhook.Events) != 1 {
		t.Fatalf("response=%#v err=%v", response, err)
	}
}

func TestWebhookUsesStablePseudonymousDeviceIdentifiers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.json")
	firstSender := &capturingWebhookSender{events: make(chan *models.NetworkAuditEvent, 2)}
	first := NewNetworkAuditModule(path, firstSender)
	if response, err := first.SetWebhook(&models.WebhookConfig{Enabled: true, URL: "https://example.com/hook", Events: []string{"policy_changed"}}); err != nil || response.Result.Error != nil {
		t.Fatalf("enable webhook: response=%#v err=%v", response, err)
	}
	first.Record("mac:aa:bb:cc:dd:ee:01", "device_policy", "policy_changed", "success", "private detail")
	firstEvent := receiveWebhookEvent(t, firstSender)
	if firstEvent.DeviceID == "" || strings.Contains(firstEvent.DeviceID, "mac:") || strings.Contains(strings.ToLower(firstEvent.DeviceID), "aa:bb:cc") || firstEvent.Reason != "" {
		t.Fatalf("webhook leaked local data: %#v", firstEvent)
	}

	secondSender := &capturingWebhookSender{events: make(chan *models.NetworkAuditEvent, 2)}
	restarted := NewNetworkAuditModule(path, secondSender)
	restarted.Record("mac:aa:bb:cc:dd:ee:01", "device_policy", "policy_changed", "success", "another private detail")
	restarted.Record("mac:aa:bb:cc:dd:ee:02", "device_policy", "policy_changed", "success", "")
	secondEvent := receiveWebhookEvent(t, secondSender)
	thirdEvent := receiveWebhookEvent(t, secondSender)
	if secondEvent.DeviceID != firstEvent.DeviceID {
		t.Fatalf("pseudonym changed across restart: %q != %q", secondEvent.DeviceID, firstEvent.DeviceID)
	}
	if thirdEvent.DeviceID == secondEvent.DeviceID {
		t.Fatalf("different devices share pseudonym: %#v %#v", secondEvent, thirdEvent)
	}

	local, err := restarted.List("mac:aa:bb:cc:dd:ee:01", 10)
	if err != nil || len(local.Result.Events) == 0 || local.Result.Events[0].DeviceID != "mac:aa:bb:cc:dd:ee:01" {
		t.Fatalf("local audit association was lost: response=%#v err=%v", local, err)
	}
}

func TestWebhookRetryKeepsTheSamePrivatePayload(t *testing.T) {
	sender := &retryWebhookSender{events: make(chan *models.NetworkAuditEvent, 2)}
	module := NewNetworkAuditModule(filepath.Join(t.TempDir(), "audit.json"), sender)
	module.retryDelay = func(int) time.Duration { return 0 }
	if response, err := module.SetWebhook(&models.WebhookConfig{Enabled: true, URL: "https://example.com/hook", Events: []string{"policy_changed"}}); err != nil || response.Result.Error != nil {
		t.Fatalf("enable webhook: response=%#v err=%v", response, err)
	}
	module.Record("mac:aa:bb:cc:dd:ee:01", "device_policy", "policy_changed", "success", "private detail")
	first := receiveWebhookEvent(t, &capturingWebhookSender{events: sender.events})
	second := receiveWebhookEvent(t, &capturingWebhookSender{events: sender.events})
	if first.DeviceID != second.DeviceID || first.Reason != "" || second.Reason != "" || sender.attempts != 2 {
		t.Fatalf("unsafe or inconsistent retry: first=%#v second=%#v attempts=%d", first, second, sender.attempts)
	}
}

func TestWebhookQueueSaturationDoesNotBlockAudit(t *testing.T) {
	sender := &blockingWebhookSender{entered: make(chan struct{}, 1), release: make(chan struct{})}
	module := NewNetworkAuditModule(filepath.Join(t.TempDir(), "audit.json"), sender)
	module.persist = func(string, []byte) error { return nil }
	if response, err := module.SetWebhook(&models.WebhookConfig{Enabled: true, URL: "https://example.com/hook", Events: []string{"policy_changed"}}); err != nil || response.Result.Error != nil {
		t.Fatalf("enable webhook: response=%#v err=%v", response, err)
	}
	module.Record("mac:first", "device_policy", "policy_changed", "success", "")
	select {
	case <-sender.entered:
	case <-time.After(time.Second):
		t.Fatal("webhook worker did not start")
	}
	done := make(chan struct{})
	go func() {
		for index := 0; index < webhookQueueLimit+100; index++ {
			module.Record(fmt.Sprintf("mac:%d", index), "device_policy", "policy_changed", "success", "")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("saturated webhook queue blocked the audit path")
	}
	if len(module.queue) > cap(module.queue) || cap(module.queue) != webhookQueueLimit {
		t.Fatalf("unbounded webhook queue: len=%d cap=%d", len(module.queue), cap(module.queue))
	}
	close(sender.release)
}

func TestDefaultWebhookClientDoesNotFollowRedirects(t *testing.T) {
	module := NewNetworkAuditModule(filepath.Join(t.TempDir(), "audit.json"), nil)
	sender, ok := module.sender.(*httpWebhookSender)
	if !ok || sender.client.CheckRedirect == nil {
		t.Fatal("default webhook sender must define a redirect policy")
	}
	request, _ := http.NewRequest(http.MethodGet, "https://example.com/next", nil)
	if err := sender.client.CheckRedirect(request, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("redirect policy returned %v", err)
	}
}

func TestAuditReasonTruncatesUnicodeWithoutCorruption(t *testing.T) {
	reason := strings.Repeat("网", 161)
	got := sanitizeAuditReason(reason)
	if len([]rune(got)) != 160 || strings.ContainsRune(got, '\uFFFD') {
		t.Fatalf("invalid truncated reason: runes=%d", len([]rune(got)))
	}
}

func TestPolicyBundleDryRunApplyRestartAndRollback(t *testing.T) {
	directory := t.TempDir()
	groupStore := newJSONDeviceGroupStore(filepath.Join(directory, "groups.json"))
	groups := NewDeviceGroupModule(groupStore, nil)
	state, _ := groupStore.Read(context.Background())
	_, err := groupStore.Mutate(context.Background(), &models.DeviceGroupMutationRequest{Action: "upsert_group", Group: &models.DeviceGroup{ID: "old", Name: "Old", Members: []string{}, Policy: &models.GroupPolicy{}}}, state.Version)
	if err != nil {
		t.Fatal(err)
	}
	trafficStore := newTrafficInsightsFileStore(filepath.Join(directory, "traffic.json"))
	traffic := NewTrafficInsightsModule(trafficStore, nil)
	audit := NewNetworkAuditModule(filepath.Join(directory, "audit.json"), nil)
	audit.persist = func(string, []byte) error { return nil }
	module := NewPolicyBundleModule(groups, traffic, audit)
	bundle := &models.PolicyBundle{SchemaVersion: 1, Scope: "device_groups_and_quotas", Groups: []*models.DeviceGroup{{ID: "new", Name: "New", Members: []string{}, Policy: &models.GroupPolicy{}}}, DevicePolicies: map[string]*models.GroupPolicy{}, Quotas: map[string]*models.TrafficQuota{"mac:a": {DeviceID: "mac:a", Enabled: true, Period: "monthly", LimitBytes: 1000, Action: "notify"}}}
	plan, err := module.ImportPlan(context.Background(), &models.PolicyImportRequest{Bundle: bundle})
	if err != nil || plan.Result.Error != nil || !plan.Result.Plan.CanApply {
		t.Fatalf("plan=%#v err=%v", plan, err)
	}
	bad, _ := module.ImportPlan(context.Background(), &models.PolicyImportRequest{Bundle: &models.PolicyBundle{SchemaVersion: 99, Scope: bundle.Scope}})
	if bad.Result.Error == nil {
		t.Fatal("bad version accepted")
	}
	withoutPlan, _ := module.ImportApply(context.Background(), &models.PolicyImportRequest{Bundle: bundle, Confirmed: true})
	if withoutPlan.Result.Error == nil || withoutPlan.Result.Error.Code != "conflict" {
		t.Fatal("apply without matching dry-run accepted")
	}
	applied, err := module.ImportApply(context.Background(), &models.PolicyImportRequest{Bundle: bundle, Confirmed: true, PlanChecksum: plan.Result.Plan.Checksum, ExpectedGroupVersion: plan.Result.Plan.GroupVersion})
	if err != nil || applied.Result.Error != nil || !applied.Result.Changed {
		t.Fatalf("applied=%#v err=%v", applied, err)
	}
	restartedGroups, _ := newJSONDeviceGroupStore(groupStore.path).Read(context.Background())
	restartedTraffic, _, _ := newTrafficInsightsFileStore(trafficStore.path).Load()
	if len(restartedGroups.Groups) != 1 || restartedGroups.Groups[0].ID != "new" || restartedTraffic.Quotas["mac:a"] == nil {
		t.Fatalf("restart lost import")
	}
	repeatPlan, _ := module.ImportPlan(context.Background(), &models.PolicyImportRequest{Bundle: bundle})
	repeated, repeatErr := module.ImportApply(context.Background(), &models.PolicyImportRequest{Bundle: bundle, Confirmed: true, PlanChecksum: repeatPlan.Result.Plan.Checksum, ExpectedGroupVersion: repeatPlan.Result.Plan.GroupVersion})
	if repeatErr != nil || repeated.Result.Error != nil || repeated.Result.Changed {
		t.Fatalf("identical import was not idempotent: response=%#v err=%v", repeated, repeatErr)
	}

	traffic.store.persist = func(string, []byte) error { return errors.New("injected failure") }
	rollbackBundle := *bundle
	rollbackBundle.Groups = []*models.DeviceGroup{{ID: "broken", Name: "Broken", Members: []string{}, Policy: &models.GroupPolicy{}}}
	rollbackPlan, _ := module.ImportPlan(context.Background(), &models.PolicyImportRequest{Bundle: &rollbackBundle})
	failed, _ := module.ImportApply(context.Background(), &models.PolicyImportRequest{Bundle: &rollbackBundle, Confirmed: true, PlanChecksum: rollbackPlan.Result.Plan.Checksum, ExpectedGroupVersion: rollbackPlan.Result.Plan.GroupVersion})
	if failed.Result.Error == nil {
		t.Fatal("injected failure accepted")
	}
	after, _ := groupStore.Read(context.Background())
	if len(after.Groups) != 1 || after.Groups[0].ID != "new" {
		t.Fatalf("group rollback failed: %#v", after.Groups)
	}
}

func TestPolicyImportHTTPBodyIsBoundedAndSingleValue(t *testing.T) {
	t.Parallel()

	backend := &ServiceBackend{}
	tooLarge := httptest.NewRequest(http.MethodPost, "/policy-import", strings.NewReader(`{"bundle":{"schemaVersion":1,"scope":"`+strings.Repeat("x", (4<<20)+1)+`"}}`))
	response, err := backend.PostPolicyImportPlanV2(context.Background(), tooLarge)
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "validation_failed" {
		t.Fatalf("oversized response=%#v err=%v", response, err)
	}
	multiple := httptest.NewRequest(http.MethodPost, "/policy-import", strings.NewReader(`{"bundle":null}{"bundle":null}`))
	response, err = backend.PostPolicyImportPlanV2(context.Background(), multiple)
	if err != nil || response.Result.Error == nil || response.Result.Error.Code != "validation_failed" {
		t.Fatalf("multiple response=%#v err=%v", response, err)
	}
}

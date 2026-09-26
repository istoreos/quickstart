package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/istoreos/quickstart/backend/models"
)

const (
	defaultNetworkAuditPath = "/etc/quickstart/network-audit.json"
	networkAuditLimit       = 512
	networkAuditDeviceLimit = 2048
	webhookQueueLimit       = 64
	webhookAttemptLimit     = 5
)

type networkAuditDocument struct {
	SchemaVersion  int                         `json:"schemaVersion"`
	Sequence       uint64                      `json:"sequence"`
	Events         []*models.NetworkAuditEvent `json:"events"`
	Webhook        *models.WebhookConfig       `json:"webhook"`
	SeenDevices    map[string]bool             `json:"seenDevices"`
	DeviceOnline   map[string]bool             `json:"deviceOnline"`
	DeviceOrder    []string                    `json:"deviceOrder"`
	FloatingHolder string                      `json:"floatingGatewayHolder,omitempty"`
	WebhookSecret  string                      `json:"webhookSecret"`
}

type webhookDelivery struct {
	event   *models.NetworkAuditEvent
	attempt int
}

type webhookSender interface {
	Send(context.Context, string, *models.NetworkAuditEvent) error
}

type httpWebhookSender struct{ client *http.Client }

func (sender *httpWebhookSender) Send(ctx context.Context, target string, event *models.NetworkAuditEvent) error {
	raw, _ := json.Marshal(event)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := sender.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return errors.New("webhook returned non-success status")
	}
	return nil
}

type NetworkAuditModule struct {
	mu         sync.Mutex
	path       string
	document   networkAuditDocument
	loaded     bool
	sender     webhookSender
	queue      chan webhookDelivery
	start      sync.Once
	now        func() time.Time
	persist    func(string, []byte) error
	retryDelay func(int) time.Duration
}

func NewNetworkAuditModule(path string, sender webhookSender) *NetworkAuditModule {
	if sender == nil {
		sender = &httpWebhookSender{client: &http.Client{
			Timeout: 5 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}}
	}
	return &NetworkAuditModule{
		path: path, sender: sender, queue: make(chan webhookDelivery, webhookQueueLimit), now: time.Now, persist: persistClassificationOverrides,
		retryDelay: func(attempt int) time.Duration { return time.Second << minInt(attempt-1, 5) },
	}
}

func NewDefaultNetworkAuditModule() *NetworkAuditModule {
	return NewNetworkAuditModule(defaultNetworkAuditPath, nil)
}

func (module *NetworkAuditModule) Record(deviceID, resource, action, state, reason string) {
	if module == nil {
		return
	}
	module.mu.Lock()
	if err := module.loadLocked(); err != nil {
		module.mu.Unlock()
		return
	}
	event := module.appendEventLocked(deviceID, resource, action, state, reason)
	_ = module.persistLocked()
	webhook := cloneWebhookConfig(module.document.Webhook)
	module.mu.Unlock()
	if webhook != nil && webhook.Enabled && webhookEventEnabled(webhook, action) {
		module.startWorker()
		select {
		case module.queue <- webhookDelivery{event: event}:
		default:
		}
	}
}

func (module *NetworkAuditModule) ObserveInventory(response *models.DeviceInventoryResponse) {
	if module == nil || response == nil || response.Result == nil {
		return
	}
	module.mu.Lock()
	if err := module.loadLocked(); err != nil {
		module.mu.Unlock()
		return
	}
	deliveries := make([]*models.NetworkAuditEvent, 0)
	for _, device := range response.Result.Devices {
		if device == nil || device.DeviceID == "" {
			continue
		}
		previous, hasPrevious := module.document.DeviceOnline[device.DeviceID]
		if !module.document.SeenDevices[device.DeviceID] {
			module.rememberDeviceLocked(device.DeviceID)
			state := "offline"
			if device.Online {
				state = "online"
			}
			deliveries = append(deliveries, module.appendEventLocked(device.DeviceID, "device", "new_device", state, ""))
		} else if hasPrevious && previous != device.Online {
			action, state := "device_offline", "offline"
			if device.Online {
				action, state = "device_online", "online"
			}
			deliveries = append(deliveries, module.appendEventLocked(device.DeviceID, "device", action, state, ""))
		}
		module.document.DeviceOnline[device.DeviceID] = device.Online
	}
	if len(deliveries) > 0 {
		_ = module.persistLocked()
	}
	webhook := cloneWebhookConfig(module.document.Webhook)
	module.mu.Unlock()
	if webhook != nil && webhook.Enabled {
		for _, event := range deliveries {
			if webhookEventEnabled(webhook, event.Action) {
				module.startWorker()
				select {
				case module.queue <- webhookDelivery{event: event}:
				default:
				}
			}
		}
	}
}

func (module *NetworkAuditModule) ObserveFloatingGateway(response *models.FloatingGatewayResponse) {
	if module == nil || response == nil || response.Result == nil || response.Result.Config == nil || response.Result.Status == nil || !response.Result.Config.Enabled || response.Result.Status.Capability != "available" {
		return
	}
	holder := response.Result.Status.Holder
	if holder == "" {
		return
	}
	module.mu.Lock()
	if err := module.loadLocked(); err != nil {
		module.mu.Unlock()
		return
	}
	previous := module.document.FloatingHolder
	module.document.FloatingHolder = holder
	var event *models.NetworkAuditEvent
	if previous != "" && previous != holder {
		event = module.appendEventLocked("", "floating_gateway", "gateway_failover", holder, "")
	}
	if previous != holder {
		_ = module.persistLocked()
	}
	webhook := cloneWebhookConfig(module.document.Webhook)
	module.mu.Unlock()
	if event != nil && webhook != nil && webhook.Enabled && webhookEventEnabled(webhook, event.Action) {
		module.startWorker()
		select {
		case module.queue <- webhookDelivery{event: event}:
		default:
		}
	}
}

func (module *NetworkAuditModule) List(deviceID string, limit int) (*models.AdvancedNetworkResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	if err := module.loadLocked(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > networkAuditLimit {
		limit = 50
	}
	result := make([]*models.NetworkAuditEvent, 0, limit)
	for index := len(module.document.Events) - 1; index >= 0 && len(result) < limit; index-- {
		event := module.document.Events[index]
		if deviceID == "" || event.DeviceID == deviceID {
			copyValue := *event
			result = append(result, &copyValue)
		}
	}
	return &models.AdvancedNetworkResponse{Result: &models.AdvancedNetworkResult{Events: result, EventLimit: networkAuditLimit, Webhook: publicWebhookConfig(module.document.Webhook)}}, nil
}

func (module *NetworkAuditModule) SetWebhook(config *models.WebhookConfig) (*models.AdvancedNetworkResponse, error) {
	if err := validateWebhookConfig(config); err != nil {
		return advancedNetworkFailure("validation_failed", err.Error()), nil
	}
	module.mu.Lock()
	defer module.mu.Unlock()
	if err := module.loadLocked(); err != nil {
		return nil, err
	}
	module.document.Webhook = cloneWebhookConfig(config)
	if err := module.persistLocked(); err != nil {
		return nil, err
	}
	return &models.AdvancedNetworkResponse{Result: &models.AdvancedNetworkResult{Events: []*models.NetworkAuditEvent{}, EventLimit: networkAuditLimit, Webhook: publicWebhookConfig(config)}}, nil
}

func (module *NetworkAuditModule) loadLocked() error {
	if module.loaded {
		return nil
	}
	module.document = networkAuditDocument{SchemaVersion: 1, Events: []*models.NetworkAuditEvent{}, Webhook: &models.WebhookConfig{Events: []string{}}, SeenDevices: map[string]bool{}, DeviceOnline: map[string]bool{}}
	raw, err := os.ReadFile(module.path)
	if errors.Is(err, os.ErrNotExist) {
		if err := module.ensureWebhookSecretLocked(); err != nil {
			return err
		}
		module.loaded = true
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, &module.document); err != nil || module.document.SchemaVersion != 1 {
		return errors.New("invalid network audit file")
	}
	if module.document.Events == nil {
		module.document.Events = []*models.NetworkAuditEvent{}
	}
	if len(module.document.Events) > networkAuditLimit {
		module.document.Events = module.document.Events[len(module.document.Events)-networkAuditLimit:]
	}
	if module.document.Webhook == nil {
		module.document.Webhook = &models.WebhookConfig{Events: []string{}}
	}
	if module.document.SeenDevices == nil {
		module.document.SeenDevices = map[string]bool{}
	}
	if module.document.DeviceOnline == nil {
		module.document.DeviceOnline = map[string]bool{}
	}
	if err := module.ensureWebhookSecretLocked(); err != nil {
		return err
	}
	module.normalizeDeviceStateLocked()
	module.loaded = true
	return nil
}

func (module *NetworkAuditModule) ensureWebhookSecretLocked() error {
	if module.document.WebhookSecret != "" {
		return nil
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return err
	}
	module.document.WebhookSecret = hex.EncodeToString(secret)
	return nil
}

func (module *NetworkAuditModule) persistLocked() error {
	raw, err := json.Marshal(module.document)
	if err != nil {
		return err
	}
	return module.persist(module.path, raw)
}

func (module *NetworkAuditModule) startWorker() {
	module.start.Do(func() { go module.webhookWorker() })
}

func (module *NetworkAuditModule) webhookWorker() {
	for delivery := range module.queue {
		module.mu.Lock()
		webhook := cloneWebhookConfig(module.document.Webhook)
		event := module.webhookEventLocked(delivery.event)
		module.mu.Unlock()
		if webhook == nil || !webhook.Enabled {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := module.sender.Send(ctx, webhook.URL, event)
		cancel()
		if err != nil && delivery.attempt+1 < webhookAttemptLimit {
			delivery.attempt++
			delay := module.retryDelay(delivery.attempt)
			timer := time.NewTimer(delay)
			<-timer.C
			select {
			case module.queue <- delivery:
			default:
			}
		}
	}
}

func (module *NetworkAuditModule) webhookEventLocked(event *models.NetworkAuditEvent) *models.NetworkAuditEvent {
	copyValue := *event
	copyValue.Reason = ""
	if event.DeviceID != "" {
		key, err := hex.DecodeString(module.document.WebhookSecret)
		if err != nil || len(key) == 0 {
			key = []byte(module.document.WebhookSecret)
		}
		digest := hmac.New(sha256.New, key)
		_, _ = digest.Write([]byte(event.DeviceID))
		copyValue.DeviceID = "device_" + hex.EncodeToString(digest.Sum(nil)[:10])
	}
	return &copyValue
}

func (module *NetworkAuditModule) rememberDeviceLocked(deviceID string) {
	module.document.SeenDevices[deviceID] = true
	module.document.DeviceOrder = append(module.document.DeviceOrder, deviceID)
	for len(module.document.DeviceOrder) > networkAuditDeviceLimit {
		oldest := module.document.DeviceOrder[0]
		module.document.DeviceOrder = module.document.DeviceOrder[1:]
		delete(module.document.SeenDevices, oldest)
		delete(module.document.DeviceOnline, oldest)
	}
}

func (module *NetworkAuditModule) normalizeDeviceStateLocked() {
	seenOrder := make([]string, 0, len(module.document.SeenDevices))
	known := map[string]bool{}
	for _, deviceID := range module.document.DeviceOrder {
		if deviceID != "" && module.document.SeenDevices[deviceID] && !known[deviceID] {
			known[deviceID] = true
			seenOrder = append(seenOrder, deviceID)
		}
	}
	missing := make([]string, 0)
	for deviceID := range module.document.SeenDevices {
		if !known[deviceID] {
			missing = append(missing, deviceID)
		}
	}
	sort.Strings(missing)
	seenOrder = append(seenOrder, missing...)
	if len(seenOrder) > networkAuditDeviceLimit {
		seenOrder = seenOrder[len(seenOrder)-networkAuditDeviceLimit:]
	}
	boundedSeen, boundedOnline := map[string]bool{}, map[string]bool{}
	for _, deviceID := range seenOrder {
		boundedSeen[deviceID] = true
		boundedOnline[deviceID] = module.document.DeviceOnline[deviceID]
	}
	module.document.DeviceOrder = seenOrder
	module.document.SeenDevices = boundedSeen
	module.document.DeviceOnline = boundedOnline
}

func (module *NetworkAuditModule) appendEventLocked(deviceID, resource, action, state, reason string) *models.NetworkAuditEvent {
	module.document.Sequence++
	if resource == "dns" {
		reason = ""
	}
	event := &models.NetworkAuditEvent{ID: fmtAuditID(module.document.Sequence), At: module.now().UTC().Format(time.RFC3339Nano), DeviceID: deviceID, Resource: resource, Action: action, State: state, Reason: sanitizeAuditReason(reason)}
	module.document.Events = append(module.document.Events, event)
	if len(module.document.Events) > networkAuditLimit {
		module.document.Events = module.document.Events[len(module.document.Events)-networkAuditLimit:]
	}
	return event
}

func validateWebhookConfig(config *models.WebhookConfig) error {
	if config == nil {
		return errors.New("webhook config is required")
	}
	if !config.Enabled {
		config.URL = ""
		config.Events = []string{}
		return nil
	}
	parsed, err := url.Parse(config.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || len(config.URL) > 2048 {
		return errors.New("webhook URL must be an HTTP(S) URL without embedded credentials")
	}
	allowed := map[string]bool{"new_device": true, "gateway_failover": true, "policy_changed": true, "quota_exceeded": true}
	seen := map[string]bool{}
	events := make([]string, 0, len(config.Events))
	for _, event := range config.Events {
		if !allowed[event] {
			return errors.New("unsupported webhook event")
		}
		if !seen[event] {
			seen[event] = true
			events = append(events, event)
		}
	}
	if len(events) == 0 {
		return errors.New("at least one webhook event is required")
	}
	config.Events = events
	return nil
}

func webhookEventEnabled(config *models.WebhookConfig, action string) bool {
	for _, value := range config.Events {
		if value == action {
			return true
		}
	}
	return false
}
func cloneWebhookConfig(config *models.WebhookConfig) *models.WebhookConfig {
	if config == nil {
		return &models.WebhookConfig{Events: []string{}}
	}
	copyValue := *config
	copyValue.Events = append([]string(nil), config.Events...)
	return &copyValue
}
func publicWebhookConfig(config *models.WebhookConfig) *models.WebhookConfig {
	copyValue := cloneWebhookConfig(config)
	if copyValue.Enabled && copyValue.URL != "" {
		copyValue.URL = "configured"
	}
	return copyValue
}
func sanitizeAuditReason(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > 160 {
		value = string(runes[:160])
	}
	return value
}
func fmtAuditID(sequence uint64) string { return "event_" + strconv.FormatUint(sequence, 10) }
func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

var networkInterfaceWriteUsecaseTestMu sync.Mutex

type fakeNetworkInterfaceConfigFacade struct {
	lastInput NetworkInterfaceWriteInput
	resp      *models.SDKNormalResponse
	err       error
	called    int
}

func (svc *fakeNetworkInterfaceConfigFacade) ApplyConfigSet(ctx context.Context, input NetworkInterfaceWriteInput) (*models.SDKNormalResponse, error) {
	svc.called++
	svc.lastInput = input
	return svc.resp, svc.err
}

func TestNetworkInterfaceSetConfigDelegatesTypedInputToService(t *testing.T) {
	networkInterfaceWriteUsecaseTestMu.Lock()
	defer networkInterfaceWriteUsecaseTestMu.Unlock()

	oldFactory := newNetworkInterfaceConfigService
	defer func() {
		newNetworkInterfaceConfigService = oldFactory
	}()

	success := models.ResponseSuccess(int64(0))
	fakeSvc := &fakeNetworkInterfaceConfigFacade{
		resp: &models.SDKNormalResponse{Success: &success},
	}
	newNetworkInterfaceConfigService = func() networkInterfaceConfigFacade {
		return fakeSvc
	}

	input := NetworkInterfaceWriteInput{
		Configs: []*models.NetworkInterfaceConfig{
			{Name: "lan", Proto: "dhcp", Devices: []string{"eth0"}, FirewallType: "lan"},
		},
	}

	resp, err := NetworkInterfaceSetConfig(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected typed service error: %v", err)
	}
	if fakeSvc.called != 1 {
		t.Fatalf("expected facade called once, got %d", fakeSvc.called)
	}
	if resp != fakeSvc.resp {
		t.Fatalf("expected typed service to return service response pointer")
	}
	if len(fakeSvc.lastInput.Configs) != 1 || fakeSvc.lastInput.Configs[0].Name != "lan" {
		t.Fatalf("unexpected forwarded input: %#v", fakeSvc.lastInput)
	}
}

func TestNetworkInterfacePostConfigCompatibilityDelegatesToService(t *testing.T) {
	networkInterfaceWriteUsecaseTestMu.Lock()
	defer networkInterfaceWriteUsecaseTestMu.Unlock()

	oldFactory := newNetworkInterfaceConfigService
	defer func() {
		newNetworkInterfaceConfigService = oldFactory
	}()

	success := models.ResponseSuccess(int64(0))
	fakeSvc := &fakeNetworkInterfaceConfigFacade{
		resp: &models.SDKNormalResponse{Success: &success},
	}
	newNetworkInterfaceConfigService = func() networkInterfaceConfigFacade {
		return fakeSvc
	}

	body := `{"configs":[{"name":"lan","proto":"dhcp","devices":["eth0"],"firewallType":"lan"},{"name":"wan","proto":"pppoe","devices":["eth1"],"firewallType":"wan"}]}`
	req := httptest.NewRequest("POST", "/network/interface", strings.NewReader(body))

	resp, err := NetworkInterfacePostConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected wrapper error: %v", err)
	}
	if fakeSvc.called != 1 {
		t.Fatalf("expected facade called once, got %d", fakeSvc.called)
	}
	if resp != fakeSvc.resp {
		t.Fatalf("expected wrapper to return service response pointer")
	}
	if len(fakeSvc.lastInput.Configs) != 2 || fakeSvc.lastInput.Configs[0].Name != "lan" || fakeSvc.lastInput.Configs[1].Name != "wan" {
		t.Fatalf("unexpected forwarded configs: %#v", fakeSvc.lastInput.Configs)
	}
}

func TestNetworkInterfacePostConfigCompatibilityPropagatesServiceError(t *testing.T) {
	networkInterfaceWriteUsecaseTestMu.Lock()
	defer networkInterfaceWriteUsecaseTestMu.Unlock()

	oldFactory := newNetworkInterfaceConfigService
	defer func() {
		newNetworkInterfaceConfigService = oldFactory
	}()

	fakeSvc := &fakeNetworkInterfaceConfigFacade{err: errors.New("config failed")}
	newNetworkInterfaceConfigService = func() networkInterfaceConfigFacade {
		return fakeSvc
	}

	bodyBytes, _ := json.Marshal(models.NetworkInterfaceSetConfigRequest{
		Configs: []*models.NetworkInterfaceConfig{{Name: "lan", Proto: "dhcp", Devices: []string{"eth0"}, FirewallType: "lan"}},
	})
	req := httptest.NewRequest("POST", "/network/interface", strings.NewReader(string(bodyBytes)))

	_, err := NetworkInterfacePostConfig(context.Background(), req)
	if !errors.Is(err, fakeSvc.err) {
		t.Fatalf("expected service error propagated, got %v", err)
	}
	if fakeSvc.called != 1 {
		t.Fatalf("expected facade called once, got %d", fakeSvc.called)
	}
}

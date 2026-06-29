package runtime

import (
	"testing"

	"github.com/linkease/quick-start/istore-backend/models"
)

func TestMarkIfaceChangeStartsWorkerOnlyOnceAndDrainResetsPendingFlag(t *testing.T) {
	t.Parallel()

	state := NewState()

	if !state.MarkIfaceChange() {
		t.Fatal("expected first iface change to start worker")
	}
	if state.MarkIfaceChange() {
		t.Fatal("did not expect second iface change to start worker")
	}

	pending := state.Drain()
	if !pending.IfaceChange || pending.DHCPChange || pending.DHCP != nil {
		t.Fatalf("unexpected first drain: %#v", pending)
	}
	pending = state.Drain()
	if pending.IfaceChange || pending.DHCPChange || pending.DHCP != nil {
		t.Fatalf("expected pending flags to reset, got %#v", pending)
	}

	state.Finish()
	if !state.MarkIfaceChange() {
		t.Fatal("expected iface change after finish to start worker")
	}
}

func TestRegisterDHCPWaiterAllowsOneWaiterAndClearAllowsNext(t *testing.T) {
	t.Parallel()

	state := NewState()
	first := make(chan struct{}, 1)
	second := make(chan struct{}, 1)

	if !state.RegisterDHCPWaiter(first) {
		t.Fatal("expected first waiter to register")
	}
	if state.RegisterDHCPWaiter(second) {
		t.Fatal("did not expect second waiter to register")
	}
	state.ClearDHCPWaiter()
	if !state.RegisterDHCPWaiter(second) {
		t.Fatal("expected waiter registration after clear")
	}
}

func TestMarkDHCPValidNotifiesWaiterAndStartsWorkerOnlyWhenIdle(t *testing.T) {
	t.Parallel()

	state := NewState()
	waiter := make(chan struct{}, 1)
	if !state.RegisterDHCPWaiter(waiter) {
		t.Fatal("expected waiter registration")
	}

	start, notify := state.MarkDHCPValid(models.DHNSDhcpValidRequest{Ip: "10.0.0.2"})
	if !start {
		t.Fatal("expected idle worker to start")
	}
	if notify != waiter {
		t.Fatalf("expected waiter notification channel")
	}
	if state.RegisterDHCPWaiter(make(chan struct{}, 1)) == false {
		t.Fatal("expected waiter slot to be cleared by DHCP valid")
	}
	state.ClearDHCPWaiter()

	pending := state.Drain()
	if pending.IfaceChange || !pending.DHCPChange || pending.DHCP == nil || pending.DHCP.Ip != "10.0.0.2" {
		t.Fatalf("unexpected pending DHCP: %#v", pending)
	}

	start, notify = state.MarkDHCPValid(models.DHNSDhcpValidRequest{Ip: "10.0.0.3"})
	if start {
		t.Fatal("did not expect running worker to start again")
	}
	if notify != nil {
		t.Fatal("did not expect notification without waiter")
	}
}

func TestResetDHCPStateClearsPendingDHCP(t *testing.T) {
	t.Parallel()

	state := NewState()
	state.MarkDHCPValid(models.DHNSDhcpValidRequest{Ip: "10.0.0.2"})
	state.ResetDHCP()

	pending := state.Drain()
	if pending.DHCPChange || pending.DHCP != nil {
		t.Fatalf("expected DHCP state reset, got %#v", pending)
	}
}

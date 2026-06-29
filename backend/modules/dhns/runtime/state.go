package runtime

import (
	"sync"

	"github.com/linkease/quick-start/istore-backend/models"
)

type Pending struct {
	IfaceChange bool
	DHCPChange  bool
	DHCP        *models.DHNSDhcpValidRequest
}

type State struct {
	mu          sync.Mutex
	running     bool
	ifaceChange bool
	dhcpChange  bool
	lastDHCP    *models.DHNSDhcpValidRequest
	dhcpWaiter  chan struct{}
}

func NewState() *State {
	return &State{}
}

func (state *State) MarkIfaceChange() bool {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.ifaceChange = true
	return state.ensureRunningLocked()
}

func (state *State) Drain() Pending {
	state.mu.Lock()
	defer state.mu.Unlock()

	pending := Pending{
		IfaceChange: state.ifaceChange,
		DHCPChange:  state.dhcpChange,
		DHCP:        state.lastDHCP,
	}
	state.ifaceChange = false
	state.dhcpChange = false
	return pending
}

func (state *State) Finish() {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.running = false
}

func (state *State) RegisterDHCPWaiter(waiter chan struct{}) bool {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.dhcpWaiter != nil {
		return false
	}
	state.dhcpWaiter = waiter
	return true
}

func (state *State) ClearDHCPWaiter() {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.dhcpWaiter = nil
}

func (state *State) MarkDHCPValid(info models.DHNSDhcpValidRequest) (bool, chan struct{}) {
	state.mu.Lock()
	defer state.mu.Unlock()

	notify := state.dhcpWaiter
	state.dhcpWaiter = nil
	state.dhcpChange = true
	state.lastDHCP = &info
	return state.ensureRunningLocked(), notify
}

func (state *State) ResetDHCP() {
	state.mu.Lock()
	defer state.mu.Unlock()

	state.lastDHCP = nil
	state.dhcpChange = false
}

func (state *State) ensureRunningLocked() bool {
	if state.running {
		return false
	}
	state.running = true
	return true
}

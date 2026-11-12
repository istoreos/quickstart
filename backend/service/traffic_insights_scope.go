package service

import (
	"context"
	"errors"
	"github.com/istoreos/quickstart/backend/models"
)

// A child owns one immutable storage path and one document. Business logic is
// shared; there is no process-wide current-network switch or extra worker.
func (module *TrafficInsightsModule) forContext(ctx context.Context) *TrafficInsightsModule {
	if module == nil || module.root != nil || selectedLAN(ctx) == defaultLAN {
		return module
	}
	network := selectedLAN(ctx)
	if !validRouterContextLAN.MatchString(network) {
		return module
	}
	module.mu.Lock()
	defer module.mu.Unlock()
	return module.childLocked(network)
}

func (module *TrafficInsightsModule) childLocked(network string) *TrafficInsightsModule {
	if module.children == nil {
		module.children = map[string]*TrafficInsightsModule{}
	}
	if child := module.children[network]; child != nil {
		return child
	}
	store := newTrafficInsightsFileStore(lanStatePath(module.store.path, network))
	store.persist = module.store.persist
	child := NewTrafficInsightsModule(store, module.policy)
	child.network, child.root, child.now = network, module, module.now
	module.children[network] = child
	return child
}

func (module *TrafficInsightsModule) AttachLANScopes(resolver *LANScopeModule) {
	module.mu.Lock()
	module.resolver = resolver
	// Saved quotas resume after restart without requiring a page visit. Removed
	// networks remain stored and cannot enforce until they resolve again.
	for network := range lanStateFiles(module.store.path) {
		if network != defaultLAN {
			module.childLocked(network)
		}
	}
	module.mu.Unlock()
	module.wakeCollector()
}

func (module *TrafficInsightsModule) collectAllQuotaSamples() {
	module.mu.Lock()
	children := make([]*TrafficInsightsModule, 0, len(module.children))
	for _, child := range module.children {
		children = append(children, child)
	}
	collector, resolver, audit := module.collector, module.resolver, module.audit
	module.mu.Unlock()
	module.collectQuotaSample()
	for _, child := range children {
		child.mu.Lock()
		child.collector, child.resolver = collector, resolver
		if audit != nil {
			child.audit = audit.forContext(withLANScope(context.Background(), &models.LANScope{NetworkID: child.network}))
		}
		child.mu.Unlock()
		child.collectQuotaSample()
	}
}

// An enabled quota needs a usable sampler, not only an installed firewall.
// Run before taking the document lock; Record shares the normal attribution
// path and preserves any existing history before the quota is changed.
func (module *TrafficInsightsModule) prepareQuotaSample(ctx context.Context) error {
	root := module
	if module.root != nil {
		root = module.root
	}
	root.mu.Lock()
	collector := root.collector
	root.mu.Unlock()
	if collector == nil {
		return nil
	} // Existing embedders may feed Record themselves.
	response, err := collector(ctx)
	if err != nil {
		return err
	}
	if response == nil || response.Result == nil || response.Result.Health == nil {
		return errors.New("traffic_sampling_unavailable")
	}
	state := response.Result.Health.State
	if state != "ready" && state != "warming_up" {
		return errors.New("traffic_sampling_unavailable")
	}
	return module.Record(ctx, response)
}

func (module *TrafficInsightsModule) AttachAudit(audit *NetworkAuditModule) {
	module.mu.Lock()
	module.audit = audit
	module.mu.Unlock()
}

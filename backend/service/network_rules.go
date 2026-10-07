package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"

	"github.com/istoreos/quickstart/backend/models"
)

const networkRulesBulkLimit = 128

type networkRulesSnapshot struct {
	Rules   []*models.NetworkRule
	Version string
}
type networkRulesStore interface {
	Read(context.Context) (networkRulesSnapshot, error)
	Apply(context.Context, []*models.NetworkRule) error
}

type NetworkRulesModule struct {
	mu    sync.Mutex
	store networkRulesStore
}

func NewNetworkRulesModule(store networkRulesStore) *NetworkRulesModule {
	return &NetworkRulesModule{store: store}
}
func NewDefaultNetworkRulesModule(inventory *DeviceInventoryModule, policy *DevicePolicyModule, gateway *GatewayPolicyModule) *NetworkRulesModule {
	return NewNetworkRulesModule(&systemNetworkRulesStore{inventory: inventory, policy: policy, gateway: gateway})
}

func (module *NetworkRulesModule) List(ctx context.Context) (*models.NetworkRulesResponse, error) {
	if module == nil || module.store == nil {
		return nil, errors.New("network rules module is unavailable")
	}
	snapshot, err := module.store.Read(ctx)
	if err != nil {
		return nil, err
	}
	return &models.NetworkRulesResponse{Result: &models.NetworkRulesResult{Rules: snapshot.Rules, Version: snapshot.Version}}, nil
}

func (module *NetworkRulesModule) Plan(ctx context.Context, request *models.NetworkRulesBulkRequest) (*models.NetworkRulesBulkResponse, error) {
	_, plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	return &models.NetworkRulesBulkResponse{Result: &models.NetworkRulesBulkResult{Plan: plan}}, nil
}

func (module *NetworkRulesModule) Apply(ctx context.Context, request *models.NetworkRulesBulkRequest) (*models.NetworkRulesBulkResponse, error) {
	module.mu.Lock()
	defer module.mu.Unlock()
	selected, plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	result := &models.NetworkRulesBulkResult{Plan: plan}
	if plan.Error != nil || !plan.CanApply {
		result.Error = plan.Error
		return &models.NetworkRulesBulkResponse{Result: result}, nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != plan.Version {
		result.Error = &models.DevicePolicyError{Code: "conflict", Message: "network rules changed; review the operation again"}
		return &models.NetworkRulesBulkResponse{Result: result}, nil
	}
	if err := module.store.Apply(ctx, selected); err != nil {
		result.Error = &models.DevicePolicyError{Code: "apply_failed", Message: err.Error()}
		return &models.NetworkRulesBulkResponse{Result: result}, nil
	}
	result.Changed = len(selected) > 0
	return &models.NetworkRulesBulkResponse{Result: result}, nil
}

func (module *NetworkRulesModule) plan(ctx context.Context, request *models.NetworkRulesBulkRequest) ([]*models.NetworkRule, *models.NetworkRulesBulkPlan, error) {
	snapshot, err := module.store.Read(ctx)
	if err != nil {
		return nil, nil, err
	}
	plan := &models.NetworkRulesBulkPlan{Action: "delete", RuleIDs: []string{}, Version: snapshot.Version, RollbackScope: []string{}}
	if request == nil || request.Action != "delete" {
		plan.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "only delete is supported"}
		return nil, plan, nil
	}
	ids := uniqueSortedStrings(request.RuleIDs)
	plan.RuleIDs = ids
	if len(ids) == 0 || len(ids) > networkRulesBulkLimit {
		plan.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "select between 1 and 128 rules"}
		return nil, plan, nil
	}
	byID := make(map[string]*models.NetworkRule, len(snapshot.Rules))
	for _, rule := range snapshot.Rules {
		byID[rule.ID] = rule
	}
	selected := make([]*models.NetworkRule, 0, len(ids))
	scopes := map[string]bool{}
	for _, id := range ids {
		rule := byID[id]
		if rule == nil {
			plan.Error = &models.DevicePolicyError{Code: "conflict", Message: "a selected rule no longer exists"}
			return nil, plan, nil
		}
		selected = append(selected, rule)
		if rule.Kind == "static" || rule.Kind == "route" {
			scopes["dhcp"] = true
		} else {
			scopes["eqos"] = true
			scopes["firewall"] = true
		}
	}
	for scope := range scopes {
		plan.RollbackScope = append(plan.RollbackScope, scope)
	}
	sort.Strings(plan.RollbackScope)
	plan.AffectedCount = len(selected)
	plan.CanApply = true
	return selected, plan, nil
}

func stableNetworkRuleID(kind string, values ...string) string {
	sum := sha256.Sum256([]byte(kind + "|" + strings.Join(values, "|")))
	return kind + ":" + hex.EncodeToString(sum[:6])
}
func versionNetworkRules(rules []*models.NetworkRule) string {
	raw, _ := json.Marshal(rules)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

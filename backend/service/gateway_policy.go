package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strings"

	"github.com/istoreos/quickstart/backend/models"
)

type gatewayPolicyHost struct {
	SectionName string
	MAC         string
	TagName     string
}

type gatewayPolicySnapshot struct {
	LAN                 LanStatusSnapshot
	DHCP                *LanDhcpState
	Hosts               []gatewayPolicyHost
	Prefixes            []netip.Prefix
	UnreachableGateways map[string]bool
	Version             string
}

type gatewayTargetDescriptor struct {
	Public      *models.GatewayTarget
	TagName     string
	TagTitle    string
	Options     []string
	Materialize bool
	Stored      bool
}

type gatewayPolicyExecutionPlan struct {
	Public *models.GatewayAssignmentPlan
	State  gatewayPolicySnapshot
	MAC    string
	Target *gatewayTargetDescriptor
}

type gatewayTargetMutationExecutionPlan struct {
	Public       *models.GatewayTargetMutationPlan
	State        gatewayPolicySnapshot
	Target       *gatewayTargetDescriptor
	Replacement  *gatewayTargetDescriptor
	GroupVersion string
}

type gatewayPolicyStore interface {
	ReadState(context.Context) (gatewayPolicySnapshot, error)
	Apply(context.Context, gatewayPolicyExecutionPlan) error
	ApplyTargetMutation(context.Context, gatewayTargetMutationExecutionPlan) error
}

type GatewayPolicyModule struct {
	inventory    *DeviceInventoryModule
	store        gatewayPolicyStore
	groups       *jsonDeviceGroupStore
	transactions *TaskTransactionJournal
}

func NewGatewayPolicyModule(inventory *DeviceInventoryModule, store gatewayPolicyStore) *GatewayPolicyModule {
	return &GatewayPolicyModule{inventory: inventory, store: store, transactions: newMemoryTaskTransactionJournal()}
}

func NewDefaultGatewayPolicyModule(inventory *DeviceInventoryModule) *GatewayPolicyModule {
	groups := newJSONDeviceGroupStore(defaultDeviceGroupStorePath)
	module := NewGatewayPolicyModule(inventory, newDefaultGatewayPolicyStore(groups))
	module.groups = groups
	module.transactions = NewDefaultTaskTransactionJournal()
	return module
}

func (module *GatewayPolicyModule) ListTargets(ctx context.Context) (*models.GatewayTargetListResponse, error) {
	state, err := module.store.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	index := buildGatewayTargetIndex(state)
	countGatewayReferences(index, state.Hosts)
	module.countManagedGatewayReferences(ctx, index, state)
	targets := make([]*models.GatewayTarget, 0, len(index.targets))
	for _, target := range index.targets {
		targets = append(targets, target.Public)
	}
	sort.Slice(targets, func(i, j int) bool {
		left, right := gatewayTargetRank(targets[i].Kind), gatewayTargetRank(targets[j].Kind)
		if left != right {
			return left < right
		}
		return targets[i].ID < targets[j].ID
	})
	return &models.GatewayTargetListResponse{Result: &models.GatewayTargetListResult{
		Targets: targets, Version: state.Version, Health: "ready",
	}}, nil
}

func (module *GatewayPolicyModule) PlanAssignment(ctx context.Context, request *models.GatewayAssignmentRequest) (*models.GatewayAssignmentPlanResponse, error) {
	plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	return &models.GatewayAssignmentPlanResponse{Result: plan.Public}, nil
}

func (module *GatewayPolicyModule) ApplyAssignment(ctx context.Context, request *models.GatewayAssignmentRequest) (*models.GatewayAssignmentApplyResponse, error) {
	plan, err := module.plan(ctx, request)
	if err != nil {
		return nil, err
	}
	result := &models.GatewayAssignmentApplyResult{Plan: plan.Public, EffectState: "unverifiable"}
	if plan.Public.Error != nil || !plan.Public.CanApply {
		result.Error = plan.Public.Error
		return &models.GatewayAssignmentApplyResponse{Result: result}, nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != plan.State.Version {
		result.Error = &models.DevicePolicyError{Code: "conflict", Message: "gateway configuration changed; plan again"}
		return &models.GatewayAssignmentApplyResponse{Result: result}, nil
	}
	if len(plan.Public.Changes) == 0 {
		result.EffectState = "active"
		return &models.GatewayAssignmentApplyResponse{Result: result}, nil
	}
	if err := module.store.Apply(ctx, plan); err != nil {
		result.Error = &models.DevicePolicyError{Code: "apply_failed", Message: err.Error()}
		return &models.GatewayAssignmentApplyResponse{Result: result}, nil
	}
	result.Changed = true
	if plan.Public.RequiresRenewal {
		result.EffectState = "pending_renewal"
	} else {
		result.EffectState = "active"
	}
	return &models.GatewayAssignmentApplyResponse{Result: result}, nil
}

func (module *GatewayPolicyModule) References(ctx context.Context, targetID string) (*models.GatewayReferencesResponse, error) {
	state, err := module.store.ReadState(ctx)
	if err != nil {
		return nil, err
	}
	index := buildGatewayTargetIndex(state)
	if index.targets[targetID] == nil {
		return &models.GatewayReferencesResponse{Result: &models.GatewayReferencesResult{TargetID: targetID, References: []*models.GatewayReference{}}}, nil
	}
	deviceByMAC := map[string]*models.DeviceInventoryItem{}
	if module.inventory != nil {
		if snapshot, snapshotErr := module.inventory.Snapshot(ctx); snapshotErr == nil && snapshot.Result != nil {
			for _, device := range snapshot.Result.Devices {
				if device != nil && device.Mac != "" {
					deviceByMAC[normalizeInventoryMAC(device.Mac)] = device
				}
			}
		}
	}
	references := make([]*models.GatewayReference, 0)
	for _, host := range state.Hosts {
		if index.targetIDForTag(host.TagName) != targetID {
			continue
		}
		reference := &models.GatewayReference{Scope: "orphaned"}
		if device := deviceByMAC[normalizeInventoryMAC(host.MAC)]; device != nil {
			reference.DeviceID = device.DeviceID
			reference.Scope = deviceProfileScope(device)
		}
		references = append(references, reference)
	}
	if module.groups != nil {
		groupReferences, _, _, groupErr := module.groups.GatewayReferences(targetID)
		if groupErr != nil {
			return nil, groupErr
		}
		references = append(references, groupReferences...)
	}
	if target := index.targets[targetID]; gatewayTargetIsLanDefault(target, state) {
		references = append(references, &models.GatewayReference{Scope: "lan_default"})
	}
	sort.Slice(references, func(i, j int) bool { return references[i].DeviceID < references[j].DeviceID })
	return &models.GatewayReferencesResponse{Result: &models.GatewayReferencesResult{TargetID: targetID, References: references}}, nil
}

func (module *GatewayPolicyModule) countManagedGatewayReferences(ctx context.Context, index gatewayTargetIndex, state gatewayPolicySnapshot) {
	for _, target := range index.targets {
		if gatewayTargetIsLanDefault(target, state) {
			target.Public.ReferenceCount++
		}
	}
	if module.groups == nil {
		return
	}
	groupState, err := module.groups.Read(ctx)
	if err != nil {
		return
	}
	increment := func(policy *models.GroupPolicy) {
		if policy != nil && policy.TargetID != "" {
			if target := index.targets[policy.TargetID]; target != nil {
				target.Public.ReferenceCount++
			}
		}
	}
	increment(groupState.GlobalPolicy)
	for _, group := range groupState.Groups {
		if group != nil {
			increment(group.Policy)
		}
	}
	for _, policy := range groupState.DevicePolicies {
		increment(policy)
	}
}

func (module *GatewayPolicyModule) PlanTargetMutation(ctx context.Context, request *models.GatewayTargetMutationRequest) (*models.GatewayTargetMutationPlanResponse, error) {
	plan, err := module.planTargetMutation(ctx, request)
	if err != nil {
		return nil, err
	}
	return &models.GatewayTargetMutationPlanResponse{Result: plan.Public}, nil
}

func (module *GatewayPolicyModule) ApplyTargetMutation(ctx context.Context, request *models.GatewayTargetMutationRequest) (*models.GatewayTargetMutationApplyResponse, error) {
	if request == nil {
		return &models.GatewayTargetMutationApplyResponse{Result: &models.GatewayTargetMutationApplyResult{
			Plan:  &models.GatewayTargetMutationPlan{ReferenceSummary: &models.GatewayReferenceSummary{}},
			Error: &models.DevicePolicyError{Code: "validation_failed", Message: "request is required"},
		}}, nil
	}
	fingerprint, _ := json.Marshal(request)
	begin, beginErr := module.transactions.Begin(ctx, "gateway_target", request.IdempotencyKey, string(fingerprint))
	if beginErr != nil {
		return &models.GatewayTargetMutationApplyResponse{Result: &models.GatewayTargetMutationApplyResult{
			Plan:  &models.GatewayTargetMutationPlan{Action: request.Action, ReferenceSummary: &models.GatewayReferenceSummary{}},
			Error: &models.DevicePolicyError{Code: "transaction_unavailable", Message: "could not create transaction record"},
		}}, nil
	}
	if begin.Conflict {
		return &models.GatewayTargetMutationApplyResponse{Result: &models.GatewayTargetMutationApplyResult{
			Plan: &models.GatewayTargetMutationPlan{Action: request.Action, ReferenceSummary: &models.GatewayReferenceSummary{}}, Transaction: begin.Transaction,
			Error: &models.DevicePolicyError{Code: "conflict", Message: "idempotency key was already used for a different change"},
		}}, nil
	}
	if begin.Replay {
		return &models.GatewayTargetMutationApplyResponse{Result: &models.GatewayTargetMutationApplyResult{
			Plan: replayedGatewayTargetPlan(request), Transaction: begin.Transaction,
		}}, nil
	}
	plan, err := module.planTargetMutation(ctx, request)
	if err != nil {
		return nil, err
	}
	result := &models.GatewayTargetMutationApplyResult{Plan: plan.Public, Transaction: begin.Transaction}
	if plan.Public.Error != nil || !plan.Public.CanApply {
		result.Error = plan.Public.Error
		_ = module.transactions.Advance(ctx, begin.Transaction, "validate", "rejected", "")
		return &models.GatewayTargetMutationApplyResponse{Result: result}, nil
	}
	if request.ExpectedVersion != "" && request.ExpectedVersion != plan.State.Version {
		result.Error = &models.DevicePolicyError{Code: "conflict", Message: "gateway targets changed; plan again"}
		_ = module.transactions.Advance(ctx, begin.Transaction, "validate", "rejected", "reload_and_plan")
		return &models.GatewayTargetMutationApplyResponse{Result: result}, nil
	}
	if err := module.store.ApplyTargetMutation(ctx, plan); err != nil {
		result.Error = &models.DevicePolicyError{Code: "apply_failed", Message: err.Error()}
		_ = module.transactions.Advance(ctx, begin.Transaction, "rollback", "rolled_back", "retry")
		return &models.GatewayTargetMutationApplyResponse{Result: result}, nil
	}
	result.Changed = true
	_ = module.transactions.Advance(ctx, begin.Transaction, "verify", "committed", "")
	return &models.GatewayTargetMutationApplyResponse{Result: result}, nil
}

func replayedGatewayTargetPlan(request *models.GatewayTargetMutationRequest) *models.GatewayTargetMutationPlan {
	targetID := request.TargetID
	if request.Action == "create" && targetID == "" {
		targetID = gatewayOpaqueID(strings.TrimSpace(request.Kind), strings.TrimSpace(request.Name)+"|"+strings.TrimSpace(request.Gateway))
	}
	return &models.GatewayTargetMutationPlan{
		Action: request.Action, Target: &models.GatewayTarget{ID: targetID, Name: strings.TrimSpace(request.Name), Kind: strings.TrimSpace(request.Kind), Gateway: strings.TrimSpace(request.Gateway)},
		ReplacementTargetID: request.ReplacementTargetID, ReferenceSummary: &models.GatewayReferenceSummary{}, AffectedDevices: []string{}, CanApply: true,
	}
}

func (module *GatewayPolicyModule) planTargetMutation(ctx context.Context, request *models.GatewayTargetMutationRequest) (gatewayTargetMutationExecutionPlan, error) {
	if module == nil || module.store == nil {
		return gatewayTargetMutationExecutionPlan{}, errors.New("gateway policy module is unavailable")
	}
	state, err := module.store.ReadState(ctx)
	if err != nil {
		return gatewayTargetMutationExecutionPlan{}, err
	}
	index := buildGatewayTargetIndex(state)
	countGatewayReferences(index, state.Hosts)
	public := &models.GatewayTargetMutationPlan{
		AffectedDevices: []string{}, ReferenceSummary: &models.GatewayReferenceSummary{},
		Version: state.Version, RollbackPoint: state.Version,
	}
	plan := gatewayTargetMutationExecutionPlan{Public: public, State: state}
	if request == nil {
		public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "request is required"}
		return plan, nil
	}
	public.Action = request.Action
	switch request.Action {
	case "create", "update":
		name, kind, gateway, validation := validateGatewayTargetInput(request.Name, request.Kind, request.Gateway, state.Prefixes)
		if validation != nil {
			public.Error = validation
			return plan, nil
		}
		targetID := request.TargetID
		var existing *gatewayTargetDescriptor
		if request.Action == "create" {
			if targetID != "" {
				public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "targetId must be empty when creating a route"}
				return plan, nil
			}
			targetID = gatewayOpaqueID(kind, name+"|"+gateway)
			if index.targets[targetID] != nil {
				public.Error = &models.DevicePolicyError{Code: "conflict", Message: "an equivalent gateway target already exists"}
				return plan, nil
			}
		} else {
			existing = index.targets[targetID]
			if existing == nil {
				public.Error = &models.DevicePolicyError{Code: "not_found", Message: "gateway target was not found"}
				return plan, nil
			}
			if !existing.Stored || !gatewayTargetEditable(existing.Public.Kind) {
				public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "built-in gateway target cannot be edited"}
				return plan, nil
			}
		}
		tagName := quickstartGatewayTag(targetID)
		if existing != nil && existing.TagName != "" {
			tagName = existing.TagName
		}
		plan.Target = &gatewayTargetDescriptor{Public: &models.GatewayTarget{
			ID: targetID, Name: name, Kind: kind, Gateway: gateway, DNS: []string{gateway}, Supported: true,
			DesiredState: "available", EffectState: "unverified", Version: state.Version,
		}, TagName: tagName, TagTitle: name, Options: gatewayAndDNSOptions(gateway), Materialize: true, Stored: existing != nil}
		public.Target = plan.Target.Public
		public.CanApply = true
		return plan, nil
	case "delete":
		target := index.targets[request.TargetID]
		if target == nil {
			public.Error = &models.DevicePolicyError{Code: "not_found", Message: "gateway target was not found"}
			return plan, nil
		}
		if !target.Stored || !gatewayTargetEditable(target.Public.Kind) {
			public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "built-in gateway target cannot be deleted"}
			return plan, nil
		}
		plan.Target, public.Target = target, target.Public
		for _, host := range state.Hosts {
			if index.targetIDForTag(host.TagName) != request.TargetID {
				continue
			}
			public.ReferenceSummary.Devices++
			if deviceID := module.deviceIDForMAC(ctx, host.MAC); deviceID != "" {
				public.AffectedDevices = append(public.AffectedDevices, deviceID)
			}
		}
		if gatewayTargetIsLanDefault(target, state) {
			public.ReferenceSummary.LanDefault = 1
		}
		if module.groups != nil {
			groupReferences, groupDevices, groupVersion, groupErr := module.groups.GatewayReferences(request.TargetID)
			if groupErr != nil {
				return gatewayTargetMutationExecutionPlan{}, groupErr
			}
			plan.GroupVersion = groupVersion
			seenDevices := map[string]bool{}
			for _, deviceID := range public.AffectedDevices {
				seenDevices[deviceID] = true
			}
			for _, reference := range groupReferences {
				switch reference.Scope {
				case "group":
					public.ReferenceSummary.Groups++
				case "global_policy":
					public.ReferenceSummary.GlobalPolicy++
				case "device_policy":
					if !seenDevices[reference.DeviceID] {
						public.ReferenceSummary.Devices++
					}
				}
			}
			for _, deviceID := range groupDevices {
				if !seenDevices[deviceID] {
					public.AffectedDevices = append(public.AffectedDevices, deviceID)
					seenDevices[deviceID] = true
				}
			}
		}
		sort.Strings(public.AffectedDevices)
		public.ReplacementTargetID = request.ReplacementTargetID
		totalReferences := public.ReferenceSummary.Devices + public.ReferenceSummary.Groups + public.ReferenceSummary.GlobalPolicy + public.ReferenceSummary.LanDefault
		if totalReferences > 0 && request.ReplacementTargetID == "" {
			public.Error = &models.DevicePolicyError{Code: "replacement_required", Message: "select a replacement for devices using this route"}
			return plan, nil
		}
		if request.ReplacementTargetID != "" {
			replacement := index.targets[request.ReplacementTargetID]
			if replacement == nil || replacement == target || !replacement.Public.Supported {
				public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "replacement gateway target is unavailable"}
				return plan, nil
			}
			plan.Replacement = replacement
		}
		public.CanApply = true
		return plan, nil
	default:
		public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "action must be create, update or delete"}
		return plan, nil
	}
}

func gatewayTargetIsLanDefault(target *gatewayTargetDescriptor, state gatewayPolicySnapshot) bool {
	if target == nil || target.Public == nil || state.DHCP == nil {
		return false
	}
	return target.Public.Gateway != "" && detectDhcpGateway(state.LAN, state.DHCP.DhcpOptions) == target.Public.Gateway
}

func validateGatewayTargetInput(name, kind, gateway string, prefixes []netip.Prefix) (string, string, string, *models.DevicePolicyError) {
	name, kind, gateway = strings.TrimSpace(name), strings.TrimSpace(kind), strings.TrimSpace(gateway)
	if name == "" || len([]rune(name)) > 64 || strings.ContainsAny(name, "\r\n\x00") {
		return "", "", "", &models.DevicePolicyError{Code: "validation_failed", Message: "route name must contain 1 to 64 visible characters"}
	}
	if kind != "bypass" && kind != "custom" {
		return "", "", "", &models.DevicePolicyError{Code: "validation_failed", Message: "route kind must be bypass or custom"}
	}
	address, err := netip.ParseAddr(gateway)
	if err != nil || !address.Is4() || !gatewayInPrefixes(gateway, prefixes) {
		return "", "", "", &models.DevicePolicyError{Code: "validation_failed", Message: "gateway must be an IPv4 address in the selected LAN"}
	}
	return name, kind, address.String(), nil
}

func gatewayTargetEditable(kind string) bool { return kind == "bypass" || kind == "custom" }

func quickstartGatewayTag(targetID string) string {
	sum := sha256.Sum256([]byte(targetID))
	return "qs_route_" + hex.EncodeToString(sum[:6])
}

func (module *GatewayPolicyModule) deviceIDForMAC(ctx context.Context, mac string) string {
	if module.inventory == nil {
		return ""
	}
	device, err := findGatewayPolicyDeviceByMAC(ctx, module.inventory, mac)
	if err != nil || device == nil {
		return ""
	}
	return device.DeviceID
}

func findGatewayPolicyDeviceByMAC(ctx context.Context, inventory *DeviceInventoryModule, mac string) (*models.DeviceInventoryItem, error) {
	snapshot, err := inventory.Snapshot(ctx)
	if err != nil || snapshot == nil || snapshot.Result == nil {
		return nil, err
	}
	for _, device := range snapshot.Result.Devices {
		if device != nil && normalizeInventoryMAC(device.Mac) == normalizeInventoryMAC(mac) {
			return device, nil
		}
	}
	return nil, nil
}

func (module *GatewayPolicyModule) plan(ctx context.Context, request *models.GatewayAssignmentRequest) (gatewayPolicyExecutionPlan, error) {
	if module == nil || module.store == nil {
		return gatewayPolicyExecutionPlan{}, errors.New("gateway policy module is unavailable")
	}
	state, err := module.store.ReadState(ctx)
	if err != nil {
		return gatewayPolicyExecutionPlan{}, err
	}
	index := buildGatewayTargetIndex(state)
	countGatewayReferences(index, state.Hosts)
	public := &models.GatewayAssignmentPlan{
		Action: requestAction(request), TargetID: requestTargetID(request), Version: state.Version,
		RollbackPoint: state.Version, Changes: []*models.GatewayPlanChange{}, AffectedDevices: []string{},
	}
	result := gatewayPolicyExecutionPlan{Public: public, State: state}
	if request == nil || request.TargetID == "" {
		public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "targetId is required"}
		return result, nil
	}
	target := index.targets[request.TargetID]
	if target == nil {
		public.Error = &models.DevicePolicyError{Code: "not_found", Message: "gateway target was not found"}
		return result, nil
	}
	result.Target = target
	switch request.Action {
	case "delete_target":
		if !target.Stored || target.Public.Kind == "default" || target.Public.Kind == "self" || target.Public.Kind == "upstream" {
			public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "built-in gateway target cannot be deleted"}
			return result, nil
		}
		if target.Public.ReferenceCount > 0 {
			public.Error = &models.DevicePolicyError{Code: "conflict", Message: "gateway target is still referenced by devices"}
			return result, nil
		}
		public.Changes = append(public.Changes, &models.GatewayPlanChange{Kind: "delete_target", Description: "remove unused gateway target"})
		public.CanApply = true
		return result, nil
	case "assign":
	default:
		public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "action must be assign or delete_target"}
		return result, nil
	}
	if state.DHCP == nil || state.DHCP.DhcpIgnore {
		public.Error = &models.DevicePolicyError{Code: "dhcp_authority_unavailable", Message: "local DHCPv4 authority is required to assign an internet path"}
		return result, nil
	}
	public.DeviceID = request.DeviceID
	if request.DeviceID == "" || module.inventory == nil {
		public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "deviceId is required"}
		return result, nil
	}
	device, findErr := findGatewayPolicyDevice(ctx, module.inventory, request.DeviceID)
	if findErr != nil || device.Mac == "" {
		public.Error = &models.DevicePolicyError{Code: "not_found", Message: "a persistent MAC device is required"}
		return result, nil
	}
	if !target.Public.Supported {
		public.Error = &models.DevicePolicyError{Code: "validation_failed", Message: "gateway target contains unsupported DHCP options"}
		return result, nil
	}
	if target.Public.Gateway != "" && !gatewayInPrefixes(target.Public.Gateway, state.Prefixes) {
		public.Error = &models.DevicePolicyError{Code: "gateway_outside_lan", Message: "gateway target is outside the LAN subnet"}
		return result, nil
	}
	if target.Public.Gateway != "" && state.UnreachableGateways[target.Public.Gateway] {
		public.Error = &models.DevicePolicyError{Code: "gateway_unreachable", Message: "gateway did not answer neighbor discovery; confirm that it is powered on and connected to this LAN, then retry"}
		return result, nil
	}
	result.MAC = device.Mac
	currentTag := ""
	for _, host := range state.Hosts {
		if normalizeInventoryMAC(host.MAC) == normalizeInventoryMAC(device.Mac) {
			currentTag = host.TagName
			break
		}
	}
	public.CurrentTargetID = index.targetIDForTag(currentTag)
	public.CanApply = true
	if public.CurrentTargetID == request.TargetID || gatewayTargetsAreEquivalent(public.CurrentTargetID, request.TargetID, index, state) {
		return result, nil
	}
	public.Changes = append(public.Changes, &models.GatewayPlanChange{Kind: "assignment", Description: "change the device internet path"})
	public.AffectedDevices = append(public.AffectedDevices, request.DeviceID)
	public.RequiresRenewal = true
	return result, nil
}

func gatewayTargetsAreEquivalent(currentID, requestedID string, index gatewayTargetIndex, state gatewayPolicySnapshot) bool {
	currentGateway := gatewayTargetEffectiveGateway(currentID, index, state)
	requestedGateway := gatewayTargetEffectiveGateway(requestedID, index, state)
	return currentGateway != "" && currentGateway == requestedGateway
}

func gatewayTargetEffectiveGateway(targetID string, index gatewayTargetIndex, state gatewayPolicySnapshot) string {
	if targetID == "default" {
		plan := BuildAutoDhcpPlan(state.LAN, state.DHCP)
		if plan.DhcpGateway != "" {
			return plan.DhcpGateway
		}
		return state.LAN.LanAddr
	}
	if target := index.targets[targetID]; target != nil {
		return target.Public.Gateway
	}
	return ""
}

type gatewayTargetIndex struct {
	targets map[string]*gatewayTargetDescriptor
	tagToID map[string]string
}

func (index gatewayTargetIndex) targetIDForTag(tag string) string {
	if tag == "" {
		return "default"
	}
	if targetID := index.tagToID[tag]; targetID != "" {
		return targetID
	}
	return "unknown"
}

func buildGatewayTargetIndex(state gatewayPolicySnapshot) gatewayTargetIndex {
	index := gatewayTargetIndex{targets: map[string]*gatewayTargetDescriptor{}, tagToID: map[string]string{}}
	addBuiltIn := func(id, name, kind, gateway string) {
		tagName := ""
		options := []string{}
		if id != "default" && gateway != "" {
			tagName = ipToDhcpTag(gateway)
			options = []string{"3," + gateway, "6," + gateway}
		}
		index.targets[id] = &gatewayTargetDescriptor{Public: &models.GatewayTarget{
			ID: id, Name: name, Kind: kind, Gateway: gateway, Supported: true,
			DesiredState: "available", EffectState: "unverified", Version: state.Version,
		}, TagName: tagName, TagTitle: kind, Options: options, Materialize: id != "default"}
	}
	addBuiltIn("default", "跟随网络默认", "default", "")
	if state.LAN.LanAddr != "" {
		addBuiltIn("self", "本机路由", "self", state.LAN.LanAddr)
	}
	if state.LAN.Nexthop != "" && state.LAN.Nexthop != state.LAN.LanAddr {
		addBuiltIn("upstream", "上级路由", "upstream", state.LAN.Nexthop)
	}
	stored := map[string]DhcpTagRecord{}
	if state.DHCP != nil {
		for _, record := range state.DHCP.Tags {
			stored[record.TagName] = record
		}
		for _, tag := range buildGlobalDhcpTags(state.LAN, state.DHCP) {
			if tag == nil || tag.TagName == "" {
				continue
			}
			targetID, kind, name := gatewayTargetIdentity(tag.TagTitle, tag.Gateway, state.LAN)
			storedRecord, isStored := stored[tag.TagName]
			if storedRecord.TargetID != "" && (storedRecord.TargetKind == "bypass" || storedRecord.TargetKind == "custom") {
				targetID, kind, name = storedRecord.TargetID, storedRecord.TargetKind, gatewayTargetCustomName(tag.TagTitle)
			}
			if targetID == "default" {
				index.tagToID[tag.TagName] = targetID
				continue
			}
			descriptor := index.targets[targetID]
			if descriptor == nil {
				supported, reasons, dns := gatewayTargetOptions(tag.DhcpOption, tag.Gateway)
				descriptor = &gatewayTargetDescriptor{Public: &models.GatewayTarget{
					ID: targetID, Name: name, Kind: kind, Gateway: tag.Gateway, DNS: dns,
					Supported: supported, Reasons: reasons, DesiredState: "available", EffectState: "unverified", Version: state.Version,
				}}
				index.targets[targetID] = descriptor
			}
			index.tagToID[tag.TagName] = targetID
			if descriptor.TagName == "" || isStored {
				descriptor.TagName = tag.TagName
				descriptor.TagTitle = tag.TagTitle
				canonical := gatewayAndDNSOptions(tag.Gateway)
				if descriptor.Public.Supported {
					descriptor.Options = canonical
					descriptor.Public.DNS = []string{tag.Gateway}
				} else {
					descriptor.Options = append([]string(nil), tag.DhcpOption...)
				}
				descriptor.Stored = isStored
				descriptor.Materialize = !isStored || (descriptor.Public.Supported && !sameStringSet(tag.DhcpOption, canonical))
			}
		}
	}
	for _, descriptor := range index.targets {
		if descriptor.Public.Gateway != "" && state.UnreachableGateways[descriptor.Public.Gateway] {
			descriptor.Public.EffectState = "unreachable"
			descriptor.Public.Reasons = uniqueSortedStrings(append(descriptor.Public.Reasons, "gateway_unreachable"))
		}
	}
	return index
}

func gatewayAndDNSOptions(gateway string) []string {
	if strings.TrimSpace(gateway) == "" {
		return []string{}
	}
	return []string{"3," + gateway, "6," + gateway}
}

func sameStringSet(left, right []string) bool {
	left = uniqueSortedStrings(left)
	right = uniqueSortedStrings(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func gatewayTargetIdentity(title, gateway string, lan LanStatusSnapshot) (string, string, string) {
	switch {
	case title == "default":
		return "default", "default", "跟随网络默认"
	case gateway != "" && gateway == lan.LanAddr:
		return "self", "self", "本机路由"
	case gateway != "" && gateway == lan.Nexthop:
		return "upstream", "upstream", "上级路由"
	case title == "floatip":
		return gatewayOpaqueID("floating", gateway), "floating", "浮动网关"
	case title == "bypass":
		return gatewayOpaqueID("bypass", gateway), "bypass", "旁路由"
	default:
		return gatewayOpaqueID("custom", title+"|"+gateway), "custom", gatewayTargetCustomName(title)
	}
}

func gatewayTargetOptions(options []string, gateway string) (bool, []string, []string) {
	supported := gateway != ""
	reasons := []string{}
	dns := []string{}
	for _, option := range options {
		parts := splitDhcpOption(option)
		if len(parts) != 2 || (parts[0] != "3" && parts[0] != "6") {
			supported = false
			reasons = append(reasons, "unsupported_dhcp_option")
			continue
		}
		if parts[0] == "6" {
			dns = append(dns, parts[1])
		}
	}
	if gateway == "" {
		reasons = append(reasons, "gateway_missing")
	}
	return supported, uniqueSortedStrings(reasons), uniqueSortedStrings(dns)
}

func countGatewayReferences(index gatewayTargetIndex, hosts []gatewayPolicyHost) {
	for _, host := range hosts {
		if target := index.targets[index.targetIDForTag(host.TagName)]; target != nil {
			target.Public.ReferenceCount++
		}
	}
}

func findGatewayPolicyDevice(ctx context.Context, inventory *DeviceInventoryModule, deviceID string) (*models.DeviceInventoryItem, error) {
	snapshot, err := inventory.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	if snapshot == nil || snapshot.Result == nil {
		return nil, errors.New("device inventory is unavailable")
	}
	for _, device := range snapshot.Result.Devices {
		if device != nil && device.DeviceID == deviceID {
			return device, nil
		}
	}
	return nil, errors.New("device was not found")
}

func gatewayInPrefixes(raw string, prefixes []netip.Prefix) bool {
	address, err := netip.ParseAddr(raw)
	if err != nil {
		return false
	}
	for _, prefix := range prefixes {
		if prefix.Contains(address.Unmap()) {
			return true
		}
	}
	return false
}

func gatewayOpaqueID(kind, value string) string {
	sum := sha256.Sum256([]byte(value))
	return kind + ":" + hex.EncodeToString(sum[:6])
}

func gatewayTargetCustomName(title string) string {
	if title = strings.TrimSpace(title); title != "" {
		return title
	}
	return "自定义网关"
}

func gatewayTargetRank(kind string) int {
	for index, value := range []string{"default", "self", "upstream", "bypass", "floating", "custom"} {
		if kind == value {
			return index
		}
	}
	return 99
}

func requestAction(request *models.GatewayAssignmentRequest) string {
	if request == nil {
		return ""
	}
	return request.Action
}

func requestTargetID(request *models.GatewayAssignmentRequest) string {
	if request == nil {
		return ""
	}
	return request.TargetID
}

func gatewayPlanSummary(plan gatewayPolicyExecutionPlan) string {
	return fmt.Sprintf("%s:%s:%s", plan.Public.Action, plan.Public.DeviceID, plan.Public.TargetID)
}

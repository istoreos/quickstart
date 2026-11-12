package service

import (
	"context"
	"errors"
	"sort"

	"github.com/istoreos/quickstart/backend/models"
)

func (module *DeviceTrafficModule) snapshotAll(ctx context.Context) (*models.DeviceTrafficResponse, error) {
	resolver, _ := ctx.Value(lanScopeResolverKey{}).(*LANScopeModule)
	if resolver == nil {
		return nil, errors.New("traffic network resolver unavailable")
	}
	scopes, err := resolver.List(ctx)
	if err != nil {
		return nil, err
	}
	result := &models.DeviceTrafficResult{Items: []*models.DeviceTrafficItem{}, Health: &models.DeviceInventoryHealth{State: "ready"}, NetworkHealth: map[string]*models.DeviceInventoryHealth{}}
	for _, scope := range scopes {
		response, err := module.Snapshot(withLANScope(ctx, scope))
		if err != nil {
			return nil, err
		}
		result.Items = append(result.Items, response.Result.Items...)
		result.NetworkHealth[scope.NetworkID] = response.Result.Health
		if response.Result.Health.State != "ready" {
			result.Health.State = "partial"
		}
	}
	inventory, err := module.inventory.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	for _, device := range inventory.Result.Devices {
		if device != nil && device.NetworkID == "" {
			result.Health.State = "partial"
			result.Items = append(result.Items, &models.DeviceTrafficItem{DeviceID: device.DeviceID, State: "unavailable"})
		}
	}
	sort.Slice(result.Items, func(i, j int) bool {
		if result.Items[i].NetworkID != result.Items[j].NetworkID {
			return result.Items[i].NetworkID < result.Items[j].NetworkID
		}
		return result.Items[i].DeviceID < result.Items[j].DeviceID
	})
	return &models.DeviceTrafficResponse{Result: result}, nil
}

func (module *TrafficInsightsModule) recordScopedResponse(ctx context.Context, response *models.DeviceTrafficResponse) error {
	if selectedLAN(ctx) != "all" {
		return module.Record(ctx, response)
	}
	resolver, _ := ctx.Value(lanScopeResolverKey{}).(*LANScopeModule)
	if resolver == nil {
		return errors.New("traffic network resolver unavailable")
	}
	grouped := map[string][]*models.DeviceTrafficItem{}
	for _, item := range response.Result.Items {
		if item != nil && item.NetworkID != "" {
			grouped[item.NetworkID] = append(grouped[item.NetworkID], item)
		}
	}
	var failures []error
	for network, items := range grouped {
		scope, err := resolver.resolve(ctx, network, false)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if err := module.Record(withLANScope(ctx, scope), &models.DeviceTrafficResponse{Result: &models.DeviceTrafficResult{Items: items}}); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

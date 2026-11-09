package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/istoreos/quickstart/backend/models"
)

func (backend *ServiceBackend) lanScopeModule() *LANScopeModule {
	backend.mu.Lock()
	defer backend.mu.Unlock()
	if backend.lanScopes == nil {
		backend.lanScopes = NewLANScopeModule()
	}
	return backend.lanScopes
}

func (backend *ServiceBackend) GetLANNetworks(ctx context.Context) (*models.LANNetworksResponse, error) {
	networks, err := backend.lanScopeModule().List(ctx)
	if err != nil {
		return nil, err
	}
	result := &models.LANNetworksResult{Networks: networks, ScopeVersion: 1}
	for _, network := range networks {
		if network.NetworkID == defaultLAN {
			result.DefaultNetwork = defaultLAN
		}
	}
	return &models.LANNetworksResponse{Result: result}, nil
}

// PrepareLANRequest is the single request boundary for both legacy and v2 LAN
// routes. The selected scope is request-local and is revalidated for mutations.
func (backend *ServiceBackend) PrepareLANRequest(ctx context.Context, request *http.Request) (context.Context, error) {
	if request.URL.Path == "/cgi-bin/luci/istore/lanctrl/v2/networks/" {
		return ctx, nil
	}
	network, err := requestedLAN(request)
	if err != nil {
		return nil, err
	}
	if network == "all" {
		if request.Method == http.MethodGet && (request.URL.Path == "/cgi-bin/luci/istore/lanctrl/v2/devices/" || request.URL.Path == "/cgi-bin/luci/istore/lanctrl/v2/device-traffic/") {
			return context.WithValue(withLANScope(ctx, &models.LANScope{NetworkID: "all"}), lanScopeResolverKey{}, backend.lanScopeModule()), nil
		}
		return nil, &PolicyError{Code: "network_selection_required", Message: "此操作需要选择一个局域网"}
	}
	write := request.Method != http.MethodGet && request.Method != http.MethodHead
	scope, err := backend.lanScopeModule().resolve(ctx, network, write)
	if err != nil {
		return nil, err
	}
	ctx = withLANScope(ctx, scope)
	ctx = context.WithValue(ctx, lanScopeResolverKey{}, backend.lanScopeModule())
	metadata := strings.HasSuffix(request.URL.Path, "/management-probe/") || strings.HasSuffix(request.URL.Path, "/network-webhook/") || request.URL.Path == "/cgi-bin/luci/istore/lanctrl/v2/device-profile/" || request.URL.Path == "/cgi-bin/luci/istore/lanctrl/v2/device-classification/" || request.URL.Path == "/cgi-bin/luci/istore/lanctrl/v2/devices/"
	if write && strings.Contains(request.URL.Path, "/capability-action/") && request.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(request.Body, 1<<20))
		if err != nil {
			return nil, err
		}
		request.Body = io.NopCloser(bytes.NewReader(raw))
		var action struct {
			Action string `json:"action"`
		}
		if json.Unmarshal(raw, &action) == nil && action.Action == "install" {
			metadata = true
		}
	}
	if write && !metadata {
		if err := backend.lanScopeModule().requireWritable(ctx); err != nil {
			return nil, err
		}
	}
	path := strings.TrimPrefix(request.URL.Path, "/cgi-bin/luci/istore/lanctrl/v2/")
	if strings.HasPrefix(path, "floating-gateway/") {
		if err := requireDefaultFloatingLAN(ctx); err != nil {
			return nil, err
		}
	}
	if scope.NetworkID != defaultLAN {
		if strings.HasPrefix(path, "rate-limit-migration/") {
			return nil, &PolicyError{Code: "provider_scope_unsupported", Message: "兼容限速迁移仅用于原局域网"}
		}
		if !strings.HasPrefix(request.URL.Path, "/cgi-bin/luci/istore/lanctrl/v2/") && request.URL.Path != "/cgi-bin/luci/istore/lanctrl/globalConfigs/" && request.URL.Path != "/cgi-bin/luci/istore/lanctrl/listStaticDevices/" {
			return nil, &PolicyError{Code: "legacy_endpoint_scope_unsupported", Message: "此兼容入口仅支持原局域网，请使用新版设备管理"}
		}
	}

	return ctx, nil
}

func requestedLAN(request *http.Request) (string, error) {
	network := ""
	set := func(value string) error {
		if value == "" || (network != "" && value != network) {
			return &PolicyError{Code: "invalid_network", Message: "请求中的网络目标为空或不一致"}
		}
		network = value
		return nil
	}
	for _, key := range []string{"network", "lan"} {
		if values, present := request.URL.Query()[key]; present {
			if len(values) != 1 {
				return "", errors.New("exactly one network target is required")
			}
			if err := set(values[0]); err != nil {
				return "", err
			}
		}
	}
	if request.Body != nil && request.Method == http.MethodPost {
		body, err := io.ReadAll(io.LimitReader(request.Body, (1<<20)+1))
		if err != nil || len(body) > 1<<20 {
			return "", errors.New("invalid LAN request body")
		}
		request.Body = io.NopCloser(bytes.NewReader(body))
		var input struct {
			NetworkID *string `json:"networkId"`
		}
		if len(bytes.TrimSpace(body)) > 0 {
			if err := json.Unmarshal(body, &input); err != nil {
				return "", errors.New("invalid LAN request JSON")
			}
		}
		if input.NetworkID != nil {
			if err := set(*input.NetworkID); err != nil {
				return "", err
			}
		}
	}
	if network == "" {
		network = defaultLAN
	}
	return network, nil
}

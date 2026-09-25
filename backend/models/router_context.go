package models

// RouterContextEvidence is a bounded, user-safe explanation of one topology fact.
type RouterContextEvidence struct {
	Kind       string `json:"kind"`
	Source     string `json:"source"`
	Value      string `json:"value,omitempty"`
	Confidence string `json:"confidence"`
}

type RouteEditability struct {
	Editable bool   `json:"editable"`
	Reason   string `json:"reason"`
	Guidance string `json:"guidance,omitempty"`
}

// RouterContext describes one selected LAN. It is not a global router-role flag.
type RouterContext struct {
	LAN                 string                   `json:"lan"`
	TopologyPosition    string                   `json:"topologyPosition"`
	LocalAddress        string                   `json:"localAddress,omitempty"`
	UpstreamGateways    []string                 `json:"upstreamGateways"`
	LocalDHCPConfigured bool                     `json:"localDhcpConfigured"`
	LocalDHCPHealthy    bool                     `json:"localDhcpHealthy"`
	DHCPAuthority       string                   `json:"dhcpAuthority"`
	ExternalDHCPServers []string                 `json:"externalDhcpServers"`
	RouteEditability    *RouteEditability        `json:"routeEditability"`
	Evidence            []*RouterContextEvidence `json:"evidence"`
}

type RouterContextResponse struct {
	JSONResponse
	Result *RouterContext `json:"result,omitempty"`
}

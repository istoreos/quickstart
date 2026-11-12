package models

// LANScope describes a logical network, not a physical switch port.
type LANScope struct {
	NetworkID      string           `json:"networkId"`
	DisplayName    string           `json:"displayName"`
	Device         string           `json:"device"`
	Aliases        []string         `json:"aliases"`
	State          string           `json:"state"`
	IPv4Prefixes   []string         `json:"ipv4Prefixes"`
	IPv6Prefixes   []string         `json:"ipv6Prefixes"`
	FirewallZones  []string         `json:"firewallZones"`
	DHCPBindings   []LANDHCPBinding `json:"dhcpBindings"`
	Version        string           `json:"version"`
	ReadOnlyReason string           `json:"readOnlyReason,omitempty"`
}

type LANDHCPBinding struct {
	RuntimeInstance string `json:"runtimeInstance,omitempty"`
	Section         string `json:"section"`
	Instance        string `json:"instance"`
	LeaseFile       string `json:"leaseFile"`
}

type LANNetworksResult struct {
	Networks       []*LANScope `json:"networks"`
	DefaultNetwork string      `json:"defaultNetwork,omitempty"`
	ScopeVersion   int         `json:"scopeVersion"`
}

type LANNetworksResponse struct {
	JSONResponse
	Result *LANNetworksResult `json:"result,omitempty"`
}

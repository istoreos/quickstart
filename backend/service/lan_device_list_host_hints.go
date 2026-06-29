package service

type ubusHostHintItem struct {
	Name      string   `json:"name"`
	IPAddrs   []string `json:"ipaddrs"`
	IPv6Addrs []string `json:"ip6addrs"`
}

type ubusHostHintMap map[string]*ubusHostHintItem

package service

import (
	"bufio"
	"context"
	"io"
	"net/netip"
	"strconv"
	"strings"

	"github.com/istoreos/quickstart/backend/models"
)

func scopeLANPrefixes(scope *models.LANScope) []netip.Prefix {
	prefixes := []netip.Prefix{}
	for _, raw := range append(append([]string{}, scope.IPv4Prefixes...), scope.IPv6Prefixes...) {
		if prefix, err := netip.ParsePrefix(raw); err == nil {
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes
}

func parseLANInventoryARP(reader io.Reader, scope *models.LANScope) []deviceInventoryObservation {
	items := []deviceInventoryObservation{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 6 || fields[5] != scope.Device || !validIPv4(fields[0]) {
			continue
		}
		flags, err := strconv.ParseUint(fields[2], 0, 32)
		if err != nil || flags&2 == 0 {
			continue
		}
		mac := normalizeInventoryMAC(fields[3])
		if mac == "" || mac == "00:00:00:00:00:00" {
			continue
		}
		items = append(items, deviceInventoryObservation{NetworkID: scope.NetworkID, MAC: mac, IPv4: fields[0], Online: true})
	}
	return items
}

func leaseLANOwner(address, file string, scopes []*models.LANScope) string {
	parsed, err := netip.ParseAddr(address)
	if err != nil {
		return unassignedLAN
	}
	owner := ""
	for _, scope := range scopes {
		if !addressInPrefixes(parsed, scopeLANPrefixes(scope)) {
			continue
		}
		bound := len(scope.DHCPBindings) == 0
		for _, binding := range scope.DHCPBindings {
			bound = bound || binding.LeaseFile == file
		}
		if !bound {
			continue
		}
		if owner != "" {
			return unassignedLAN
		}
		owner = scope.NetworkID
	}
	if owner == "" {
		return unassignedLAN
	}
	return owner
}

// Host hints are global and may describe the same MAC on several links. Keep
// only network-local addresses, and do not spread a global hostname across LANs.
func scopeInventoryHints(snapshot *deviceInventorySourceSnapshot, scopes []*models.LANScope) {
	macNetworks := map[string]map[string]bool{}
	proof := map[string]bool{}
	for _, observation := range append(append([]deviceInventoryObservation{}, snapshot.ARP...), snapshot.NDP...) {
		mac := normalizeInventoryMAC(observation.MAC)
		if macNetworks[mac] == nil {
			macNetworks[mac] = map[string]bool{}
		}
		macNetworks[mac][observation.NetworkID] = true
		for _, address := range []string{observation.IPv4, observation.IPv6} {
			proof[inventoryLANKey(observation.NetworkID, mac+"/"+address)] = true
		}
	}
	for network, hints := range snapshot.NetworkHints {
		for mac, hint := range hints {
			filter := func(addresses []string) []string {
				result := []string{}
				for _, raw := range addresses {
					address, err := netip.ParseAddr(raw)
					if err != nil {
						continue
					}
					matches := 0
					for _, scope := range scopes {
						if addressInPrefixes(address, scopeLANPrefixes(scope)) {
							matches++
						}
					}
					if matches == 1 || proof[inventoryLANKey(network, mac+"/"+raw)] {
						result = append(result, raw)
					}
				}
				return result
			}
			hint.IPAddrs, hint.IPv6Addrs = filter(hint.IPAddrs), filter(hint.IPv6Addrs)
			if len(macNetworks[mac]) > 1 {
				hint.Hostname = ""
			}
			if len(hint.IPAddrs)+len(hint.IPv6Addrs) == 0 {
				delete(hints, mac)
			} else {
				hints[mac] = hint
			}
		}
	}
}

func readLANWifiAssociations(ctx context.Context) map[string]map[string]struct{} {
	result := map[string]map[string]struct{}{}
	var radios map[string]struct {
		Interfaces []struct {
			IfName string `json:"ifname"`
			Config struct {
				Network []string `json:"network"`
			} `json:"config"`
		} `json:"interfaces"`
	}
	if UbusCallWithObject(ctx, "network.wireless status", &radios) != nil {
		return result
	}
	for _, radio := range radios {
		for _, iface := range radio.Interfaces {
			if iface.IfName == "" || strings.ContainsAny(iface.IfName, " /\t\r\n") {
				continue
			}
			var clients struct {
				Clients map[string]struct{} `json:"clients"`
			}
			if UbusCallWithObject(ctx, "hostapd."+iface.IfName+" get_clients", &clients) != nil {
				continue
			}
			for _, network := range iface.Config.Network {
				if result[network] == nil {
					result[network] = map[string]struct{}{}
				}
				for mac := range clients.Clients {
					if normalized := normalizeInventoryMAC(mac); normalized != "" {
						result[network][normalized] = struct{}{}
					}
				}
			}
		}
	}
	return result
}

package service

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net/netip"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type systemDeviceInventorySource struct{}

func (systemDeviceInventorySource) Read(ctx context.Context) deviceInventorySourceSnapshot {
	snapshot := deviceInventorySourceSnapshot{
		HostHints: map[string]HostHintSnapshot{},
		WifiMACs:  map[string]struct{}{},
	}

	devices, err := NewDefaultDeviceInventoryReader().ReadInventory(ctx)
	if err != nil {
		snapshot.Reasons = append(snapshot.Reasons, "arp_unavailable")
	} else {
		for _, device := range devices {
			if device != nil {
				snapshot.ARP = append(snapshot.ARP, deviceInventoryObservation{MAC: device.Mac, IPv4: device.IP, Online: true})
			}
		}
	}

	leases, err := readDeviceInventoryDHCPv4("/tmp/dhcp.leases", time.Now())
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		snapshot.Reasons = append(snapshot.Reasons, "dhcpv4_unavailable")
	} else {
		snapshot.DHCPv4 = leases
	}

	ndp, err := readDeviceInventoryNDP(ctx, "br-lan")
	if err != nil {
		snapshot.Reasons = append(snapshot.Reasons, "ndp_unavailable")
	} else {
		snapshot.NDP = ndp
	}

	dhcpv6, err := readDeviceInventoryDHCPv6("/tmp/hosts/odhcpd", time.Now(), "br-lan")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		snapshot.Reasons = append(snapshot.Reasons, "dhcpv6_unavailable")
	} else {
		snapshot.DHCPv6 = dhcpv6
	}

	rawHints := ubusHostHintMap{}
	if err := UbusCallWithObject(ctx, "luci-rpc getHostHints", &rawHints); err != nil {
		snapshot.Reasons = append(snapshot.Reasons, "host_hints_unavailable")
	} else {
		prefixes, prefixErr := readSystemLANPrefixes()
		if prefixErr != nil {
			snapshot.Reasons = append(snapshot.Reasons, "lan_prefixes_unavailable")
		} else {
			snapshot.HostHints = buildLANHostHints(rawHints, prefixes, inventoryMACSet(ndp))
		}
	}

	driver := wifiSelect()
	if driver != nil {
		wifiMACs, err := driver.AssocMacList(ctx)
		if err != nil {
			snapshot.Reasons = append(snapshot.Reasons, "wifi_assoc_unavailable")
		} else {
			for mac := range wifiMACs {
				if normalized := normalizeInventoryMAC(mac); normalized != "" {
					snapshot.WifiMACs[normalized] = struct{}{}
				}
			}
		}
	}

	return snapshot
}

func readDeviceInventoryNDP(ctx context.Context, interfaceName string) ([]deviceInventoryObservation, error) {
	output, err := exec.CommandContext(ctx, "ip", "-6", "neigh", "show", "dev", interfaceName).Output()
	if err != nil {
		return nil, err
	}
	return parseDeviceInventoryNDP(strings.NewReader(string(output)))
}

func parseDeviceInventoryNDP(reader io.Reader) ([]deviceInventoryObservation, error) {
	result := make([]deviceInventoryObservation, 0)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		address, err := netip.ParseAddr(strings.TrimSpace(fields[0]))
		if err != nil || !address.Is6() || address.IsMulticast() || address.IsUnspecified() {
			continue
		}
		mac := ""
		failed := false
		for index, field := range fields {
			if field == "lladdr" && index+1 < len(fields) {
				mac = normalizeInventoryMAC(fields[index+1])
			}
			if field == "FAILED" || field == "INCOMPLETE" {
				failed = true
			}
		}
		if mac == "" || failed {
			continue
		}
		result = append(result, deviceInventoryObservation{MAC: mac, IPv6: address.String(), Online: true})
	}
	return result, scanner.Err()
}

func readDeviceInventoryDHCPv6(path string, now time.Time, interfaceName string) ([]deviceInventoryObservation, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return parseDeviceInventoryDHCPv6(file, now, interfaceName)
}

func parseDeviceInventoryDHCPv6(reader io.Reader, now time.Time, interfaceName string) ([]deviceInventoryObservation, error) {
	type addressLine struct{ address, hostname string }
	var pending addressLine
	result := make([]deviceInventoryObservation, 0)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			fields := strings.Fields(line)
			pending = addressLine{}
			if len(fields) >= 1 {
				address, err := netip.ParseAddr(fields[0])
				if err == nil && address.Is6() {
					pending.address = address.String()
					if len(fields) >= 2 && fields[1] != "*" {
						pending.hostname = fields[1]
					}
				}
			}
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "#"))
		if pending.address == "" || len(fields) < 6 || fields[0] != interfaceName {
			continue
		}
		expires, err := strconv.ParseInt(fields[4], 10, 64)
		if err != nil || (expires > 0 && expires <= now.Unix()) {
			pending = addressLine{}
			continue
		}
		duid := normalizeInventoryHexID(fields[1])
		iaid := normalizeInventoryHexID(fields[2])
		if duid == "" || iaid == "" {
			pending = addressLine{}
			continue
		}
		hostname := pending.hostname
		if fields[3] != "*" && fields[3] != "" {
			hostname = fields[3]
		}
		result = append(result, deviceInventoryObservation{IPv6: pending.address, DUID: duid, IAID: iaid, Hostname: hostname})
		pending = addressLine{}
	}
	return result, scanner.Err()
}

func readDeviceInventoryDHCPv4(path string, now time.Time) ([]deviceInventoryObservation, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make([]deviceInventoryObservation, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		expires, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil || (expires > 0 && expires <= now.Unix()) {
			continue
		}
		mac := normalizeInventoryMAC(fields[1])
		if mac == "" || !validIPv4(fields[2]) {
			continue
		}
		hostname := strings.TrimSpace(fields[3])
		if hostname == "*" {
			hostname = ""
		}
		result = append(result, deviceInventoryObservation{MAC: mac, IPv4: fields[2], Hostname: hostname})
	}
	return result, scanner.Err()
}

func buildLANHostHints(raw ubusHostHintMap, prefixes []netip.Prefix, ndpMACs map[string]struct{}) map[string]HostHintSnapshot {
	result := make(map[string]HostHintSnapshot)
	for mac, hint := range raw {
		if hint == nil {
			continue
		}
		normalized := normalizeInventoryMAC(mac)
		if normalized == "" {
			continue
		}
		addresses := make([]string, 0, len(hint.IPAddrs))
		for _, value := range hint.IPAddrs {
			address, err := netip.ParseAddr(strings.TrimSpace(value))
			if err != nil || !address.Is4() || !addressInPrefixes(address, prefixes) {
				continue
			}
			addresses = append(addresses, address.String())
		}
		ipv6Addresses := make([]string, 0, len(hint.IPv6Addrs))
		for _, value := range hint.IPv6Addrs {
			address, err := netip.ParseAddr(strings.TrimSpace(value))
			if err != nil || !address.Is6() || !addressInPrefixes(address, prefixes) {
				continue
			}
			if address.IsLinkLocalUnicast() {
				if _, ok := ndpMACs[normalized]; !ok {
					continue
				}
			}
			ipv6Addresses = append(ipv6Addresses, address.String())
		}
		if len(addresses) == 0 && len(ipv6Addresses) == 0 {
			continue
		}
		result[normalized] = HostHintSnapshot{
			Hostname: buildHostHintHostname(hint.Name), IPAddrs: uniqueSortedStrings(addresses), IPv6Addrs: uniqueSortedStrings(ipv6Addresses),
		}
	}
	return result
}

func buildLANIPv4HostHints(raw ubusHostHintMap, prefixes []netip.Prefix) map[string]HostHintSnapshot {
	return buildLANHostHints(raw, prefixes, map[string]struct{}{})
}

func inventoryMACSet(observations []deviceInventoryObservation) map[string]struct{} {
	result := make(map[string]struct{}, len(observations))
	for _, observation := range observations {
		if mac := normalizeInventoryMAC(observation.MAC); mac != "" {
			result[mac] = struct{}{}
		}
	}
	return result
}

func normalizeInventoryHexID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	for _, char := range value {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return ""
		}
	}
	return value
}

func addressInPrefixes(address netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

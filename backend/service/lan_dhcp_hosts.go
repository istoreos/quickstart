package service

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/istoreos/quickstart/backend/models"
)

var dnsmasqScopeTag = regexp.MustCompile(`^[a-zA-Z0-9_.:-]+$`)

func bindDHCPHost(tree uci.Tree, section string, scope *models.LANScope) error {
	if scope == nil {
		return nil
	}
	if !dnsmasqScopeTag.MatchString(scope.Device) || len(scope.DHCPBindings) != 1 || scope.DHCPBindings[0].Instance == "" {
		return errors.New("DHCP host requires a unique service binding and ingress device")
	}
	runtime := scope.DHCPBindings[0].RuntimeInstance
	if runtime == "" {
		runtime = scope.DHCPBindings[0].Instance
	}
	if strings.HasPrefix(runtime, "@") {
		return errors.New("DHCP service instance identity is unavailable")
	}
	if !tree.Set("dhcp", section, "quickstart_instance", scope.DHCPBindings[0].Instance) || !tree.Set("dhcp", section, "quickstart_network", scope.NetworkID) ||
		!tree.Set("dhcp", section, "instance", runtime) ||
		!tree.SetType("dhcp", section, "match_tag", uci.TypeList, scope.Device) {
		return errors.New("set DHCP host network binding")
	}
	return nil
}

const dhcpHostsBegin = "# quickstart scoped hosts begin"
const dhcpHostsEnd = "# quickstart scoped hosts end"

// OpenWrt skips host records without an IP/name before generating dnsmasq
// options. Use its supported extraconftext facility for those records only.
// The generated block lives in the same UCI file and transaction as the hosts;
// existing administrator configuration outside the block remains intact.
func compileScopedDHCPHosts(tree uci.Tree) error {
	lines := map[string][]string{}
	instances, _ := tree.GetSections("dhcp", "dnsmasq")
	hosts, _ := tree.GetSections("dhcp", "host")
	for _, host := range hosts {
		owner, _ := tree.GetLast("dhcp", host, "quickstart_network")
		ip, _ := tree.GetLast("dhcp", host, "ip")
		name, _ := tree.GetLast("dhcp", host, "name")
		tag, _ := tree.GetLast("dhcp", host, "tag")
		if owner == "" || ip != "" || name != "" || tag == "" {
			continue
		}
		instance, _ := tree.GetLast("dhcp", host, "quickstart_instance")
		mac, _ := tree.GetLast("dhcp", host, "mac")
		match, _ := tree.Get("dhcp", host, "match_tag")
		if len(match) != 1 || !dnsmasqScopeTag.MatchString(match[0]) || !dnsmasqScopeTag.MatchString(tag) || normalizeInventoryMAC(mac) == "" || !dhcpTreeHasSection(tree, "dnsmasq", instance) {
			return fmt.Errorf("invalid scoped DHCP host %s", host)
		}
		lines[instance] = append(lines[instance], "dhcp-host=tag:"+match[0]+","+mac+",set:"+tag)
	}
	for _, instance := range instances {
		text, _ := tree.GetLast("dhcp", instance, "extraconftext")
		text = strings.ReplaceAll(text, `\n`, "\n")
		start, end := strings.Index(text, dhcpHostsBegin), strings.Index(text, dhcpHostsEnd)
		if (start >= 0) != (end >= 0) || (start >= 0 && end < start) {
			return errors.New("managed DHCP host block is damaged; review configuration")
		}
		if start >= 0 {
			text = text[:start] + text[end+len(dhcpHostsEnd):]
		}
		text = strings.TrimRight(text, "\n")
		if len(lines[instance]) > 0 {
			confdir, _ := tree.GetLast("dhcp", instance, "confdir")
			if confdir != "" && (!strings.HasPrefix(confdir, "/") || strings.Contains(confdir, ",")) {
				return errors.New("custom dnsmasq include filtering does not support scoped host generation")
			}
			if text != "" {
				text += "\n"
			}
			text += dhcpHostsBegin + "\n" + strings.Join(lines[instance], "\n") + "\n" + dhcpHostsEnd
		}
		// An empty value also replaces the generated file on OpenWrt reload,
		// removing its last managed host instead of leaving stale runtime data.
		if text == "" && start >= 0 {
			text = "# no Quickstart scoped hosts"
		}
		if text != "" && !tree.Set("dhcp", instance, "extraconftext", strings.ReplaceAll(text, "\n", `\n`)) {
			return errors.New("set scoped DHCP host configuration")
		}
	}
	return nil
}

func verifyScopedDHCPHostSupport(scope *models.LANScope) error {
	if scope == nil {
		return nil
	}
	data, err := os.ReadFile("/etc/init.d/dnsmasq")
	if err != nil {
		return err
	}
	text := string(data)
	if !strings.Contains(text, "match_tag") || !strings.Contains(text, "extraconftext") {
		return &PolicyError{Code: "dhcp_scope_unsupported", Message: "此固件的 DHCP 服务不支持隔离设备路线"}
	}
	return nil
}

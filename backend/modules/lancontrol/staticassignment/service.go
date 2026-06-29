package staticassignment

import "fmt"

type Input struct {
	Action      string
	AssignedMAC string
	AssignedIP  string
	BindIP      bool
	Hostname    string
	TagName     string
	TagTitle    string
}

type HostRecord struct {
	SectionName string
	MAC         string
	IP          string
}

func HasDuplicateIPConflict(input Input, hosts []HostRecord) bool {
	if input.Action == "delete" || !input.BindIP || input.AssignedIP == "" {
		return false
	}
	for _, host := range hosts {
		if host.MAC != "" && host.MAC != input.AssignedMAC && host.IP == input.AssignedIP {
			return true
		}
	}
	return false
}

func BuildCommands(input Input, hosts []HostRecord, includeAutoTag bool) []string {
	commands := make([]string, 0, 16)
	for _, host := range hosts {
		if host.MAC == input.AssignedMAC {
			commands = append(commands,
				fmt.Sprintf("uci del dhcp.%s", host.SectionName),
				"uci commit dhcp",
			)
		}
	}
	if input.Action == "delete" {
		return commands
	}
	if includeAutoTag && input.TagName != "" {
		commands = append(commands, fmt.Sprintf("uci set dhcp.%s=tag", input.TagName))
	}
	commands = append(commands,
		"uci add dhcp host",
		"uci set dhcp.@host[-1].enabled='1'",
		fmt.Sprintf("uci set dhcp.@host[-1].mac='%s'", input.AssignedMAC),
	)
	if input.TagName != "" {
		commands = append(commands, fmt.Sprintf("uci set dhcp.@host[-1].tag='%s'", input.TagName))
	}
	if input.TagTitle != "" {
		commands = append(commands, fmt.Sprintf("uci set dhcp.@host[-1].tag_title='%s'", input.TagTitle))
	}
	if input.Hostname != "" {
		commands = append(commands, fmt.Sprintf("uci set dhcp.@host[-1].name='%s'", input.Hostname))
	}
	if input.BindIP && input.AssignedIP != "" {
		commands = append(commands, fmt.Sprintf("uci set dhcp.@host[-1].ip='%s'", input.AssignedIP))
	}
	return commands
}

package staticassignment

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

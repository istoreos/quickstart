package floatgateway

import "strings"

type Input struct {
	Enabled         bool
	Role            string
	SetIP           string
	CheckIP         string
	CheckURL        string
	CheckURLTimeout int64
}

type Config struct {
	Enabled            bool
	Role               string
	SetIP              string
	UseSetIP           bool
	ScalarCheckIP      string
	UseScalarCheckIP   bool
	CheckURL           string
	CheckURLTimeout    int64
	UseURLProbeSetting bool
	CheckIPs           []string
}

type StateSnapshot struct {
	Enabled bool
	SetIP   string
	CheckIP string
}

type DhcpTagSnapshot struct {
	SectionName string
}

type DhcpHostSnapshot struct {
	SectionName string
	Tag         string
}

type DhcpCleanupPlan struct {
	DeleteTagSections  []string
	DeleteHostSections []string
}

func BuildConfig(input Input) Config {
	switch input.Role {
	case "fallback":
		config := Config{
			Enabled:          input.Enabled,
			Role:             input.Role,
			SetIP:            input.SetIP,
			UseSetIP:         true,
			ScalarCheckIP:    input.CheckIP,
			UseScalarCheckIP: true,
		}
		return config
	case "main":
		config := Config{
			Enabled:            input.Enabled,
			Role:               input.Role,
			SetIP:              input.SetIP,
			UseSetIP:           true,
			CheckURL:           input.CheckURL,
			CheckURLTimeout:    input.CheckURLTimeout,
			UseURLProbeSetting: true,
			CheckIPs:           []string{input.CheckIP},
		}
		return config
	default:
		return Config{}
	}
}

func ShouldCleanupDhcp(state StateSnapshot, input Input) bool {
	return (state.Enabled && !input.Enabled) ||
		state.SetIP != input.SetIP ||
		state.CheckIP != input.CheckIP
}

func BuildDhcpCleanupPlan(tags []DhcpTagSnapshot, hosts []DhcpHostSnapshot) DhcpCleanupPlan {
	plan := DhcpCleanupPlan{
		DeleteTagSections:  make([]string, 0, len(tags)),
		DeleteHostSections: make([]string, 0, len(hosts)),
	}
	autoTags := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		if strings.HasPrefix(tag.SectionName, "t_auto_") {
			plan.DeleteTagSections = append(plan.DeleteTagSections, tag.SectionName)
			autoTags[tag.SectionName] = struct{}{}
		}
	}
	for _, host := range hosts {
		if _, ok := autoTags[host.Tag]; ok {
			plan.DeleteHostSections = append(plan.DeleteHostSections, host.SectionName)
		}
	}
	return plan
}

package service

import (
	"sort"
	"strings"
)

func BuildAutoDhcpPlan(lanStatus LanStatusSnapshot, state *LanDhcpState) DhcpTagPlan {
	plan := DhcpTagPlan{
		DhcpEnabled: true,
		Tags:        make([]DhcpTagRecord, 0, 2),
	}
	if state == nil {
		sortDhcpTagPlan(plan.Tags)
		return plan
	}

	plan.DhcpGateway = detectDhcpGateway(lanStatus, state.DhcpOptions)
	plan.DhcpEnabled = !state.DhcpIgnore
	if state.DhcpIgnore {
		plan.DhcpGateway = ""
	}

	if !state.DhcpIgnore && lanStatus.IsDefaultGateway {
		if plan.DhcpGateway == "" {
			appendDhcpTag(&plan.Tags, "default", lanStatus.LanAddr)
			appendDhcpTag(&plan.Tags, "parent", lanStatus.Nexthop)
		} else {
			appendDhcpTag(&plan.Tags, "default", plan.DhcpGateway)
			appendDhcpTag(&plan.Tags, "myself", lanStatus.LanAddr)
		}
	} else {
		appendDhcpTag(&plan.Tags, "default", lanStatus.LanAddr)
	}

	sortDhcpTagPlan(plan.Tags)
	return plan
}

func detectDhcpGateway(lanStatus LanStatusSnapshot, dhcpOptions []string) string {
	for _, val := range dhcpOptions {
		parts := splitDhcpOption(val)
		if len(parts) == 2 && parts[0] == "3" && parts[1] != lanStatus.LanAddr {
			return parts[1]
		}
	}
	return ""
}

func appendDhcpTag(tags *[]DhcpTagRecord, title string, gateway string) {
	tagName := ipToDhcpTag(gateway)
	if tagName == "" {
		return
	}
	*tags = append(*tags, DhcpTagRecord{
		TagName:     tagName,
		TagTitle:    title,
		AutoCreated: true,
		Gateway:     gateway,
		DhcpOption:  []string{"3," + gateway, "6," + gateway},
	})
}

func sortDhcpTagPlan(tags []DhcpTagRecord) {
	sort.Slice(tags, func(i, j int) bool {
		leftRank := dhcpTagTitleSortRank(tags[i].TagTitle)
		rightRank := dhcpTagTitleSortRank(tags[j].TagTitle)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		if tags[i].TagName != tags[j].TagName {
			return tags[i].TagName < tags[j].TagName
		}
		if tags[i].TagTitle != tags[j].TagTitle {
			return tags[i].TagTitle < tags[j].TagTitle
		}
		return tags[i].Gateway < tags[j].Gateway
	})
}

func dhcpTagTitleSortRank(title string) int {
	switch title {
	case "default":
		return 0
	case "parent", "myself":
		return 1
	default:
		return 2
	}
}

func splitDhcpOption(value string) []string {
	return strings.Split(value, ",")
}

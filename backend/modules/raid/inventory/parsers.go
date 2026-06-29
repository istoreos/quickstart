package inventory

import (
	"regexp"
	"strings"
)

func ParsePartedInfo(keys []string, line string) map[string]string {
	detail := make(map[string]string)
	matches := matchAllWithStr(line, `(.*?)[:;]`)
	for i, v := range matches {
		if i >= len(keys) {
			break
		}
		detail[keys[i]] = string(v[1])
	}
	return detail
}

func DiskToPart(device string, number string) string {
	if matched, _ := regexp.MatchString(`\d$`, device); matched {
		return device + "p" + number
	}
	return device + number
}

func ParsePartitionUsage(dfOutput string) (string, string, string) {
	match := matchStringOnce(dfOutput, `\s+\d+\s+(\d+)\s+(\d+)\s+(\d+)\%\s*?`)
	if match == nil {
		return "", "", ""
	}
	return match[1], match[2], match[3]
}

func ParseMountPoint(mountOutput string, partition string) string {
	pattern := `\/dev\/` + partition + " on ([^ ]*)"
	match := matchStringOnce(mountOutput, pattern)
	if match == nil {
		return ""
	}
	return match[1]
}

func ParseRaidMember(mdstat string, partition string) string {
	lines := strings.SplitAfter(mdstat, "\n")
	for _, line := range lines {
		match := matchStringOnce(line, "(md.*?):(.+)")
		if match == nil {
			continue
		}
		if strings.Contains(match[2], partition) {
			return "Raid Member: " + match[1]
		}
	}
	return ""
}

func ParseMDDetail(path string, output string) map[string]string {
	detail := make(map[string]string)
	matched, _ := regexp.MatchString(`^/dev/md\d+$`, path)
	if !matched {
		return detail
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		match := matchStringOnce(line, `^\s*(.+) : (.+)`)
		if match == nil || match[1] == "" {
			continue
		}
		detail[match[1]] = match[2]
	}
	return detail
}

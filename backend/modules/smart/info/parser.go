package info

import (
	"regexp"
	"strings"

	"github.com/linkease/quick-start/istore-backend/models"
)

func ParseSmartctlInfo(base models.SmartInfo, stdout string) *models.SmartInfo {
	model := base
	lines := strings.Split(stdout, "\n")
	var section int
	for _, line := range lines {
		var attrib, val string
		if match := matchStringOnce(line, `^=== START OF (.*) SECTION ===`); match != nil {
			attrib = match[1]
			if checkStringMatch(`INFORMATION`, attrib) {
				section = 1
			} else if checkStringMatch(`SMART DATA`, attrib) {
				section = 2
			}
			continue
		}

		if section == 1 {
			if match := matchStringOnce(line, `^(.+?):\s+(.+)`); match != nil {
				attrib = match[1]
				val = match[2]
			}
		} else if section == 2 && model.NvmeVer != "" {
			if match := matchStringOnce(line, `^(.+?):\s+(.+)`); match != nil {
				attrib = match[1]
				val = match[2]
			}
			if model.Health == "" {
				if match := matchStringOnce(line, `.+overall-health.+: (.+)`); match != nil {
					model.Health = match[1]
				}
			}
		} else if section == 2 {
			if match := matchStringOnce(line, `^([0-9 ]+)\s+[^ ]+\s+[POSRCK-]+\s+[0-9-]+\s+[0-9-]+\s+[0-9-]+\s+[0-9-]+\s+([0-9-]+)`); match != nil {
				attrib = strings.TrimSpace(match[1])
				val = match[2]
			}
			if model.Health == "" {
				if match := matchStringOnce(line, `.+overall-health.+: (.+)`); match != nil {
					model.Health = match[1]
				}
			}
		}

		if attrib == "" {
			if section != 2 {
				section = 0
			}
		} else if attrib == "Power mode is" || attrib == "Power mode was" {
			if match := matchStringOnce(val, `(\S+)`); match != nil {
				model.Status = match[1]
			}
		} else if attrib == "Serial Number" {
			model.Serial = val
		} else if attrib == "Rotation Rate" {
			model.RotaRate = val
		} else if attrib == "SATA Version is" {
			model.SataVer = val
		} else if attrib == "NVMe Version" {
			model.NvmeVer = val
		} else if attrib == "194" || attrib == "Temperature" {
			if match := matchStringOnce(val, `(\d+)`); match != nil {
				model.Temp = match[1] + "°C"
			}
		}
	}
	return &model
}

func checkStringMatch(pattern string, str string) bool {
	return regexp.MustCompile(pattern).MatchString(str)
}

func matchStringOnce(str string, pattern string) []string {
	return regexp.MustCompile(pattern).FindStringSubmatch(str)
}

package service

import "github.com/digineo/go-uci"

func haveUciSection(config, sectionType, sectionName string) bool {
	uci.LoadConfig(config, true)
	secs, ok := uci.GetSections(config, sectionType)
	if !ok {
		return false
	}
	for _, sec := range secs {
		if sec == sectionName {
			return true
		}
	}
	return false
}

package uci

import gouci "github.com/digineo/go-uci"

var loadConfig = gouci.LoadConfig
var get = gouci.Get

func ListOption(config string, section string, option string) []string {
	if err := loadConfig(config, true); err != nil {
		return []string{}
	}
	values, ok := get(config, section, option)
	if !ok || len(values) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(values))
	result = append(result, values...)
	return result
}

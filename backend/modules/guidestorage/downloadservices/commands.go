package downloadservices

import (
	"fmt"
	"strings"
)

func ParseTrackers(trackers string) []string {
	splitter := func(r rune) bool {
		return r == '\r' || r == '\n' || r == ','
	}
	parts := strings.FieldsFunc(trackers, splitter)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}

	n := 0
	for _, part := range parts {
		if len(part) > 0 {
			parts[n] = part
			n++
		}
	}
	return parts[:n]
}

func BuildAria2ConfigCommands(input Aria2InitInput) []string {
	return []string{
		"uci set aria2.main.rpc_auth_method='token'",
		fmt.Sprintf("uci set aria2.main.config_dir='%v'", input.ConfigPath),
		fmt.Sprintf("uci set aria2.main.dir='%v'", input.DownloadPath),
		"uci set aria2.main.enabled='1'",
		fmt.Sprintf("uci set aria2.main.rpc_secret='%v'", input.RPCToken),
	}
}

func BuildAria2TrackerCommandBatches(trackers []string) [][]string {
	if len(trackers) == 0 {
		return nil
	}
	cmds := [][]string{{"uci delete aria2.main.bt_tracker"}}
	for _, tracker := range trackers {
		cmds = append(cmds, []string{fmt.Sprintf("uci add_list aria2.main.bt_tracker='%v'", tracker)})
	}
	return cmds
}

func BuildAria2RestartCommands() []string {
	return []string{"uci commit aria2", "/etc/init.d/aria2 restart"}
}

func BuildQbittorrentConfigCommands(input QbittorrentInitInput) []string {
	return []string{
		fmt.Sprintf("uci set qbittorrent.main.profile='%v'", input.ConfigPath),
		fmt.Sprintf("uci set qbittorrent.main.SavePath='%v'", input.DownloadPath),
		"uci set qbittorrent.main.enabled=1",
	}
}

func BuildQbittorrentRestartCommands() []string {
	return []string{"uci commit qbittorrent", "/etc/init.d/qbittorrent restart"}
}

func BuildTransmissionConfigCommands(input TransmissionInitInput) []string {
	return []string{
		fmt.Sprintf("uci set transmission.@transmission[0].config_dir='%v'", input.ConfigPath),
		fmt.Sprintf("uci set transmission.@transmission[0].download_dir='%v'", input.DownloadPath),
		"uci set transmission.@transmission[0].enabled=1",
	}
}

func BuildTransmissionRestartCommands() []string {
	return []string{"uci commit transmission", "/etc/init.d/transmission restart"}
}

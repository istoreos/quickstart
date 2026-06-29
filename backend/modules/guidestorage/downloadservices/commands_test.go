package downloadservices

import (
	"reflect"
	"testing"
)

func TestParseTrackersPreservesLegacySeparators(t *testing.T) {
	got := ParseTrackers("udp://a,\n udp://b\r\n\nudp://c,,")
	want := []string{"udp://a", "udp://b", "udp://c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

func TestBuildAria2Commands(t *testing.T) {
	config := BuildAria2ConfigCommands(Aria2InitInput{
		ConfigPath:   "/etc/aria2",
		DownloadPath: "/mnt/download",
		RPCToken:     "secret",
	})
	wantConfig := []string{
		"uci set aria2.main.rpc_auth_method='token'",
		"uci set aria2.main.config_dir='/etc/aria2'",
		"uci set aria2.main.dir='/mnt/download'",
		"uci set aria2.main.enabled='1'",
		"uci set aria2.main.rpc_secret='secret'",
	}
	if !reflect.DeepEqual(config, wantConfig) {
		t.Fatalf("unexpected aria2 config commands: %#v", config)
	}

	trackerCalls := BuildAria2TrackerCommandBatches([]string{"udp://a", "udp://b"})
	wantTrackerCalls := [][]string{
		{"uci delete aria2.main.bt_tracker"},
		{"uci add_list aria2.main.bt_tracker='udp://a'"},
		{"uci add_list aria2.main.bt_tracker='udp://b'"},
	}
	if !reflect.DeepEqual(trackerCalls, wantTrackerCalls) {
		t.Fatalf("unexpected tracker commands: %#v", trackerCalls)
	}
	if calls := BuildAria2TrackerCommandBatches(nil); len(calls) != 0 {
		t.Fatalf("expected no tracker commands for empty input, got %#v", calls)
	}

	if restart := BuildAria2RestartCommands(); !reflect.DeepEqual(restart, []string{"uci commit aria2", "/etc/init.d/aria2 restart"}) {
		t.Fatalf("unexpected aria2 restart commands: %#v", restart)
	}
}

func TestBuildQbittorrentAndTransmissionCommands(t *testing.T) {
	qbitConfig := BuildQbittorrentConfigCommands(QbittorrentInitInput{
		ConfigPath:   "/etc/qbit",
		DownloadPath: "/mnt/qbit",
	})
	wantQbitConfig := []string{
		"uci set qbittorrent.main.profile='/etc/qbit'",
		"uci set qbittorrent.main.SavePath='/mnt/qbit'",
		"uci set qbittorrent.main.enabled=1",
	}
	if !reflect.DeepEqual(qbitConfig, wantQbitConfig) {
		t.Fatalf("unexpected qbittorrent config commands: %#v", qbitConfig)
	}
	if restart := BuildQbittorrentRestartCommands(); !reflect.DeepEqual(restart, []string{"uci commit qbittorrent", "/etc/init.d/qbittorrent restart"}) {
		t.Fatalf("unexpected qbittorrent restart commands: %#v", restart)
	}

	transmissionConfig := BuildTransmissionConfigCommands(TransmissionInitInput{
		ConfigPath:   "/etc/transmission",
		DownloadPath: "/mnt/transmission",
	})
	wantTransmissionConfig := []string{
		"uci set transmission.@transmission[0].config_dir='/etc/transmission'",
		"uci set transmission.@transmission[0].download_dir='/mnt/transmission'",
		"uci set transmission.@transmission[0].enabled=1",
	}
	if !reflect.DeepEqual(transmissionConfig, wantTransmissionConfig) {
		t.Fatalf("unexpected transmission config commands: %#v", transmissionConfig)
	}
	if restart := BuildTransmissionRestartCommands(); !reflect.DeepEqual(restart, []string{"uci commit transmission", "/etc/init.d/transmission restart"}) {
		t.Fatalf("unexpected transmission restart commands: %#v", restart)
	}
}

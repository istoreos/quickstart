package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestGuideNetworkBasicsCommandAdaptersPreservePPPoEAndInterfaceBatches(t *testing.T) {
	originalRun := runGuideNetworkBasicsUCICommands
	defer func() {
		runGuideNetworkBasicsUCICommands = originalRun
	}()

	var got [][]string
	runGuideNetworkBasicsUCICommands = func(ctx context.Context, cmds []string) error {
		got = append(got, append([]string(nil), cmds...))
		return nil
	}

	if err := uciSetPppoeWithoutCommit(context.Background(), "user", "pass"); err != nil {
		t.Fatalf("unexpected pppoe error: %v", err)
	}
	if err := uciSetInterfaceWithoutCommit(context.Background(), "wan", "static", "255.255.255.0", "10.0.0.2", "10.0.0.1"); err != nil {
		t.Fatalf("unexpected interface error: %v", err)
	}

	want := [][]string{
		{
			"uci set network.wan.proto='pppoe'",
			"uci set network.wan.username='user'",
			"uci set network.wan.password='pass'",
			"uci del network.wan.dns",
			"uci del network.wan.peerdns",
			"uci del network.wan.defaultroute",
			"uci set network.lan.defaultroute=0",
		},
		{
			"uci set network.wan.defaultroute=0",
			"uci set network.lan.defaultroute=0",
			"uci del network.wan.defaultroute",
			"uci set network.wan.proto=static",
		},
		{
			"uci del network.wan.ipaddr",
			"uci del network.wan.netmask",
			"uci set network.wan.ipaddr='10.0.0.2'",
			"uci set network.wan.netmask='255.255.255.0'",
		},
		{
			"uci set network.wan.gateway=10.0.0.1",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected command batches: got=%#v want=%#v", got, want)
	}
}

func TestGuideNetworkBasicsDNSCommandAdapterPreservesManualBatches(t *testing.T) {
	originalRun := runGuideNetworkBasicsUCICommands
	defer func() {
		runGuideNetworkBasicsUCICommands = originalRun
	}()

	var got [][]string
	runGuideNetworkBasicsUCICommands = func(ctx context.Context, cmds []string) error {
		got = append(got, append([]string(nil), cmds...))
		return nil
	}

	if err := uciSetDNSWithoutCommit(context.Background(), "wan", "manual", []string{"1.1.1.1", "", "8.8.8.8"}); err != nil {
		t.Fatalf("unexpected dns error: %v", err)
	}

	want := [][]string{
		{
			"uci del network.wan.dns",
			"uci del network.wan.peerdns",
			"uci del network.lan.dns",
			"uci del network.lan.peerdns",
		},
		{
			"uci del network.wan.dns",
			"uci set network.wan.peerdns=0",
			"uci add_list network.wan.dns='1.1.1.1'",
			"uci add_list network.wan.dns='8.8.8.8'",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected command batches: got=%#v want=%#v", got, want)
	}
}

func TestGuideNetworkBasicsDNSCommandAdapterPreservesErrorBehavior(t *testing.T) {
	originalRun := runGuideNetworkBasicsUCICommands
	defer func() {
		runGuideNetworkBasicsUCICommands = originalRun
	}()

	defaultErr := errors.New("default cleanup failed")
	manualErr := errors.New("manual dns failed")
	var calls int
	runGuideNetworkBasicsUCICommands = func(ctx context.Context, cmds []string) error {
		calls++
		if calls == 1 {
			return defaultErr
		}
		return manualErr
	}

	err := uciSetDNSWithoutCommit(context.Background(), "wan", "manual", []string{"1.1.1.1"})
	if !errors.Is(err, manualErr) {
		t.Fatalf("expected manual error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected two command batches, got %d", calls)
	}

	calls = 0
	runGuideNetworkBasicsUCICommands = func(ctx context.Context, cmds []string) error {
		calls++
		return defaultErr
	}

	err = uciSetDNSWithoutCommit(context.Background(), "wan", "auto", nil)
	if err != nil {
		t.Fatalf("expected default cleanup error to be ignored, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one command batch, got %d", calls)
	}
}

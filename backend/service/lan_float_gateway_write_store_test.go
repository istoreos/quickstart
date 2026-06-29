package service

import (
	"context"
	"errors"
	"sync"
	"testing"
)

var lanFloatGatewayWriteStoreTestMu sync.Mutex

func TestBuildFloatGatewayWriteCommandsForFallbackRole(t *testing.T) {
	t.Parallel()

	commands := buildFloatGatewayWriteCommands(FloatGatewayWriteInput{
		Enabled: true,
		Role:    "fallback",
		CheckIP: "192.168.100.2",
		SetIP:   "192.168.100.3",
	})

	want := []string{
		"uci del floatip.main",
		"uci commit floatip",
		"uci set floatip.main=floatip",
		"uci set floatip.main.enabled='1'",
		"uci set floatip.main.role='fallback'",
		"uci set floatip.main.check_ip='192.168.100.2'",
		"uci set floatip.main.set_ip='192.168.100.3'",
	}
	if len(commands) != len(want) {
		t.Fatalf("len(commands) = %d, want %d", len(commands), len(want))
	}
	for i, expected := range want {
		if commands[i] != expected {
			t.Fatalf("commands[%d] = %q, want %q", i, commands[i], expected)
		}
	}
}

func TestBuildFloatGatewayWriteCommandsForMainRole(t *testing.T) {
	t.Parallel()

	commands := buildFloatGatewayWriteCommands(FloatGatewayWriteInput{
		Enabled:         true,
		Role:            "main",
		CheckIP:         "192.168.100.2",
		SetIP:           "192.168.100.3",
		CheckURL:        "https://example.com",
		CheckURLTimeout: 8,
	})

	want := []string{
		"uci del floatip.main",
		"uci commit floatip",
		"uci set floatip.main=floatip",
		"uci set floatip.main.enabled='1'",
		"uci set floatip.main.role='main'",
		"uci set floatip.main.set_ip='192.168.100.3'",
		"uci set floatip.main.check_url='https://example.com'",
		"uci set floatip.main.check_url_timeout='8'",
		"uci add_list floatip.main.check_ip='192.168.100.2'",
	}
	if len(commands) != len(want) {
		t.Fatalf("len(commands) = %d, want %d", len(commands), len(want))
	}
	for i, expected := range want {
		if commands[i] != expected {
			t.Fatalf("commands[%d] = %q, want %q", i, commands[i], expected)
		}
	}
}

func TestShouldCleanupFloatGatewayDhcpOnDisable(t *testing.T) {
	t.Parallel()

	if !shouldCleanupFloatGatewayDhcp(
		FloatGatewayStateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		FloatGatewayWriteInput{Enabled: false, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
	) {
		t.Fatal("expected cleanup when enabled changes from true to false")
	}
}

func TestShouldCleanupFloatGatewayDhcpOnSetIPOrCheckIPChange(t *testing.T) {
	t.Parallel()

	if !shouldCleanupFloatGatewayDhcp(
		FloatGatewayStateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		FloatGatewayWriteInput{Enabled: true, SetIP: "192.168.100.4", CheckIP: "192.168.100.2"},
	) {
		t.Fatal("expected cleanup when set ip changes")
	}
	if !shouldCleanupFloatGatewayDhcp(
		FloatGatewayStateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		FloatGatewayWriteInput{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.5"},
	) {
		t.Fatal("expected cleanup when check ip changes")
	}
}

func TestBuildFloatGatewayDhcpCleanupPlanDeletesAutoTagsAndBoundHosts(t *testing.T) {
	t.Parallel()

	plan := buildFloatGatewayDhcpCleanupPlan(
		[]FloatGatewayDhcpTagSnapshot{
			{SectionName: "t_auto_lan1"},
			{SectionName: "manual"},
			{SectionName: "t_auto_lan2"},
		},
		[]FloatGatewayDhcpHostSnapshot{
			{SectionName: "cfg01", Tag: "t_auto_lan1"},
			{SectionName: "cfg02", Tag: "manual"},
			{SectionName: "cfg03", Tag: "t_auto_lan2"},
		},
	)

	if len(plan.DeleteTagSections) != 2 {
		t.Fatalf("len(DeleteTagSections) = %d, want 2", len(plan.DeleteTagSections))
	}
	if plan.DeleteTagSections[0] != "t_auto_lan1" || plan.DeleteTagSections[1] != "t_auto_lan2" {
		t.Fatalf("DeleteTagSections = %+v", plan.DeleteTagSections)
	}
	if len(plan.DeleteHostSections) != 2 {
		t.Fatalf("len(DeleteHostSections) = %d, want 2", len(plan.DeleteHostSections))
	}
	if plan.DeleteHostSections[0] != "cfg01" || plan.DeleteHostSections[1] != "cfg03" {
		t.Fatalf("DeleteHostSections = %+v", plan.DeleteHostSections)
	}
}

func TestDefaultLanFloatGatewayWriteStoreExecutesFloatipThenDhcpCleanup(t *testing.T) {
	lanFloatGatewayWriteStoreTestMu.Lock()
	defer lanFloatGatewayWriteStoreTestMu.Unlock()

	originalExec := lanFloatGatewayWriteExec
	t.Cleanup(func() {
		lanFloatGatewayWriteExec = originalExec
	})

	var got []string
	lanFloatGatewayWriteExec = func(ctx context.Context, commands []string) error {
		_ = ctx
		got = append(got, commands...)
		return nil
	}

	store := NewDefaultLanFloatGatewayWriteStore()
	err := store.ApplyPlan(context.Background(), FloatGatewayWriteExecutionPlan{
		FloatCommands: buildFloatGatewayWriteCommands(FloatGatewayWriteInput{
			Enabled: true,
			Role:    "fallback",
			CheckIP: "192.168.100.2",
			SetIP:   "192.168.100.3",
		}),
		CleanupPlan: FloatGatewayDhcpCleanupPlan{
			DeleteTagSections:  []string{"t_auto_lan1"},
			DeleteHostSections: []string{"cfg01"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyPlan returned error: %v", err)
	}
	if len(got) < 9 {
		t.Fatalf("got too few commands: %+v", got)
	}
	wantSuffix := []string{
		"uci del dhcp.t_auto_lan1",
		"uci del dhcp.cfg01",
	}
	if got[len(got)-2] != wantSuffix[0] || got[len(got)-1] != wantSuffix[1] {
		t.Fatalf("cleanup commands = %+v", got[len(got)-2:])
	}
}

func TestDefaultLanFloatGatewayWriteStorePropagatesExecError(t *testing.T) {
	lanFloatGatewayWriteStoreTestMu.Lock()
	defer lanFloatGatewayWriteStoreTestMu.Unlock()

	originalExec := lanFloatGatewayWriteExec
	t.Cleanup(func() {
		lanFloatGatewayWriteExec = originalExec
	})

	lanFloatGatewayWriteExec = func(ctx context.Context, commands []string) error {
		_ = ctx
		_ = commands
		return errors.New("exec failed")
	}

	store := NewDefaultLanFloatGatewayWriteStore()
	err := store.ApplyPlan(context.Background(), FloatGatewayWriteExecutionPlan{})
	if err == nil || err.Error() != "exec failed" {
		t.Fatalf("err = %v, want exec failed", err)
	}
}

func TestDefaultLanFloatGatewayApplyDelegatesExpectedConfigs(t *testing.T) {
	lanFloatGatewayWriteStoreTestMu.Lock()
	defer lanFloatGatewayWriteStoreTestMu.Unlock()

	originalApply := lanFloatGatewayWriteCommitAndApply
	t.Cleanup(func() {
		lanFloatGatewayWriteCommitAndApply = originalApply
	})

	var got []string
	lanFloatGatewayWriteCommitAndApply = func(ctx context.Context, configs []string) error {
		_ = ctx
		got = append(got, configs...)
		return nil
	}

	apply := NewDefaultLanFloatGatewayApply()
	err := apply.Apply(context.Background(), []string{"floatip", "dhcp", "dnsmasq"})
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	if len(got) != 3 || got[0] != "floatip" || got[1] != "dhcp" || got[2] != "dnsmasq" {
		t.Fatalf("got configs = %+v", got)
	}
}

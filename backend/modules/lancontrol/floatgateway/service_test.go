package floatgateway_test

import (
	"reflect"
	"testing"

	"github.com/linkease/quick-start/istore-backend/modules/lancontrol/floatgateway"
)

func TestBuildConfigForFallbackRoleUsesScalarCheckIP(t *testing.T) {
	t.Parallel()

	config := floatgateway.BuildConfig(floatgateway.Input{
		Enabled: true,
		Role:    "fallback",
		CheckIP: "192.168.100.2",
		SetIP:   "192.168.100.3",
	})

	want := floatgateway.Config{
		Enabled:            true,
		Role:               "fallback",
		SetIP:              "192.168.100.3",
		UseSetIP:           true,
		ScalarCheckIP:      "192.168.100.2",
		UseScalarCheckIP:   true,
		UseURLProbeSetting: false,
	}
	if !reflect.DeepEqual(config, want) {
		t.Fatalf("config = %+v, want %+v", config, want)
	}
}

func TestBuildConfigForMainRoleUsesCheckIPListAndURLProbe(t *testing.T) {
	t.Parallel()

	config := floatgateway.BuildConfig(floatgateway.Input{
		Enabled:         true,
		Role:            "main",
		CheckIP:         "192.168.100.2",
		SetIP:           "192.168.100.3",
		CheckURL:        "https://example.com",
		CheckURLTimeout: 8,
	})

	want := floatgateway.Config{
		Enabled:            true,
		Role:               "main",
		SetIP:              "192.168.100.3",
		UseSetIP:           true,
		CheckURL:           "https://example.com",
		CheckURLTimeout:    8,
		UseURLProbeSetting: true,
		CheckIPs:           []string{"192.168.100.2"},
	}
	if !reflect.DeepEqual(config, want) {
		t.Fatalf("config = %+v, want %+v", config, want)
	}
}

func TestBuildConfigForUnsupportedRoleSkipsFloatSection(t *testing.T) {
	t.Parallel()

	config := floatgateway.BuildConfig(floatgateway.Input{
		Enabled: true,
		Role:    "unexpected",
		CheckIP: "192.168.100.2",
		SetIP:   "192.168.100.3",
	})

	if !reflect.DeepEqual(config, floatgateway.Config{}) {
		t.Fatalf("config = %+v, want empty config", config)
	}
}

func TestShouldCleanupDhcpWhenEnabledGatewayIsDisabled(t *testing.T) {
	t.Parallel()

	if !floatgateway.ShouldCleanupDhcp(
		floatgateway.StateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		floatgateway.Input{Enabled: false, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
	) {
		t.Fatal("expected cleanup when enabled gateway is disabled")
	}
}

func TestShouldCleanupDhcpWhenSetIPOrCheckIPChanges(t *testing.T) {
	t.Parallel()

	if !floatgateway.ShouldCleanupDhcp(
		floatgateway.StateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		floatgateway.Input{Enabled: true, SetIP: "192.168.100.4", CheckIP: "192.168.100.2"},
	) {
		t.Fatal("expected cleanup when set ip changes")
	}
	if !floatgateway.ShouldCleanupDhcp(
		floatgateway.StateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		floatgateway.Input{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.5"},
	) {
		t.Fatal("expected cleanup when check ip changes")
	}
}

func TestShouldCleanupDhcpSkipsUnchangedEnabledGateway(t *testing.T) {
	t.Parallel()

	if floatgateway.ShouldCleanupDhcp(
		floatgateway.StateSnapshot{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
		floatgateway.Input{Enabled: true, SetIP: "192.168.100.3", CheckIP: "192.168.100.2"},
	) {
		t.Fatal("expected unchanged enabled gateway to skip cleanup")
	}
}

func TestBuildDhcpCleanupPlanDeletesAutoTagsAndBoundHosts(t *testing.T) {
	t.Parallel()

	plan := floatgateway.BuildDhcpCleanupPlan(
		[]floatgateway.DhcpTagSnapshot{
			{SectionName: "t_auto_lan1"},
			{SectionName: "manual"},
			{SectionName: "t_auto_lan2"},
		},
		[]floatgateway.DhcpHostSnapshot{
			{SectionName: "cfg01", Tag: "t_auto_lan1"},
			{SectionName: "cfg02", Tag: "manual"},
			{SectionName: "cfg03", Tag: "t_auto_lan2"},
		},
	)

	want := floatgateway.DhcpCleanupPlan{
		DeleteTagSections:  []string{"t_auto_lan1", "t_auto_lan2"},
		DeleteHostSections: []string{"cfg01", "cfg03"},
	}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("plan = %+v, want %+v", plan, want)
	}
}

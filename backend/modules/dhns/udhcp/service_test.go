package udhcp

import (
	"errors"
	"reflect"
	"testing"
)

func TestPIDFromFileDataParsesTrimmedPID(t *testing.T) {
	t.Parallel()

	if got := PIDFromFileData([]byte("1234\n"), nil); got != 1234 {
		t.Fatalf("PIDFromFileData = %d, want 1234", got)
	}
}

func TestPIDFromFileDataReturnsZeroForMissingOrInvalidData(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		data []byte
		err  error
	}{
		{name: "read error", err: errors.New("missing")},
		{name: "invalid", data: []byte("not-a-pid\n")},
		{name: "empty", data: []byte("\n")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := PIDFromFileData(tc.data, tc.err); got != 0 {
				t.Fatalf("PIDFromFileData = %d, want 0", got)
			}
		})
	}
}

func TestPlanStartSkipsWhenPIDIsRunning(t *testing.T) {
	t.Parallel()

	plan := PlanStart("br-lan", 1234, true)

	if !plan.AlreadyRunning {
		t.Fatalf("expected already-running plan: %#v", plan)
	}
	if plan.RemovePIDFile || len(plan.Commands) != 0 {
		t.Fatalf("expected no side effects when already running: %#v", plan)
	}
}

func TestPlanStartRemovesStalePIDFileAndStartsUDHCPC(t *testing.T) {
	t.Parallel()

	plan := PlanStart("br-lan", 1234, false)

	if plan.AlreadyRunning {
		t.Fatalf("did not expect already-running plan: %#v", plan)
	}
	if !plan.RemovePIDFile {
		t.Fatalf("expected stale pidfile removal: %#v", plan)
	}
	wantCommands := []string{`udhcpc -t 0 -x hostname:dhns -x lease:120 -s "/usr/libexec/quickstart/dhcpvalid.sh" -i br-lan -p /tmp/run/udhns.pid`}
	if !reflect.DeepEqual(plan.Commands, wantCommands) {
		t.Fatalf("Commands = %#v, want %#v", plan.Commands, wantCommands)
	}
}

func TestPlanStartStartsWhenPIDFileIsMissing(t *testing.T) {
	t.Parallel()

	plan := PlanStart("eth0", 0, false)

	if plan.RemovePIDFile || plan.AlreadyRunning {
		t.Fatalf("unexpected pidfile side effect: %#v", plan)
	}
	wantCommands := []string{`udhcpc -t 0 -x hostname:dhns -x lease:120 -s "/usr/libexec/quickstart/dhcpvalid.sh" -i eth0 -p /tmp/run/udhns.pid`}
	if !reflect.DeepEqual(plan.Commands, wantCommands) {
		t.Fatalf("Commands = %#v, want %#v", plan.Commands, wantCommands)
	}
}

func TestPlanStopKillsRunningPIDAndRemovesPIDFile(t *testing.T) {
	t.Parallel()

	plan := PlanStop(1234, true)

	if !plan.RemovePIDFile {
		t.Fatalf("expected pidfile removal: %#v", plan)
	}
	if !reflect.DeepEqual(plan.Commands, []string{"kill 1234"}) {
		t.Fatalf("Commands = %#v, want kill command", plan.Commands)
	}
}

func TestPlanStopOnlyRemovesStalePositivePID(t *testing.T) {
	t.Parallel()

	plan := PlanStop(1234, false)
	if !plan.RemovePIDFile || len(plan.Commands) != 0 {
		t.Fatalf("expected remove-only stale pid plan: %#v", plan)
	}

	plan = PlanStop(0, false)
	if plan.RemovePIDFile || len(plan.Commands) != 0 {
		t.Fatalf("expected no-op plan for missing pid: %#v", plan)
	}
}

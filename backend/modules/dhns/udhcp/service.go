package udhcp

import (
	"fmt"
	"strconv"
	"strings"
)

const PIDFile = "/tmp/run/udhns.pid"

type StartPlan struct {
	AlreadyRunning bool
	RemovePIDFile  bool
	Commands       []string
}

type StopPlan struct {
	RemovePIDFile bool
	Commands      []string
}

func PIDFromFileData(data []byte, readErr error) int {
	if readErr != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.Trim(string(data), "\n"))
	return pid
}

func PlanStart(lanDev string, pid int, pidExists bool) StartPlan {
	if pid > 0 && pidExists {
		return StartPlan{AlreadyRunning: true}
	}
	plan := StartPlan{
		Commands: []string{fmt.Sprintf(
			`udhcpc -t 0 -x hostname:dhns -x lease:120 -s "/usr/libexec/quickstart/dhcpvalid.sh" -i %s -p %s`,
			lanDev,
			PIDFile,
		)},
	}
	if pid > 0 {
		plan.RemovePIDFile = true
	}
	return plan
}

func PlanStop(pid int, pidExists bool) StopPlan {
	if pid <= 0 {
		return StopPlan{}
	}
	plan := StopPlan{RemovePIDFile: true}
	if pidExists {
		plan.Commands = []string{fmt.Sprintf("kill %d", pid)}
	}
	return plan
}

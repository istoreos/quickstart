package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/linkease/quick-start/istore-backend/dhns"
	dhnsruntime "github.com/linkease/quick-start/istore-backend/modules/dhns/runtime"
	systemthermal "github.com/linkease/quick-start/istore-backend/modules/system/thermal"
	"github.com/linkease/quick-start/istore-backend/utils"
	"golang.org/x/sys/unix"
)

type ServiceBackend struct {
	mu sync.Mutex

	st             *WanStats
	lstats         *LanStats
	httpClient     *http.Client
	netChecker     *NetworkOnlineChecker
	foreignChecker *ForeignChecker
	thermalZone    systemthermal.Getter
	platform       string

	dhnsServer  *dhns.DhnsServer
	dhnsState   *dhnsruntime.State
	disableDHNS bool
}

func NewServiceBackend() *ServiceBackend {
	var thermalZone systemthermal.Getter
	if unix.Access("/sbin/cpuinfo", unix.X_OK) == nil {
		l.Debugln("autocoreTemperature")
		thermalZone = systemthermal.AutocoreTemperature{}
	} else {
		var arch string
		var tempZone string
		if runtime.GOARCH == "amd64" {
			arch = "x86_64"
		} else if runtime.GOARCH == "arm64" {
			arch = "aarch64"
		} else {
			arch0, err := utils.BatchOutputCmd(context.Background(), "uname -m", 0)
			if err != nil {
				arch = "unknown"
			} else {
				arch = strings.Trim(string(arch0), "\n")
			}
		}
		l.Debugln("arch", arch)
		switch arch {
		case "aarch64":
			tempZone = "thermal_zone0"
			_, err := systemthermal.ReadZoneTemperature(tempZone)
			if err != nil {
				tempZone = systemthermal.DetectZone()
			}
		case "x86_64":
			tempZone = systemthermal.DetectZone()
			// /sys/class/hwmon/hwmon1/temp2_label
			// /sys/class/hwmon/hwmon1/temp2_input
		}
		// initial a zone
		if tempZone == "" {
			// try using hwmon
			for hwmon := 0; hwmon < 5; hwmon++ {
				nameFile := fmt.Sprintf("/sys/class/hwmon/hwmon%d/name", hwmon)
				ret, err := ioutil.ReadFile(nameFile)
				if err != nil {
					break
				}
				retStr := string(ret)
				retStr = strings.Trim(retStr, "\n")
				if strings.HasPrefix(retStr, "k8temp") || strings.HasPrefix(retStr, "k10temp") ||
					strings.HasPrefix(retStr, "coretemp") || strings.HasPrefix(retStr, "intel5500") {
					var idx int
					for idx = 1; idx < 6; idx++ {
						if _, err := systemthermal.ReadHwmonTemperature(hwmon, idx); err == nil {
							l.Debugln("hwmonTemperature", retStr, hwmon, idx)
							thermalZone = systemthermal.NewHwmonTemperature(hwmon, idx)
							break
						}
					}
					if idx < 6 {
						break
					}
				} else if strings.HasPrefix(retStr, "it86") || strings.HasPrefix(retStr, "it87") ||
					strings.HasPrefix(retStr, "via_cputemp") {
					if _, err := systemthermal.ReadHwmonTemperature(hwmon, 1); err == nil {
						l.Debugln("hwmonTemperature candidate", retStr, hwmon, 1)
						thermalZone = systemthermal.NewHwmonTemperature(hwmon, 1)
					}
				}
			}
			if thermalZone == nil {
				if _, err := systemthermal.ReadHwmonTemperature(1, 1); err == nil {
					l.Debugln("hwmonTemperature default", 1, 1)
					thermalZone = systemthermal.NewHwmonTemperature(1, 1)
				}
			}

		} else {
			l.Debugln("thermalZoneTemperature", tempZone)
			thermalZone = systemthermal.NewZoneTemperature(tempZone)
		}
		if thermalZone == nil {
			// MUST not be nil
			l.Debugln("thermalZoneTemperature default", "thermal_zone0")
			thermalZone = systemthermal.NewZoneTemperature("thermal_zone0")
		}
	}
	backend := &ServiceBackend{
		st:     NewWanStats(),
		lstats: NewLanStats(),
		httpClient: &http.Client{
			Timeout: time.Second * 20,
		},
		netChecker:     NewNetworkOnlineChecker(),
		foreignChecker: NewForeignChecker(),
		platform:       runtime.GOARCH,
		thermalZone:    thermalZone,
		dhnsState:      dhnsruntime.NewState(),
	}
	backend.setupDhns()
	return backend
}

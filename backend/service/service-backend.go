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

	"github.com/istoreos/quickstart/backend/dhns"
	dhnsruntime "github.com/istoreos/quickstart/backend/modules/dhns/runtime"
	systemthermal "github.com/istoreos/quickstart/backend/modules/system/thermal"
	"github.com/istoreos/quickstart/backend/utils"
	"golang.org/x/sys/unix"
)

type ServiceBackend struct {
	mu sync.Mutex

	st                   *WanStats
	lstats               *LanStats
	httpClient           *http.Client
	netChecker           *NetworkOnlineChecker
	foreignChecker       *ForeignChecker
	thermalZone          systemthermal.Getter
	platform             string
	deviceInventory      *DeviceInventoryModule
	deviceTraffic        *DeviceTrafficModule
	devicePolicy         *DevicePolicyModule
	deviceClassification *DeviceClassificationModule
	deviceProfile        *DeviceProfileModule
	gatewayPolicy        *GatewayPolicyModule
	deviceNetworkPolicy  *DeviceNetworkPolicyModule
	floatingGateway      *FloatingGatewayModule
	networkRules         *NetworkRulesModule
	lanDeviceMigration   *LanDeviceMigrationModule
	capabilityActions    *CapabilityActionModule
	routerContext        *RouterContextModule
	lanDHCPSettings      *LanDHCPSettingsModule
	deviceGroups         *DeviceGroupModule
	trafficInsights      *TrafficInsightsModule
	networkAudit         *NetworkAuditModule
	advancedNetwork      *AdvancedNetworkModule
	taskTransactions     *TaskTransactionJournal

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
	inventory := NewDeviceInventoryModule()
	transactions := NewDefaultTaskTransactionJournal()
	lanStats := NewLanStats()
	backend := &ServiceBackend{
		st:     NewWanStats(),
		lstats: lanStats,
		httpClient: &http.Client{
			Timeout: time.Second * 20,
		},
		netChecker:       NewNetworkOnlineChecker(),
		foreignChecker:   NewForeignChecker(),
		platform:         runtime.GOARCH,
		thermalZone:      thermalZone,
		deviceInventory:  inventory,
		taskTransactions: transactions,
		dhnsState:        dhnsruntime.NewState(),
	}
	backend.deviceTraffic = NewDeviceTrafficModule(inventory, lanStats)
	backend.devicePolicy = NewDevicePolicyModule(inventory)
	backend.devicePolicy.transactions = transactions
	backend.deviceClassification = NewDeviceClassificationModule(inventory)
	backend.deviceProfile = NewDeviceProfileModule(inventory)
	backend.deviceProfile.transactions = transactions
	backend.gatewayPolicy = NewDefaultGatewayPolicyModule(inventory)
	backend.gatewayPolicy.transactions = transactions
	backend.deviceNetworkPolicy = NewDefaultDeviceNetworkPolicyModule(inventory, backend.devicePolicy, backend.gatewayPolicy)
	backend.deviceNetworkPolicy.transactions = transactions
	backend.floatingGateway = NewDefaultFloatingGatewayModule(backend.gatewayPolicy)
	backend.networkRules = NewDefaultNetworkRulesModule(inventory, backend.devicePolicy, backend.gatewayPolicy)
	backend.lanDeviceMigration = NewDefaultLanDeviceMigrationModule(backend.networkRules)
	backend.capabilityActions = NewDefaultCapabilityActionModule()
	backend.routerContext = NewDefaultRouterContextModule()
	backend.lanDHCPSettings = NewDefaultLanDHCPSettingsModule(backend.gatewayPolicy, backend.routerContext)
	backend.lanDHCPSettings.transactions = transactions
	backend.trafficInsights = NewDefaultTrafficInsightsModule(backend.devicePolicy)
	if backend.gatewayPolicy.groups != nil {
		backend.deviceGroups = newDefaultDeviceGroupModuleWithStore(backend.gatewayPolicy.groups, backend.devicePolicy, backend.deviceNetworkPolicy, backend.trafficInsights)
	} else {
		backend.deviceGroups = NewDefaultDeviceGroupModule(backend.devicePolicy, backend.deviceNetworkPolicy, backend.trafficInsights)
	}
	backend.networkAudit = NewDefaultNetworkAuditModule()
	backend.trafficInsights.audit = backend.networkAudit
	backend.trafficInsights.AttachCollector(backend.deviceTraffic.Snapshot)
	backend.advancedNetwork = NewAdvancedNetworkModule(inventory, backend.deviceGroups, backend.trafficInsights, backend.networkAudit)
	backend.setupDhns()
	return backend
}

package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/lib/scope_error"
	"github.com/linkease/quick-start/istore-backend/models"
	wirelessguest "github.com/linkease/quick-start/istore-backend/modules/wireless/guestnetwork"
	wirelessifacecontrol "github.com/linkease/quick-start/istore-backend/modules/wireless/ifacecontrol"
	wirelesssetup "github.com/linkease/quick-start/istore-backend/modules/wireless/setup"
	"github.com/linkease/quick-start/istore-backend/utils"
)

const (
	BandType2G = "2g"
	BandType5G = "5g"
)

const (
	SsidPrefix   = "iStoreOS"
	Ssid2G       = SsidPrefix + "-2G"
	Ssid5G       = SsidPrefix + "-5G"
	Ssid5GGuest  = SsidPrefix + "-5G-Guest"
	Ssid2GGuest  = SsidPrefix + "-Guest"
	SsidGuestKey = "goodlife"
)

type BaseWifiDevice struct {
	ReadIndex int    `json:"read_index"`
	Type      string `json:"type"` // mtk,qcawificfg80211
	Band      string `json:"band"` // 2g,5g
	Channel   string `json:"channel"`
	TxPower   int    `json:"txpower"`
	Htmode    string `json:"htmode"`
}

type BaseWifiIface struct {
	Device     string `json:"device"`
	Mode       string `json:"mode"`
	Network    string `json:"network"`
	Ifname     string `json:"ifname"`
	Ssid       string `json:"ssid"`
	Encryption string `json:"encryption"`
	Key        string `json:"key"`
	Disabled   bool   `json:"disabled"`
}

type WifiInfoParam struct {
	Device    string
	IfaceName string
	Band      string
}

type BaseWifi interface {
	GetDriveType() string
	ReloadCommand() string
	ParamFor2G() *WifiInfoParam
	ParamFor5G() *WifiInfoParam
	ParamForGuest2G() *WifiInfoParam
	ParamForGuest5G() *WifiInfoParam
	ListIfaces(ctx context.Context) (*models.WirelessListIfaceResponse, error)
	WirelessInfo(param *WifiInfoParam) (*models.WirelessIfaceInfo, error)
	EnableGuest(ctx context.Context, req *models.WirelessEnableIfaceRequest) error
	WirelessEditIface(ctx context.Context, req *models.WirelessIfaceInfo) error
	EditOneIface(ctx context.Context, req *models.WirelessIfaceInfo) error
	SetPower(ctx context.Context, req *models.WirelessSetDevicePowerRequest) error
	AssocMacList(ctx context.Context) (map[string]struct{}, error)
}

func disableOfWireless(getDisabled string) bool {
	if _, ok := uci.Get("wireless", getDisabled, ""); ok {
		if val, ok := uci.GetLast("wireless", getDisabled, "disabled"); ok {
			valInt, _ := strconv.Atoi(val)
			if valInt > 0 {
				return true
			}
		}
	}
	return false
}

// wireless/iface-list
func WirelessListIfaces(ctx context.Context) (*models.WirelessListIfaceResponse, error) {
	// uci get wireless.mt798111.type
	uci.LoadConfig("wireless", true)
	//val, ok := uci.GetLast("wireless", "mt798111", "type")
	//l.Debugln("val=", val, "ok=", ok)
	wifi := wifiSelect()
	if wifi == nil {
		return nil, scope_error.NewScopeErr(errors.New("wireless not compatible"), -10001, "wireless")
	}

	return wifi.ListIfaces(ctx)

}

func WirelessEnableIface(ctx context.Context, r *http.Request) error {
	req := &models.WirelessEnableIfaceRequest{}
	err := getBody(req, r)
	if err != nil {
		return errors.New("Invalid request")
	}
	return WirelessEnableIfaceWithRequest(ctx, *req)
}

func WirelessEnableIfaceWithRequest(ctx context.Context, req models.WirelessEnableIfaceRequest) error {
	uci.LoadConfig("wireless", true)
	wifi := wifiSelect()
	if wifi == nil {
		return scope_error.NewScopeErr(errors.New("wireless not compatible"), -10001, "wireless")
	}

	var device string
	if val, ok := uci.GetLast("wireless", req.IfaceName, "device"); ok {
		device = val
	} else {
		return errors.New("device not found")
	}

	if strings.HasPrefix(req.IfaceName, "guest") {
		var oldDisabled bool
		if _, ok := uci.Get("wireless", req.IfaceName, ""); ok {
			if val, ok := uci.GetLast("wireless", req.IfaceName, "disabled"); ok {
				valInt, _ := strconv.Atoi(val)
				if valInt > 0 {
					oldDisabled = true
				}
			}
		} else {
			oldDisabled = true
		}
		if oldDisabled != req.Enable {
			// Already equal !oldDisabled == req.Enable
			utils.BatchRun(ctx, []string{wifi.ReloadCommand()}, 0)
			return nil
		}

		cmdList := make([]string, 0, 16)
		if req.Enable {

			// Enable guest network.interface
			outBytes, err := utils.BatchOutput(ctx, wirelessguest.NetworkProbeCommands(), 0)
			cmdList = append(cmdList, wirelessguest.PlanNetworkCommands(outBytes, err)...)

			// Enable guest DHCP
			cmdList = enableGuestDhcpCommand(ctx, cmdList)
			// Enable guest firewall
			cmdList = enableGuestFirewallCommand(ctx, cmdList)

			if len(cmdList) > 0 {
				cmdList = append(cmdList, wirelessguest.SuccessMarkerCommand())
				outBytes, err = utils.BatchOutput(ctx, cmdList, 0)
				if err != nil {
					return err
				}
				if !wirelessguest.HasSuccessMarker(outBytes) {
					return errors.New("unexpected error")
				}
				// Reset cmdList
				cmdList = cmdList[:0]
			}

			reqPtr := &req
			if _, ok := uci.Get("wireless", req.IfaceName, ""); !ok {
				return wifi.EnableGuest(ctx, reqPtr)
			} else {
				cmdList = []string{
					fmt.Sprintf(`uci set wireless.%s.disabled=0`, req.IfaceName),
					`uci commit wireless`,
					wifi.ReloadCommand(),
				}
				return utils.BatchRun(ctx, cmdList, 0)
			}
		}

		// Disable guest
		cmdList = []string{
			fmt.Sprintf(`uci set wireless.%s.disabled=1`, req.IfaceName),
			`uci commit wireless`,
			wifi.ReloadCommand(),
		}
		utils.BatchRun(ctx, cmdList, 0)
		return nil
	}

	// Not guest here
	return enableIface(ctx, &req, device)
}

func enableIface(ctx context.Context, req *models.WirelessEnableIfaceRequest, device string) error {
	deviceDisabled := disableOfWireless(device)
	ifaceDisabled := false
	if !deviceDisabled {
		ifaceDisabled = disableOfWireless(req.IfaceName)
	}
	plan := wirelessifacecontrol.Plan(wirelessifacecontrol.PlanInput{
		Enable:         req.Enable,
		DeviceName:     device,
		IfaceName:      req.IfaceName,
		DeviceDisabled: deviceDisabled,
		IfaceDisabled:  ifaceDisabled,
	})

	l.Debugln("cmdList=\n", strings.Join(plan.Commands, "\n"))
	return utils.BatchRun(ctx, plan.Commands, 0)
}

func enableGuestDhcpCommand(ctx context.Context, cmdList []string) []string {
	// Enable guest DHCP
	outBytes, err := utils.BatchOutput(ctx, wirelessguest.DHCPProbeCommands(), 0)
	return append(cmdList, wirelessguest.PlanDHCPCommands(outBytes, err)...)
}

func enableGuestFirewallCommand(ctx context.Context, cmdList []string) []string {
	outBytes, err := utils.BatchOutput(ctx, wirelessguest.FirewallProbeCommands(), 0)
	return append(cmdList, wirelessguest.PlanFirewallCommands(outBytes, err)...)
}

func WirelessSetDevicePower(ctx context.Context, r *http.Request) error {
	req := &models.WirelessSetDevicePowerRequest{}
	err := getBody(req, r)
	if err != nil {
		return errors.New("Invalid request")
	}
	return WirelessSetDevicePowerWithRequest(ctx, *req)
}

func WirelessSetDevicePowerWithRequest(ctx context.Context, req models.WirelessSetDevicePowerRequest) error {
	if req.Txpower <= 0 || req.Txpower > 100 {
		req.Txpower = 100
	}

	uci.LoadConfig("wireless", true)
	wifi := wifiSelect()
	if wifi == nil {
		return scope_error.NewScopeErr(errors.New("wireless not compatible"), -10001, "wireless")
	}

	return wifi.SetPower(ctx, &req)
}

func WirelessEditIface(ctx context.Context, r *http.Request) error {
	req := &models.WirelessIfaceInfo{}
	err := getBody(req, r)
	if err != nil {
		return errors.New("Invalid request")
	}
	return WirelessEditIfaceWithRequest(ctx, *req)
}

func WirelessEditIfaceWithRequest(ctx context.Context, req models.WirelessIfaceInfo) error {
	uci.LoadConfig("wireless", true)
	wifi := wifiSelect()
	if wifi == nil {
		return scope_error.NewScopeErr(errors.New("wireless not compatible"), -10001, "wireless")
	}
	return wifi.WirelessEditIface(ctx, &req)
}

func WirelessQuickSetupIface(ctx context.Context, r *http.Request) error {
	req := &models.WirelessQuickSetupRequest{}
	err := getBody(req, r)
	if err != nil {
		return errors.New("Invalid request")
	}
	return WirelessQuickSetupIfaceWithRequest(ctx, *req)
}

func WirelessQuickSetupIfaceWithRequest(ctx context.Context, req models.WirelessQuickSetupRequest) error {
	reqPtr := &req
	if err := wirelesssetup.ValidateQuickSetupRequest(reqPtr); err != nil {
		return err
	}

	uci.LoadConfig("wireless", true)
	wifi := wifiSelect()
	if wifi == nil {
		return scope_error.NewScopeErr(errors.New("wireless not compatible"), -10001, "wireless")
	}

	wifi2G, err1 := wifi.WirelessInfo(wifi.ParamFor2G())
	wifi5G, err2 := wifi.WirelessInfo(wifi.ParamFor5G())
	if err1 == nil &&
		err2 == nil &&
		wifi2G.Key == req.Wifi2g.Key &&
		wifi2G.Ssid == req.Wifi2g.Ssid &&
		wifi5G.Key == req.Wifi2g.Key &&
		wifi5G.Ssid == req.Wifi5g.Ssid {
		return nil
	}

	wirelesssetup.NormalizeQuickSetupRequest(reqPtr)

	err := wifi.EditOneIface(ctx, req.Wifi2g)
	if err != nil {
		return err
	}
	return wifi.EditOneIface(ctx, req.Wifi5g)
}

type WifiSelectFunc func(old BaseWifi) BaseWifi

type wifiSelector struct {
	mu           sync.Mutex
	funcs        []WifiSelectFunc
	selected     int
	selectedWifi BaseWifi
}

var wifiSel *wifiSelector = &wifiSelector{
	selected: -1,
}

func (sel *wifiSelector) register(f WifiSelectFunc) {
	sel.mu.Lock()
	defer sel.mu.Unlock()
	sel.funcs = append(sel.funcs, f)
}

func (sel *wifiSelector) selectAll() (BaseWifi, int) {
	for i, fn := range sel.funcs {
		wifi := fn(nil)
		if wifi != nil {
			return wifi, i
		}
	}
	return nil, -1
}

func (sel *wifiSelector) selectWifi() BaseWifi {
	var wifi BaseWifi
	var idx int
	sel.mu.Lock()
	idx = sel.selected
	wifi = sel.selectedWifi
	sel.mu.Unlock()
	if idx >= 0 {
		wifi = sel.funcs[idx](wifi)
		if wifi != nil {
			return wifi
		}
	}
	wifi, idx = sel.selectAll()
	if wifi != nil {
		sel.mu.Lock()
		sel.selected = idx
		sel.selectedWifi = wifi
		sel.mu.Unlock()
	}
	return wifi
}

func wifiSelect() BaseWifi {
	return wifiSel.selectWifi()
}

func checkHasWireless() bool {
	uci.LoadConfig("wireless", true)
	wifi := wifiSelect()
	return wifi != nil
}

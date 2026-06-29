package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	wirelessguestiface "github.com/linkease/quick-start/istore-backend/modules/wireless/guestiface"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type QCAWifiDevice struct {
	BaseWifiDevice
	Macaddr     string `json:"macaddr"`
	Hwmode      string `json:"hwmode"`
	Country     string `json:"country"`
	RandomBssid bool   `json:"random_bssid"`
	LegacyRates bool   `json:"legacy_rates"`
}

type QCAWifiIface struct {
	BaseWifiIface
	FactoryMacaddr string `json:"factory_macaddr"`
	Wds            bool   `json:"wds"`
	Isolate        bool   `json:"isolate"`
	Hidden         bool   `json:"hidden"`
	Ieee80211k     bool   `json:"ieee80211k"`
	BssTransition  bool   `json:"bss_transition"`
	Sae            bool   `json:"sae"`
}

// QCAWifi represents a QCA-based Wi-Fi device.
// TODO support multple ssids
type QcaWifi struct {
	param2G         WifiInfoParam
	param5G         WifiInfoParam
	paramGuest2G    WifiInfoParam
	paramGuest5G    WifiInfoParam
	ifaceIndex2G    int
	ifaceIndex5G    int
	encryptSelects  []string
	hwmode2GSelects []string
	hwmode5GSelects []string
}

func NewQcaWifi() *QcaWifi {
	return &QcaWifi{
		param2G: WifiInfoParam{
			Device:    "wifi0",
			IfaceName: "wifi2g",
			Band:      "2g",
		},
		param5G: WifiInfoParam{
			Device:    "wifi1",
			IfaceName: "wifi5g",
			Band:      "5g",
		},
		paramGuest2G: WifiInfoParam{
			Device:    "wifi0",
			IfaceName: "guest2g",
			Band:      "2g",
		},
		paramGuest5G: WifiInfoParam{
			Device:    "wifi1",
			IfaceName: "guest5g",
			Band:      "5g",
		},
		ifaceIndex2G:    0,
		ifaceIndex5G:    1,
		encryptSelects:  []string{"OPEN", "WPA2-PSK", "WPA/WPA2-PSK", "WPA3-SAE", "WPA2-PSK/WPA3-SEA"},
		hwmode2GSelects: []string{"11ax/be", "11b/g/n/ax/be"},
		hwmode5GSelects: []string{"11ax/be", "11ac/ax/be", "11a/n/ac/ax/be"},
	}
}

func (wifi *QcaWifi) GetDriveType() string {
	return "qcawificfg80211"
}

func (wifi *QcaWifi) ReloadCommand() string {
	return "wifi"
}

func (wifi *QcaWifi) ParamFor2G() *WifiInfoParam {
	return &wifi.param2G
}

func (wifi *QcaWifi) ParamFor5G() *WifiInfoParam {
	return &wifi.param5G
}

func (wifi *QcaWifi) ParamForGuest2G() *WifiInfoParam {
	return &wifi.paramGuest2G
}

func (wifi *QcaWifi) ParamForGuest5G() *WifiInfoParam {
	return &wifi.paramGuest5G
}

func (wifi *QcaWifi) encryptValue2Title(encVal string, sae bool) string {
	vals := []string{"none", "psk2+ccmp", "psk-mixed+ccmp", "ccmp"}
	//sels := []string{"OPEN", "WPA2-PSK", "WPA/WPA2-PSK", "WPA3-SAE", "WPA2-PSK/WPA3-SEA"}
	sels := wifi.encryptSelects

	switch encVal {
	case vals[0]:
		return sels[0]
	case vals[1]:
		if sae {
			return sels[4]
		} else {
			return sels[1]
		}
	case vals[2]:
		return sels[2]
	case vals[3]:
		return sels[3]
	default:
		return sels[0]
	}
}

func (wifi *QcaWifi) checkEncryptTitle(encTitle string) bool {
	for _, title := range wifi.encryptSelects {
		if title == encTitle {
			return true
		}
	}
	return false
}

func (wifi *QcaWifi) encryptTitle2Value(encTitle string) (string, bool) {
	vals := []string{"none", "psk2+ccmp", "psk-mixed+ccmp", "ccmp"}
	sels := wifi.encryptSelects
	switch encTitle {
	case sels[1]:
		return vals[1], false
	case sels[2]:
		return vals[2], false
	case sels[3]:
		return vals[3], true
	case sels[4]:
		return vals[1], true
	default:
		return vals[0], false
	}
}

func (wifi *QcaWifi) isHwmodeValid(hwmode string, band2G bool) bool {
	var sels []string
	if band2G {
		sels = wifi.hwmode2GSelects
	} else {
		sels = wifi.hwmode5GSelects
	}
	for _, mode := range sels {
		if mode == hwmode {
			return true
		}
	}
	return false
}

func (wifi *QcaWifi) hw2G2Title(hwmode, requireMode string) string {
	if requireMode == "ax" {
		return wifi.hwmode2GSelects[0]
	}
	return wifi.hwmode2GSelects[1]
}

func (wifi *QcaWifi) title2Hw2G(title string) (string, string) {
	switch title {
	case wifi.hwmode2GSelects[0]:
		return "11beg", "ax"
	default:
		return "11beg", ""
	}
}

func (wifi *QcaWifi) hw5G2Title(hwmode, requireMode string) string {
	if requireMode == "ax" {
		return wifi.hwmode5GSelects[0]
	} else if requireMode == "ac" {
		return wifi.hwmode5GSelects[1]
	}
	return wifi.hwmode5GSelects[2]
}

func (wifi *QcaWifi) title2Hw5G(title string) (string, string) {
	switch title {
	case wifi.hwmode5GSelects[0]:
		return "11bea", "ax"
	case wifi.hwmode5GSelects[1]:
		return "11bea", "ac"
	default:
		return "11bea", ""
	}
}

func (wifi *QcaWifi) ListIfaces(ctx context.Context) (*models.WirelessListIfaceResponse, error) {
	secs, has := uci.GetSections("wireless", "wifi-iface")
	l.Debugln("wireless.wifi-face=", has)
	if has {
		for _, sec := range secs {
			l.Debugln("sec=", sec)
		}
	}

	wifi2G, err := wifi.WirelessInfo(wifi.ParamFor2G())
	if err != nil {
		return nil, err
	}
	wifi5G, err := wifi.WirelessInfo(wifi.ParamFor5G())
	if err != nil {
		return nil, err
	}
	guest2G, err := wifi.WirelessInfo(wifi.ParamForGuest2G())
	if err != nil {
		return nil, err
	}
	guest5G, err := wifi.WirelessInfo(wifi.ParamForGuest5G())
	if err != nil {
		return nil, err
	}
	return &models.WirelessListIfaceResponse{
		Result: &models.WirelessListIfaceResponseResult{
			Ifaces: []*models.WirelessIfaceInfo{
				wifi5G, wifi2G, guest5G, guest2G,
			},
		},
	}, nil
}

func (wifi *QcaWifi) WirelessInfo(ps *WifiInfoParam) (*models.WirelessIfaceInfo, error) {
	device, ifname, bandType := ps.Device, ps.IfaceName, ps.Band
	info := &models.WirelessIfaceInfo{
		Device:     device,
		IfaceName:  ifname,
		Band:       bandType,
		Network:    "lan",
		Channel:    int64(0),
		Encryption: "none",
		Hidden:     false,
		Disabled:   true,
		IsGuest:    false,
		//IfaceIndex: 2,
	}
	if bandType == "5g" {
		info.Htmode = "80"
		info.Hwmode = wifi.hwmode5GSelects[len(wifi.hwmode5GSelects)-1]
		if strings.HasPrefix(ifname, "guest") {
			info.IsGuest = true
			info.Ssid = Ssid5GGuest
			info.Key = SsidGuestKey
		} else {
			info.Ssid = Ssid5G
		}
	} else {
		info.Htmode = "auto"
		info.Hwmode = wifi.hwmode2GSelects[len(wifi.hwmode2GSelects)-1]
		if strings.HasPrefix(ifname, "guest") {
			info.IsGuest = true
			info.Ssid = Ssid5GGuest
			info.Key = SsidGuestKey
		} else {
			info.Ssid = Ssid2G
		}
	}

	if val, ok := uci.GetLast("wireless", ifname, "ssid"); ok {
		info.Ssid = val
	}
	if val, ok := uci.GetLast("wireless", device, "channel"); ok {
		valInt, _ := strconv.Atoi(val)
		info.Channel = int64(valInt)
	}
	var sae bool
	if val, ok := uci.GetLast("wireless", ifname, "sae"); ok {
		valInt, _ := strconv.Atoi(val)
		if valInt > 0 {
			sae = true
		}
	}
	if val, ok := uci.GetLast("wireless", ifname, "encryption"); ok {
		info.Encryption = wifi.encryptValue2Title(val, sae)
	}
	if val, ok := uci.GetLast("wireless", ifname, "key"); ok {
		info.Key = val
	}
	if val, ok := uci.GetLast("wireless", ifname, "ifname"); ok {
		info.Ifname = val
	}
	if bandType == "2g" {
		info.HwmodeSelects = wifi.hwmode2GSelects
	} else {
		info.HwmodeSelects = wifi.hwmode5GSelects
	}
	info.EncryptSelects = wifi.encryptSelects
	if val, ok := uci.GetLast("wireless", device, "htmode"); ok {
		if val == "HT40" && bandType == "2g" {
			if noscan, ok := uci.GetLast("wireless", device, "noscan"); ok && noscan == "1" {
				info.Htmode = "HT40"
			} else {
				info.Htmode = "auto"
			}
		} else if strings.HasPrefix(val, "HE") {
			info.Htmode = val[2:]
		} else if strings.HasPrefix(val, "HT") {
			info.Htmode = val[2:]
		} else if strings.HasPrefix(val, "VHT") {
			info.Htmode = val[3:]
		} else if strings.HasPrefix(val, "EHT") {
			info.Htmode = val[3:]
		}
	}
	var requireMode string
	if val, ok := uci.GetLast("wireless", device, "require_mode"); ok {
		requireMode = val
	}
	if val, ok := uci.GetLast("wireless", device, "hwmode"); ok {
		if bandType == "2g" {
			info.Hwmode = wifi.hw2G2Title(val, requireMode)
		} else {
			info.Hwmode = wifi.hw5G2Title(val, requireMode)
		}
	}
	if val, ok := uci.GetLast("wireless", device, "txpower"); ok {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 || valInt > 100 {
			valInt = 100
		}
		info.Txpower = int64(wifi.uciPowerVal2WebPowerVal(valInt))
	}

	info.Disabled = disableOfWireless(device)
	if !info.Disabled {
		info.Disabled = disableOfWireless(ifname)
	}

	if val, ok := uci.GetLast("wireless", ifname, "hidden"); ok {
		valInt, _ := strconv.Atoi(val)
		if valInt > 0 {
			info.Hidden = true
		}
	}
	if val, ok := uci.GetLast("wireless", ifname, "network"); ok {
		info.Network = val
	}

	return info, nil
}

func (wifi *QcaWifi) EnableGuest(ctx context.Context, req *models.WirelessEnableIfaceRequest) error {
	ssid := Ssid2GGuest
	ifaceIndex := wifi.ifaceIndex2G
	wirelessIfName := "wlan01"
	encryptVal := "psk2+ccmp"
	if strings.HasSuffix(req.IfaceName, "5g") {
		ifaceIndex = wifi.ifaceIndex5G
		ssid = Ssid5GGuest
		wirelessIfName = "wlan11"
	}
	cmdList := wirelessguestiface.BuildCommands(wirelessguestiface.Profile{
		IfaceName:      req.IfaceName,
		IfaceIndex:     ifaceIndex,
		WirelessIfName: wirelessIfName,
		SSID:           ssid,
		Encryption:     encryptVal,
	})
	return utils.BatchRun(ctx, cmdList, 0)
}

func (wifi *QcaWifi) EditOneIface(ctx context.Context, req *models.WirelessIfaceInfo) error {
	htmodes := make(map[string]string)
	htmodeSelects := []string{"HT40", "HT20", "HT40", "HT80", "HT160"}
	var band2G bool
	if strings.Contains(req.IfaceName, "2g") {
		band2G = true
	}
	for idx, mode := range []string{"auto", "20", "40", "80", "160"} {
		htmodes[mode] = htmodeSelects[idx]
	}
	if _, ok := htmodes[req.Htmode]; !ok {
		return errors.New("invalid htmode")
	}

	if !wifi.isHwmodeValid(req.Hwmode, band2G) {
		return errors.New("invalid hwmode")
	}

	var device string
	if val, ok := uci.GetLast("wireless", req.IfaceName, "device"); ok {
		device = val
	} else {
		return errors.New("device not found")
	}

	var channel string
	if req.Channel == 0 {
		channel = "auto"
	} else {
		channel = strconv.Itoa(int(req.Channel))
	}
	encVal, sae := wifi.encryptTitle2Value(req.Encryption)
	cmdList := []string{
		fmt.Sprintf(`uci set wireless.%s.ssid="%s"`, req.IfaceName, req.Ssid),
		fmt.Sprintf(`uci set wireless.%s.encryption="%s"`,
			req.IfaceName,
			encVal),
		fmt.Sprintf(`uci set wireless.%s.key="%s"`, req.IfaceName, req.Key),
		fmt.Sprintf(`uci set wireless.%s.channel="%s"`, device, channel),
	}
	if sae {
		cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.sae=1`, req.IfaceName))
	} else {
		cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.sae=0`, req.IfaceName))
	}
	//l.Debugln("edit cmdList=", "\n"+strings.Join(cmdList, "\n"))
	if req.Hidden {
		cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.hidden=1`, req.IfaceName))
	} else {
		cmdList = append(cmdList, fmt.Sprintf(`uci del wireless.%s.hidden`, req.IfaceName))
	}

	var hwmode, requireMode string
	if band2G {
		hwmode, requireMode = wifi.title2Hw2G(req.Hwmode)
	} else {
		hwmode, requireMode = wifi.title2Hw5G(req.Hwmode)
	}
	cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.hwmode=%s`,
		device,
		hwmode))
	if requireMode == "" {
		cmdList = append(cmdList, fmt.Sprintf(`uci del wireless.%s.require_mode`, device))
	} else {
		cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.require_mode=%s`,
			device,
			requireMode))
	}

	if band2G {
		if req.Htmode == "HT40" {
			cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.noscan=1`, req.IfaceName))
		} else {
			cmdList = append(cmdList, fmt.Sprintf(`uci del wireless.%s.noscan`, req.IfaceName))
			var htmode string
			if req.Htmode == "auto" {
				htmode = "HT40"
			} else {
				htmode = htmodes[req.Htmode]
			}
			cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.htmode="%s"`, device, htmode))
		}
	} else {
		// 5G
		htmode := htmodes[req.Htmode]
		cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.htmode="%s"`, device, htmode))
	}
	// set network
	var ori_network string
	if val, ok := uci.GetLast("wireless", req.IfaceName, "network"); !ok {
		ori_network = "lan"
	} else {
		ori_network = val
	}
	cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.network="%s"`, req.IfaceName, req.Network))
	cmdList = append(cmdList, "uci commit wireless")
	if req.Network != ori_network {
		cmdList = append(cmdList, `/etc/init.d/network restart`)
	} else {
		cmdList = append(cmdList, wifi.ReloadCommand())
	}
	//l.Debugln("cmdList:\n", strings.Join(cmdList, "\n"))
	return utils.BatchRun(ctx, cmdList, 0)
}

func (wifi *QcaWifi) WirelessEditIface(ctx context.Context, req *models.WirelessIfaceInfo) error {
	if !wifi.checkEncryptTitle(req.Encryption) {
		return errors.New("invalid encryption")
	}
	encVal, sae := wifi.encryptTitle2Value(req.Encryption)
	if strings.HasPrefix(req.IfaceName, "guest") {
		// Guest network
		if _, ok := uci.Get("wireless", req.IfaceName, ""); !ok {
			return errors.New("Enable guest wifi first")
		}
		cmdList := []string{
			fmt.Sprintf(`uci set wireless.%s.ssid="%s"`, req.IfaceName, req.Ssid),
			fmt.Sprintf(`uci set wireless.%s.encryption="%s"`, req.IfaceName, encVal),
			fmt.Sprintf(`uci set wireless.%s.key="%s"`, req.IfaceName, req.Key),
		}
		if sae {
			cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.sae=1`, req.IfaceName))
		} else {
			cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.sae=0`, req.IfaceName))
		}
		if req.Hidden {
			cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.hidden=1`, req.IfaceName))
		} else {
			cmdList = append(cmdList, fmt.Sprintf(`uci del wireless.%s.hidden`, req.IfaceName))
		}

		cmdList = append(cmdList, "uci commit wireless")
		cmdList = append(cmdList, wifi.ReloadCommand())
		return utils.BatchRun(ctx, cmdList, 0)
	}

	return wifi.EditOneIface(ctx, req)
}

func (wifi *QcaWifi) uciPowerVal2WebPowerVal(uciVal int) int {
	v := []int{30, 21, 15, 9}
	v2 := []int{100, 70, 50, 30}
	if uciVal >= v[0] {
		return v2[0]
	} else if uciVal >= v[1] {
		return v2[1]
	} else if uciVal >= v[2] {
		return v2[2]
	} else {
		return v2[3]
	}
}

func (wifi *QcaWifi) webPowerVal2UciPowerVal(webVal int) int {
	v := []int{30, 21, 15, 9}
	v2 := []int{100, 70, 50, 30}
	if webVal >= v2[0] {
		return v[0]
	} else if webVal >= v2[1] {
		return v[1]
	} else if webVal >= v2[2] {
		return v[2]
	} else {
		return v[3]
	}
}

func (wifi *QcaWifi) SetPower(ctx context.Context, req *models.WirelessSetDevicePowerRequest) error {
	cmdList := []string{
		fmt.Sprintf(`uci set wireless.%s.txpower=%d`,
			req.Device,
			wifi.webPowerVal2UciPowerVal(int(req.Txpower))),
		`uci commit wireless`,
		`wifi`,
	}
	return utils.BatchRun(ctx, cmdList, 0)
}

func (wifi *QcaWifi) AssocMacList(ctx context.Context) (map[string]struct{}, error) {
	return nil, errors.New("not implemented")
}

func init() {
	wifiSel.register(func(old BaseWifi) BaseWifi {
		if val, ok := uci.GetLast("wireless", "wifi0", "type"); !ok || val != "qcawificfg80211" {
			return nil
		}
		if old != nil {
			return old
		}
		return NewQcaWifi()
	})
}

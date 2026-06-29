package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/digineo/go-uci"
	"github.com/linkease/quick-start/istore-backend/models"
	wirelessguestiface "github.com/linkease/quick-start/istore-backend/modules/wireless/guestiface"
	"github.com/linkease/quick-start/istore-backend/utils"
)

type ZX3000Device struct {
	BaseWifiDevice
}

type ZX30000Iface struct {
	BaseWifiIface
}

type ZX30000 struct {
	// Default wifi params
	param2G      WifiInfoParam
	param5G      WifiInfoParam
	paramGuest2G WifiInfoParam
	paramGuest5G WifiInfoParam

	ifaceIndex2G    int
	ifaceIndex5G    int
	encryptSelects  []string
	encryptVals     []string
	hwmode2GSelects []string
	hwmode5GSelects []string
}

func NewZX30000Wifi() *ZX30000 {
	return &ZX30000{
		param2G: WifiInfoParam{
			Device:    "radio0",
			IfaceName: "wifi2g",
			Band:      "2g",
		},
		param5G: WifiInfoParam{
			Device:    "radio1",
			IfaceName: "wifi5g",
			Band:      "5g",
		},
		paramGuest2G: WifiInfoParam{
			Device:    "radio0",
			IfaceName: "guest2g",
			Band:      "2g",
		},
		paramGuest5G: WifiInfoParam{
			Device:    "radio1",
			IfaceName: "guest5g",
			Band:      "5g",
		},
		ifaceIndex2G:    0,
		ifaceIndex5G:    1,
		encryptSelects:  []string{"OPEN", "WPA/WPA2-PSK", "WPA2-PSK/WPA3-SEA"},
		encryptVals:     []string{"none", "mixed-psk", "wpa2pskwpa3psk"},
		hwmode2GSelects: []string{"11axg"},
		hwmode5GSelects: []string{"11axa"},
	}
}

func (wifi *ZX30000) GetDriveType() string {
	return "mtk_zx3000"
}

func (wifi *ZX30000) ReloadCommand() string {
	return "wifi reload"
}

func (wifi *ZX30000) ParamFor2G() *WifiInfoParam {
	return &wifi.param2G
}

func (wifi *ZX30000) ParamFor5G() *WifiInfoParam {
	return &wifi.param5G
}

func (wifi *ZX30000) ParamForGuest2G() *WifiInfoParam {
	return &wifi.paramGuest2G
}

func (wifi *ZX30000) ParamForGuest5G() *WifiInfoParam {
	return &wifi.paramGuest5G
}

func (wifi *ZX30000) checkEncryptTitle(encTitle string) bool {
	for _, title := range wifi.encryptSelects {
		if title == encTitle {
			return true
		}
	}
	return false
}

func (wifi *ZX30000) encryptValue2Title(encVal string) string {
	maps := make(map[string]string)
	for idx, title := range wifi.encryptSelects {
		maps[wifi.encryptVals[idx]] = title
	}
	title := maps[encVal]
	if title == "" {
		title = "OPEN"
	}
	return title
}

func (wifi *ZX30000) encryptTitle2Value(encVal string) string {
	maps := make(map[string]string)
	for idx, val := range wifi.encryptVals {
		maps[wifi.encryptSelects[idx]] = val
	}
	val := maps[encVal]
	if val == "" {
		val = "none"
	}
	return val
}

func (wifi *ZX30000) ListIfaces(ctx context.Context) (*models.WirelessListIfaceResponse, error) {
	basicIfaces := make([]*models.WirelessIfaceInfo, 0, 32)
	ifaces2G := make([]*models.WirelessIfaceInfo, 0, 16)
	ifaces5G := make([]*models.WirelessIfaceInfo, 0, 16)
	basicIndex := map[string]*basicIfaceValue{
		wifi.ParamFor5G().IfaceName: &basicIfaceValue{
			priority: 0,
		},
		wifi.ParamFor2G().IfaceName: &basicIfaceValue{
			priority: 1,
		},
		wifi.ParamForGuest5G().IfaceName: &basicIfaceValue{
			priority: 2,
		},
		wifi.ParamForGuest2G().IfaceName: &basicIfaceValue{
			priority: 3,
		},
	}
	var device2G, device5G string
	secs, has := uci.GetSections("wireless", "wifi-device")
	if !has {
		return nil, errors.New("no wifi-device sections found")
	}
	for _, sec := range secs {
		band, ok := uci.GetLast("wireless", sec, "band")
		if ok {
			if band == "2g" {
				device2G = sec
			} else if band == "5g" {
				device5G = sec
			}
		}
	}
	if device2G == "" {
		return nil, errors.New("no 2g wifi-device found")
	}
	if device5G == "" {
		return nil, errors.New("no 5g wifi-device found")
	}

	secs, has = uci.GetSections("wireless", "wifi-iface")
	if !has {
		return nil, errors.New("no wifi-iface sections found")
	}
	var total int
	for _, sec := range secs {
		if strings.Contains(sec, "wisp") {
			continue
		}
		secs[total] = sec
		total++
	}
	secs = secs[:total]
	//l.Debugln("secs=", secs)

	var errRet error
	for _, sec := range secs {
		device, ok := uci.GetLast("wireless", sec, "device")
		if !ok {
			continue
		}
		var bandType string
		if device == device2G {
			bandType = "2g"
		} else if device == device5G {
			bandType = "5g"
		}
		params := &WifiInfoParam{
			Device:    device,
			IfaceName: sec,
			Band:      bandType,
		}
		info, err := wifi.WirelessInfo(params)
		if err != nil {
			if errRet == nil {
				errRet = err
			}
			continue
		}
		if v, ok := basicIndex[info.IfaceName]; ok {
			v.found = true
			basicIfaces = append(basicIfaces, info)
		} else if bandType == "2g" {
			ifaces2G = append(ifaces2G, info)
		} else {
			ifaces5G = append(ifaces5G, info)
		}
	}
	sort.Slice(basicIfaces, func(i, j int) bool {
		a, b := basicIfaces[i], basicIfaces[j]
		p1, p2 := basicIndex[a.IfaceName].priority, basicIndex[b.IfaceName].priority
		return p1 < p2
	})
	sort.Slice(ifaces2G, func(i, j int) bool {
		return ifaces2G[i].IfaceName < ifaces2G[j].IfaceName
	})
	sort.Slice(ifaces5G, func(i, j int) bool {
		return ifaces5G[i].IfaceName < ifaces5G[j].IfaceName
	})

	fromIdx := 0
	if basicIndex[wifi.ParamFor5G().IfaceName].found {
		fromIdx = 1
	}
	for _, iface := range ifaces5G {
		iface.IfaceIndex = int64(fromIdx)
		fromIdx++
		basicIfaces = append(basicIfaces, iface)
	}

	fromIdx = 0
	if basicIndex[wifi.ParamFor2G().IfaceName].found {
		fromIdx = 1
	}
	for _, iface := range ifaces2G {
		iface.IfaceIndex = int64(fromIdx)
		fromIdx++
		basicIfaces = append(basicIfaces, iface)
	}

	return &models.WirelessListIfaceResponse{
		Result: &models.WirelessListIfaceResponseResult{
			Ifaces: basicIfaces,
		},
	}, nil
}

func (wifi *ZX30000) WirelessInfo(ps *WifiInfoParam) (*models.WirelessIfaceInfo, error) {
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
	}
	if bandType == "5g" {
		info.Htmode = "80"
		info.Hwmode = "11a/n/ac/ax"
		if strings.HasPrefix(ifname, "guest") {
			info.IsGuest = true
			info.Ssid = Ssid5GGuest
			info.Key = SsidGuestKey
		} else {
			info.Ssid = Ssid5G
		}
	} else {
		info.Htmode = "auto"
		info.Hwmode = "11b/g/n/ax"
		if strings.HasPrefix(ifname, "guest") {
			info.IsGuest = true
			info.Ssid = Ssid2GGuest
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
	info.EncryptSelects = wifi.encryptSelects
	if val, ok := uci.GetLast("wireless", ifname, "encryption"); ok {
		info.Encryption = wifi.encryptValue2Title(val)
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
	if val, ok := uci.GetLast("wireless", device, "htmode"); ok {
		if val == "HE40" && bandType == "2g" {
			if noscan, ok := uci.GetLast("wireless", device, "noscan"); ok && noscan == "1" {
				info.Htmode = "HE40"
			} else {
				info.Htmode = "auto"
			}
		} else if strings.HasPrefix(val, "HE") {
			info.Htmode = val[2:]
		}
	}
	if val, ok := uci.GetLast("wireless", device, "hwmode"); ok {
		info.Hwmode = val
	}
	if val, ok := uci.GetLast("wireless", device, "power"); ok {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 || valInt > 100 {
			valInt = 100
		}
		info.Txpower = int64(valInt)
	}
	/* if val, ok := uci.GetLast("wireless", device, "disabled"); ok {
		valInt, _ := strconv.Atoi(val)
		if valInt > 0 {
			info.Disabled = true
		}
	} */

	if _, ok := uci.Get("wireless", ifname, ""); ok {
		info.Disabled = false
		if val, ok := uci.GetLast("wireless", ifname, "disabled"); ok {
			valInt, _ := strconv.Atoi(val)
			if valInt > 0 {
				info.Disabled = true
			}
		}
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

func (wifi *ZX30000) EnableGuest(ctx context.Context, req *models.WirelessEnableIfaceRequest) error {
	ssid := Ssid2GGuest
	ifaceIndex := wifi.ifaceIndex2G
	wirelessIfName := "ra1"
	encryptVal := "mixed-psk"
	if strings.HasSuffix(req.IfaceName, "5g") {
		ifaceIndex = wifi.ifaceIndex5G
		ssid = Ssid5GGuest
		wirelessIfName = "rax1"
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

// Not guest here
func (wifi *ZX30000) EditOneIface(ctx context.Context, req *models.WirelessIfaceInfo) error {
	var band2G bool
	if strings.Contains(req.IfaceName, "2g") {
		band2G = true
	}
	htmodes := make(map[string]string)
	htvals := []string{"HE40", "HE20", "HE40", "HE80", "HE160"}
	for idx, mode := range []string{"auto", "20", "40", "80", "160"} {
		htmodes[mode] = htvals[idx]
	}
	if _, ok := htmodes[req.Htmode]; !ok {
		return errors.New("invalid htmode")
	}

	// TODO for hwmode
	hwmodes := make(map[string]string)
	if band2G {
		for _, hwmode := range wifi.hwmode2GSelects {
			hwmodes[hwmode] = "todo"
		}
	} else {
		for _, hwmode := range wifi.hwmode5GSelects {
			hwmodes[hwmode] = "todo"
		}
	}
	_, ok1 := hwmodes[req.Hwmode]
	if !ok1 {
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
	cmdList := []string{
		fmt.Sprintf(`uci set wireless.%s.ssid="%s"`, req.IfaceName, req.Ssid),
		fmt.Sprintf(`uci set wireless.%s.encryption="%s"`,
			req.IfaceName,
			wifi.encryptTitle2Value(req.Encryption)),
		fmt.Sprintf(`uci set wireless.%s.key="%s"`, req.IfaceName, req.Key),
		fmt.Sprintf(`uci set wireless.%s.channel="%s"`, device, channel),
	}
	//l.Debugln("edit cmdList=", "\n"+strings.Join(cmdList, "\n"))
	if req.Hidden {
		cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.hidden=1`, req.IfaceName))
	} else {
		cmdList = append(cmdList, fmt.Sprintf(`uci del wireless.%s.hidden`, req.IfaceName))
	}
	if strings.HasSuffix(req.IfaceName, "2g") {
		if req.Htmode == "HE40" {
			cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.noscan=1`, req.IfaceName))
		} else {
			cmdList = append(cmdList, fmt.Sprintf(`uci del wireless.%s.noscan`, req.IfaceName))
			var htmode string
			if req.Htmode == "auto" {
				htmode = "HE40"
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
		cmdList = append(cmdList, "wifi reload")
	}
	return utils.BatchRun(ctx, cmdList, 0)
}

func (wifi *ZX30000) WirelessEditIface(ctx context.Context, req *models.WirelessIfaceInfo) error {
	if !wifi.checkEncryptTitle(req.Encryption) {
		return errors.New("invalid encryption")
	}
	encVal := wifi.encryptTitle2Value(req.Encryption)
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
		if req.Hidden {
			cmdList = append(cmdList, fmt.Sprintf(`uci set wireless.%s.hidden=1`, req.IfaceName))
		} else {
			cmdList = append(cmdList, fmt.Sprintf(`uci del wireless.%s.hidden`, req.IfaceName))
		}

		cmdList = append(cmdList, "uci commit wireless")
		cmdList = append(cmdList, "wifi reload")
		return utils.BatchRun(ctx, cmdList, 0)
	}

	return wifi.EditOneIface(ctx, req)
}

func (wifi *ZX30000) SetPower(ctx context.Context, req *models.WirelessSetDevicePowerRequest) error {
	cmdList := []string{
		fmt.Sprintf(`uci set wireless.%s.power=%d`, req.Device, req.Txpower),
		`uci commit wireless`,
		`wifi`,
	}
	return utils.BatchRun(ctx, cmdList, 0)
}

type ZX30000Assoc struct {
	Results []ZX30000AssocResult `json:"results"`
}

type ZX30000AssocResult struct {
	Mac string `json:"mac"`
}

func (wifi *ZX30000) AssocMacList(ctx context.Context) (map[string]struct{}, error) {
	var assoc ZX30000Assoc
	rets := make(map[string]struct{})
	secs, _ := uci.GetSections("wireless", "wifi-iface")
	for _, sec := range secs {
		disabled, _ := uci.GetLast("wireless", sec, "disabled")
		if disabled == "1" {
			continue
		}
		ifname, _ := uci.GetLast("wireless", sec, "ifname")
		err := UbusCallWithObject(ctx, fmt.Sprintf("iwinfo assoclist {\"device\":\"%s\"}", ifname), &assoc)
		if err != nil {
			continue
		}
		for _, mac := range assoc.Results {
			rets[strings.ToUpper(mac.Mac)] = struct{}{}
		}
	}
	return rets, nil
}

func init() {
	wifiSel.register(func(old BaseWifi) BaseWifi {
		if val, ok := uci.GetLast("wireless", "radio0", "type"); !ok || val != "mt7981" {
			return nil
		}
		if old != nil {
			return old
		}
		return NewZX30000Wifi()
	})
}

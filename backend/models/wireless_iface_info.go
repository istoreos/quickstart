package models

// swagger:model wirelessIfaceInfo
type WirelessIfaceInfo struct {

	// 2g, 5g
	Band string `json:"band,omitempty"`

	// 信道 2g: [0-1-13], 5g: [0,36,40,44,48,52(DFS),56(DFS),60(DFS),64(DFS),149,153,157,161]
	Channel int64 `json:"channel,omitempty"`

	// 非显示，调用 set-device-power 的时候用到
	Device string `json:"device,omitempty"`

	// enable or not?
	Disabled bool `json:"disabled,omitempty"`

	// 加密方式: [OPEN WPA2-PSK WPA/WPA2-PSK WPA3-SAE WPA2-PSK/WPA3-SEA]
	EncryptSelects []string `json:"encryptSelects"`

	// Wi-Fi 安全性 [none, psk2, psk-mixed, sae, sae-mixed] ==> [OPEN, WPA2-PSK, WPA/WPA2-PSK, WPA3-SAE, WPA2-PSK/WPA3-SEA]
	Encryption string `json:"encryption,omitempty"`

	// 是否隐藏 SSID
	Hidden bool `json:"hidden,omitempty"`

	// 频宽 2g: [20, 40, auto] => [20 MHz, 40 MHz, 20/40 MHz], 5g: [20, 40, 80, 160] => [20 MHz, 40 MHz, 80MHz, 160 MHz]
	Htmode string `json:"htmode,omitempty"`

	// 无线模式选择，根据 hwmodeSelects 来选择
	Hwmode string `json:"hwmode,omitempty"`

	// 无线模式 2g: [11n/ax 11g/n/ax 11b/g/b/ax 11b/g/n], 5g: [11ac/ax 11n/ac/ax 11a/n/ac/ax 11a/n/ac]
	HwmodeSelects []string `json:"hwmodeSelects"`

	// 如果是0则不用写。如果 band=2g,ifaceIndex=2，isGuest=false, 则菜单标题是： 5G Wi-Fi [2]
	IfaceIndex int64 `json:"ifaceIndex,omitempty"`

	// wifi2g, wifi5g, guest2g, guest5g
	IfaceName string `json:"ifaceName,omitempty"`

	// ra1 rax2 and so on
	Ifname string `json:"ifname,omitempty"`

	// is guest wifi
	IsGuest bool `json:"isGuest,omitempty"`

	// Wi-Fi 密码
	Key string `json:"key,omitempty"`

	// Wi-Fi 名称(SSID)
	Ssid string `json:"ssid,omitempty"`

	// 发射功率
	Txpower int64 `json:"txpower,omitempty"`

	// 网络类型
	Network string `json:"network,omitempty"`
}

package ubus

import (
	"encoding/json"
	"errors"
	"strconv"
)

type UbusWirelessDevice struct {
	Devices []string
}

type UbusWirelessInfoData struct {
	Phy        string
	SSID       string
	BSSID      string
	Country    string
	Mode       string
	Channel    int
	Frequency  int
	TXPower    int
	Quality    int
	Qualitymax int
	Signal     int
	Noise      int
	Bitrate    int
	Encryption UbusWirelessEncryption
	Hwmodes    []string
	Hardware   UbusWirelessInfoHardware
}

type UbusWirelessEncryption struct {
	Enabled        bool
	Wpa            []int
	Authentication []string
	Ciphers        []string
}

type UbusWirelessInfoHardware struct {
	Name string
}

type UbusWirelessScanner struct {
	Results []UbusWirelessScannerData
}

type UbusWirelessScannerData struct {
	SSID       string
	BSSID      string
	Mode       string
	Channel    int
	Signal     int
	Quality    int
	QualityMax int
	Encryption UbusWirelessEncryption
}

type UbusWirelessAssocList struct {
	Results []UbusWirelessAssocListData
}

type UbusWirelessAssocListData struct {
	Mac      string
	Signal   int
	Noise    int
	Inactive int
	Rx       UbusWirelessAssocListRate
	Tx       UbusWirelessAssocListRate
}

type UbusWirelessAssocListRate struct {
	Rate    int
	Mcs     int
	Is40Mhz bool `json:"40mhz"`
	ShortGi bool
}

type UbusWirelessFreqList struct {
	Results []UbusWirelessFreqListData
}

type UbusWirelessFreqListData struct {
	Channel    int
	Mhz        int
	Restricted bool
	Active     bool
}

type UbusTxPowerList struct {
	Results []UbusTxPowerListData
}

type UbusTxPowerListData struct {
	Dbm    int
	Mw     int
	Active bool
}

type UbusCountryList struct {
	Results []UbusCountryListData
}

type UbusCountryListData struct {
	Code    string
	Country string
	ISO3166 string
	Active  bool
}

func (u *Ubus) WirelessCountryList(id int, device string) (UbusCountryList, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusCountryList{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"iwinfo", 
				"countrylist", 
				{ 
					"device": "` + device + `"
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusCountryList{}, err
	}
	ubusData := UbusCountryList{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusCountryList{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) WirelessTxPowerList(id int, device string) (UbusTxPowerList, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusTxPowerList{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"iwinfo", 
				"txpowerlist", 
				{ 
					"device": "` + device + `"
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusTxPowerList{}, err
	}
	ubusData := UbusTxPowerList{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusTxPowerList{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) WirelessFreqList(id int, device string) (UbusWirelessFreqList, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusWirelessFreqList{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"iwinfo", 
				"freqlist", 
				{ 
					"device": "` + device + `"
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusWirelessFreqList{}, err
	}
	ubusData := UbusWirelessFreqList{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusWirelessFreqList{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) WirelessAssocList(id int, device string, mac string) (UbusWirelessAssocList, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusWirelessAssocList{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"iwinfo", 
				"assoclist", 
				{ 
					"device": "` + device + `",
					"mac": "` + mac + `"
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusWirelessAssocList{}, err
	}
	ubusData := UbusWirelessAssocList{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusWirelessAssocList{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) WirelessScanner(id int, device string) (UbusWirelessScanner, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusWirelessScanner{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"iwinfo", 
				"scan", 
				{ 
					"device": "` + device + `"
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusWirelessScanner{}, err
	}
	ubusData := UbusWirelessScanner{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusWirelessScanner{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) WirelessInfo(id int, device string) (UbusWirelessInfoData, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusWirelessInfoData{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"iwinfo", 
				"info", 
				{ 
					"device": "` + device + `"
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusWirelessInfoData{}, err
	}
	ubusData := UbusWirelessInfoData{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusWirelessInfoData{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) WirelessDevices(id int) (UbusWirelessDevice, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusWirelessDevice{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"iwinfo", 
				"devices",
				{}
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusWirelessDevice{}, err
	}
	ubusData := UbusWirelessDevice{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusWirelessDevice{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

package ubus

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

type DHCPIPv4Lease struct {
	Hostname  string
	IPv4      string
	MAC       string
	Leasetime time.Time
	ClientID  string
}

type DHCPIPv4Leases []DHCPIPv4Lease

type DHCPIPv6Leases struct {
	Devices []DHCPIPv6Device
}

type DHCPIPv6Device struct {
	Device string
	Leases []DHCPIPv6Lease
}

type DHCPIPv6Lease struct {
	DUID         string
	IAID         int
	Hostname     string
	AcceptReconf bool `json:"accept-reconf"`
	Assigned     int
	Flags        []string
	IPv6Addr     []DHCPIPv6Address `json:"ipv6-addr"`
	Valid        int
}

type DHCPIPv6Address struct {
	Address           string
	PreferredLifetime int `json:"preferred-lifetime"`
	ValidLifetime     int `json:"valid-lifetime"`
}

func (u *Ubus) DHCPIPv4Leases(id int) (DHCPIPv4Leases, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return DHCPIPv4Leases{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"file", 
				"exec", 
				{ 
					"command": "cat",
					"params": ['/tmp/dhcp.leases']
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		if strings.Compare(err.Error(), "Object not found") == 0 {
			return DHCPIPv4Leases{}, errors.New("File module not found, try 'opkg update && opkg install rpcd-mod-file && service rpcd restart'")
		}
		return DHCPIPv4Leases{}, err
	}
	DHCPData := DHCPIPv4Leases{}
	ubusData := UbusExec{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return DHCPIPv4Leases{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	for _, line := range strings.Split(ubusData.Stdout, "\n") {
		if len(line) > 5 {
			data := strings.Split(line, " ")
			timedata, _ := strconv.Atoi(data[0])
			DHCPData = append(DHCPData, DHCPIPv4Lease{
				Leasetime: time.Unix(int64(timedata), 0),
				MAC:       data[1],
				IPv4:      data[2],
				Hostname:  data[3],
				ClientID:  data[4],
			})
		}
	}
	return DHCPData, nil
}

func (u *Ubus) DHCPIPv6Leases(id int) (DHCPIPv6Leases, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return DHCPIPv6Leases{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"dhcp", 
				"ipv6leases", 
				{} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return DHCPIPv6Leases{}, err
	}
	ubusData := DHCPIPv6Leases{}
	if err != nil {
		return DHCPIPv6Leases{}, errors.New("Data error")
	}
	var ubusDataDevices DHCPIPv6Device
	for deviceName, val := range call.Result.([]interface{})[1].(map[string]interface{})["device"].(map[string]interface{}) {
		ubusDataDevices = DHCPIPv6Device{}
		// fmt.Println(val.(map[string]interface{}))
		for _, lease := range val.(map[string]interface{})["leases"].([]interface{}) {
			ubusDataLease := DHCPIPv6Lease{}
			ubusDataByte, err := json.Marshal(lease)
			if err != nil {
				return DHCPIPv6Leases{}, errors.New("Error Parsing leases")
			}
			json.Unmarshal(ubusDataByte, &ubusDataLease)
			ubusDataDevices.Leases = append(ubusDataDevices.Leases, ubusDataLease)
		}
		ubusDataDevices.Device = string(deviceName)
		ubusData.Devices = append(ubusData.Devices, ubusDataDevices)
	}
	return ubusData, nil
}

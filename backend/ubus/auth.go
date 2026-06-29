package ubus

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

//UbusAuthData represents the Data response from auth module
type UbusAuthData struct {
	UbusRPCSession string `json:"ubus_rpc_session"`
	Timeout        int
	Expires        int
	ExpireTime     time.Time
	ACLs           UbusAuthACLS `json:"acls"`
	Data           map[string]string
}

//UbusAuthACLS represents the ACL from user on Authentication
type UbusAuthACLS struct {
	AccessGroup map[string][]string `json:"access-group"`
	Ubus        map[string][]string
	Uci         map[string][]string
}

//AuthLogin Call JSON-RPC method to Router Authentication
func (u *Ubus) AuthLogin(username, password string) (UbusResponse, error) {
	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": 1, 
			"method": "call", 
			"params": [ 
				"00000000000000000000000000000000", 
				"session", 
				"login", 
				{ 
					"username": "` + username + `", 
					"password": "` + password + `"  
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		if strings.Compare(err.Error(), "404 Not Found") == 0 {
			return UbusResponse{}, errors.New("Ubus module not installed, try 'opkg update && opkg install uhttpd-mod-ubus && service uhttpd restart'")
		}

		return UbusResponse{}, err
	}
	ubusData := UbusAuthData{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusResponse{}, errors.New("Error Parsing Login Data")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	//Set AuthData to Ubus struct
	u.AuthData = ubusData
	//Editing expire to current time + expire time to check in functions if the session has expired
	u.AuthData.ExpireTime = time.Now().Add(time.Second * time.Duration(u.AuthData.Expires))
	return call, nil
}

package ubus

import (
	"encoding/json"
	"errors"
	"strconv"
)

type UbusUciConfigs struct {
	Configs []string
}

type UbusUciRequest struct {
	Config  string            `json:"config"`
	Section string            `json:"section,omitempty"`
	Option  string            `json:"option,omitempty"`
	Type    string            `json:"type,omitempty"`
	Match   string            `json:"match,omitempty"`
	Values  map[string]string `json:"values,omitempty"`
}

type UbusUciResponse struct {
	Values interface{}
}

func (u *Ubus) UciGetConfigs(id int) (UbusUciConfigs, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusUciConfigs{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"uci", 
				"configs", 
				{} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusUciConfigs{}, err
	}
	ubusData := UbusUciConfigs{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusUciConfigs{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) UciGetConfig(id int, request UbusUciRequest) (UbusUciResponse, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusUciResponse{}, errLogin
	} */

	jsonData, err := json.Marshal(request)
	if err != nil {
		return UbusUciResponse{}, errors.New("Error Parsing UCI Request Data")
	}
	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"uci", 
				"get", 
				` + string(jsonData) + ` 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusUciResponse{}, err
	}
	ubusData := UbusUciResponse{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusUciResponse{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) UciSetConfig(id int, request UbusUciRequest) error {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	} */

	jsonData, err := json.Marshal(request)
	if err != nil {
		return errors.New("Error Parsing UCI Request Data")
	}
	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"uci", 
				"set", 
				` + string(jsonData) + ` 
			] 
		}`)
	_, err = u.Call(jsonStr)
	if err != nil {
		return err
	}
	return nil
}

func (u *Ubus) UciChanges(id int) (map[string]map[string][][]string, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return nil, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"uci", 
				"changes", 
				{}
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return nil, err
	}
	// fmt.Println(call)
	var ubusData map[string]map[string][][]string
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return nil, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}

func (u *Ubus) UciCommit(id int) error {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"uci", 
				"commit", 
				{}
			] 
		}`)
	_, err := u.Call(jsonStr)
	if err != nil {
		return err
	}
	return nil
}

func (u *Ubus) UciReloadConfig(id int) error {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"uci", 
				"reload_config",
				{}
			] 
		}`)
	_, err := u.Call(jsonStr)
	if err != nil {
		return err
	}
	return nil
}

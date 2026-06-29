package ubus

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

//UbusResponseCode represent the status code from JSON-RPC Call
//go:generate stringer -type=UbusResponseCode
type UbusResponseCode int

//Represents enum ubus_msg_status from https://git.openwrt.org/?p=project/ubus.git;a=blob;f=ubusmsg.h;h=398b126b6dc01833937749a110181ea0debb1476;hb=HEAD
const (
	UbusStatusOK               UbusResponseCode = 0
	UbusStatusInvalidCommand   UbusResponseCode = 1
	UbusStatusInvalidArgument  UbusResponseCode = 2
	UbusStatusMethodNotFound   UbusResponseCode = 3
	UbusStatusNotFound         UbusResponseCode = 4
	UbusStatusNoData           UbusResponseCode = 5
	UbusStatusPermissionDenied UbusResponseCode = 6
	UbusStatusTimeout          UbusResponseCode = 7
	UbusStatusNotSupported     UbusResponseCode = 8
	UbusStatusUnknownError     UbusResponseCode = 9
	UbusStatusConnectionFailed UbusResponseCode = 10
	UbusStatusLast             UbusResponseCode = 11
)

//Ubus represents information to JSON-RPC Interaction with router
type Ubus struct {
	URL      string
	AuthData UbusAuthData
	client   *http.Client
}

//UbusResponse represents a response from JSON-RPC
type UbusResponse struct {
	JSONRPC          string
	ID               int
	Error            UbusResponseError
	Result           interface{}
	UbusResponseCode UbusResponseCode
}

type UbusResponseError struct {
	Code    int
	Message string
}

type UbusExec struct {
	Code   int
	Stdout string
}

func NewUbusClient(unixSock string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.Dial("unix", unixSock)
			},
		},
	}
}

func NewUbus(sid string, client *http.Client) (*Ubus, error) {
	return &Ubus{
		URL:    "http://127.0.0.1",
		client: client,
	}, nil
}

//LoginCheck check if login RPC Session id has expired
func (u *Ubus) LoginCheck(username, password string) error {
	var err error
	var i uint8
	for start := time.Now(); time.Since(start) < 3*time.Second; {
		if u.AuthData.ExpireTime.Before(time.Now()) {
			_, err = u.AuthLogin(username, password)
			if err == nil {
				break
			}
		} else {
			break
		}
		i++
		time.Sleep(time.Second)
	}
	if i == 3 {
		return err
	}
	return nil
}

func (u *Ubus) SessionGet(sid string) error {
	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": 1, 
			"method": "call", 
			"params": [ 
				"00000000000000000000000000000000", 
				"session", 
				"get", 
				[
					` + sid + ` 
				] 
			] 
		}`)

	resp, err := u.Call(jsonStr)
	if err != nil {
		return err
	}

	data, _ := json.MarshalIndent(resp, "", "\t")
	fmt.Printf(string(data))
	return nil
}

//Call do a call to Json-RPC to get/set information
func (u *Ubus) Call(jsonStr []byte) (UbusResponse, error) {
	req, err := http.NewRequest("POST", u.URL, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	resp, err := u.client.Do(req)
	if err != nil {
		return UbusResponse{}, err
	}
	defer resp.Body.Close()
	if resp.Status != "200 OK" {
		return UbusResponse{}, fmt.Errorf("Error %s on (%s)", resp.Status, u.URL)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	result := UbusResponse{}
	json.Unmarshal([]byte(body), &result)
	//Function Error
	if result.Error.Code != 0 {
		if strings.Compare(result.Error.Message, "Access denied") == 0 {
			return UbusResponse{}, errors.New("Access denied for this instance, read https://openwrt.org/docs/techref/ubus#acls ")
		}
		return UbusResponse{}, errors.New(result.Error.Message)
	}
	//Workaround cause response code not contempled by unmarshal function
	result.UbusResponseCode = UbusResponseCode(result.Result.([]interface{})[0].(float64))
	//Workaround to get UbusData cause the structure of this array has a problem with unmarshal
	if result.UbusResponseCode == UbusStatusOK {
		return result, nil
	}
	return UbusResponse{}, fmt.Errorf("Ubus Status Failed: %s", result.UbusResponseCode)
}

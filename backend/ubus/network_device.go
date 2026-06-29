package ubus

import (
	"encoding/json"
	"errors"
	"strconv"
)

type UbusNetworkDevice struct {
	Acceptlocal         bool                       `json:"acceptlocal"`
	Carrier             bool                       `json:"carrier"`
	Dadtransmits        int                        `json:"dadtransmits"`
	External            bool                       `json:"external"`
	Igmpversion         int                        `json:"igmpversion"`
	Ipv6                bool                       `json:"ipv6"`
	Macaddr             string                     `json:"macaddr"`
	Mldversion          int                        `json:"mldversion"`
	Mtu                 int                        `json:"mtu"`
	Mtu6                int                        `json:"mtu6"`
	Multicast           bool                       `json:"multicast"`
	Neigh4gcstaletime   int                        `json:"neigh4gcstaletime"`
	Neigh4locktime      int                        `json:"neigh4locktime"`
	Neigh4reachabletime int                        `json:"neigh4reachabletime"`
	Neigh6gcstaletime   int                        `json:"neigh6gcstabletime"`
	Neigh6reachabletime int                        `json:"neigh6reachabletime"`
	Present             bool                       `json:"present"`
	Promisc             bool                       `json:"promisc"`
	Rpfilter            int                        `json:"rpfilter"`
	Sendredirects       bool                       `json:"sendredirects"`
	Statistics          UbusNetworkDeviceStatistic `json:"statistics"`
	Txqueuelen          int                        `json:"txqueuelen"`
	Type                string                     `json:"type"`
	Up                  bool                       `json:"up"`
}

type UbusNetworkDeviceStatistic struct {
	Collisions       int `json:"collisions"`
	RxFrameErrors    int `json:"rx_frame_errors"`
	TxCompressed     int `json:"tx_compressed"`
	Multicast        int `json:"multicast"`
	RxLengthErrors   int `json:"rx_length_errors"`
	TxDropped        int `json:"tx_dropped"`
	RxBytes          int `json:"rx_bytes"`
	RxMissedErrors   int `json:"rx_missed_errors"`
	TxErrors         int `json:"tx_errors"`
	RxCompressed     int `json:"rx_compressed"`
	RxOverErrors     int `json:"rx_over_errors"`
	TxFifoErrors     int `json:"tx_fifo_errors"`
	RxCrcErrors      int `json:"rx_crc_errors"`
	RxPackets        int `json:"rx_packets"`
	TxHeatbeatErrors int `json:"tx_heatbeat_errors"`
	RxDropped        int `json:"rx_dropped"`
	TxAbortedErrors  int `json:"tx_aborted_errors"`
	TxPackets        int `json:"tx_packets"`
	RxErrors         int `json:"rx_errors"`
	TxBytes          int `json:"tx_bytes"`
	TxWindowErrors   int `json:"tx_window_errors"`
	RxFifoErrors     int `json:"rx_fifo_errors"`
	TxCarrierErrors  int `json:"tx_carrier_errors"`
}

func (u *Ubus) NetworkDeviceStatus(id int, name string) (UbusNetworkDevice, error) {
	/* errLogin := u.LoginCheck()
	if errLogin != nil {
		return UbusNetworkDevice{}, errLogin
	} */

	var jsonStr = []byte(`
		{ 
			"jsonrpc": "2.0", 
			"id": ` + strconv.Itoa(id) + `, 
			"method": "call", 
			"params": [ 
				"` + u.AuthData.UbusRPCSession + `", 
				"network.device", 
				"status", 
				{ 
					"name": "` + name + `",
				} 
			] 
		}`)
	call, err := u.Call(jsonStr)
	if err != nil {
		return UbusNetworkDevice{}, err
	}
	ubusData := UbusNetworkDevice{}
	ubusDataByte, err := json.Marshal(call.Result.([]interface{})[1])
	if err != nil {
		return UbusNetworkDevice{}, errors.New("Data error")
	}
	json.Unmarshal(ubusDataByte, &ubusData)
	return ubusData, nil
}
